package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/docker/distribution"
	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/ocischema"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/errcode"
	v2 "github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/auth"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/storage/validation"
	"github.com/gorilla/handlers"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// These constants determine which architecture and OS to choose from a
// manifest list when falling back to a schema2 manifest.
const (
	defaultArch         = "amd64"
	defaultOS           = "linux"
	maxManifestBodySize = 4 << 20
	imageClass          = "image"
)

type storageType int

const (
	manifestSchema1        storageType = iota // 0
	manifestSchema2                           // 1
	manifestlistSchema                        // 2
	ociImageManifestSchema                    // 3
	ociImageIndexSchema                       // 4
	numStorageTypes                           // 5
)

// manifestDispatcher takes the request context and builds the
// appropriate handler for handling manifest requests.
func manifestDispatcher(ctx *Context, r *http.Request) http.Handler {
	manifestHandler := &manifestHandler{
		Context: ctx,
	}
	ref := getReference(ctx)
	dgst, err := digest.Parse(ref)
	if err != nil {
		// We just have a tag
		manifestHandler.Tag = ref
	} else {
		manifestHandler.Digest = dgst
	}

	mhandler := handlers.MethodHandler{
		"GET":  http.HandlerFunc(manifestHandler.GetManifest),
		"HEAD": http.HandlerFunc(manifestHandler.GetManifest),
	}

	if !ctx.readOnly {
		mhandler["PUT"] = http.HandlerFunc(manifestHandler.PutManifest)
		mhandler["DELETE"] = http.HandlerFunc(manifestHandler.DeleteManifest)
	}

	return migrationWrapper(ctx, mhandler)
}

// manifestHandler handles http operations on image manifests.
type manifestHandler struct {
	*Context

	// One of tag or digest gets set, depending on what is present in context.
	Tag    string
	Digest digest.Digest
}

// GetManifest fetches the image manifest from the storage backend, if it exists.
func (imh *manifestHandler) GetManifest(w http.ResponseWriter, r *http.Request) {
	dcontext.GetLogger(imh).Debug("GetImageManifest")
	manifestService, err := imh.Repository.Manifests(imh)
	if err != nil {
		imh.Errors = append(imh.Errors, err)
		return
	}
	var supports [numStorageTypes]bool

	// this parsing of Accept headers is not quite as full-featured as godoc.org's parser, but we don't care about "q=" values
	// https://github.com/golang/gddo/blob/e91d4165076d7474d20abda83f92d15c7ebc3e81/httputil/header/header.go#L165-L202
	for _, acceptHeader := range r.Header["Accept"] {
		// r.Header[...] is a slice in case the request contains the same header more than once
		// if the header isn't set, we'll get the zero value, which "range" will handle gracefully

		// we need to split each header value on "," to get the full list of "Accept" values (per RFC 2616)
		// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.1
		for _, mediaType := range strings.Split(acceptHeader, ",") {
			if mediaType, _, err = mime.ParseMediaType(mediaType); err != nil {
				continue
			}

			if mediaType == schema2.MediaTypeManifest {
				supports[manifestSchema2] = true
			}
			if mediaType == manifestlist.MediaTypeManifestList {
				supports[manifestlistSchema] = true
			}
			if mediaType == v1.MediaTypeImageManifest {
				supports[ociImageManifestSchema] = true
			}
			if mediaType == v1.MediaTypeImageIndex {
				supports[ociImageIndexSchema] = true
			}
		}
	}

	var manifest distribution.Manifest

	if imh.Tag != "" {
		var dgst digest.Digest
		var dbErr error

		if imh.Config.Database.Enabled {
			manifest, dgst, dbErr = dbGetManifestByTag(imh.Context, imh.App.db, imh.Tag, imh.Repository.Named().Name())
			if dbErr != nil {
				// Use the common error handling code below.
				err = dbErr
			}
		}

		if !imh.Config.Database.Enabled || dbErr != nil {
			var desc distribution.Descriptor

			tags := imh.Repository.Tags(imh)
			desc, err = tags.Get(imh, imh.Tag)
			dgst = desc.Digest
		}

		if err != nil {
			if errors.As(err, &distribution.ErrTagUnknown{}) ||
				errors.Is(err, digest.ErrDigestInvalidFormat) ||
				errors.As(err, &distribution.ErrManifestUnknown{}) {
				// not found or with broken current/link (invalid digest)
				imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithDetail(err))
			} else {
				imh.Errors = append(imh.Errors, errcode.FromUnknownError(err))
			}
			return
		}
		imh.Digest = dgst
	}

	if etagMatch(r, imh.Digest.String()) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// The manifest will be nil if we retrieved the tag from the filesystem or
	// the manifest is being referenced by digest.
	if manifest == nil {
		manifest, err = dbGetManifestFilesystemFallback(imh.Context, imh.App.db, manifestService, imh.Digest, imh.Tag, imh.Repository.Named().Name(), imh.Config.Database.Enabled)
		if err != nil {
			switch {
			case errors.As(err, &distribution.ErrManifestUnknownRevision{}):
				imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithDetail(err))
			case errors.Is(err, distribution.ErrSchemaV1Unsupported):
				imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithMessage("Schema 1 manifest not supported"))
			default:
				imh.Errors = append(imh.Errors, errcode.FromUnknownError(err))
			}
			return
		}
	}

	// determine the type of the returned manifest
	manifestType := manifestSchema1
	_, isSchema2 := manifest.(*schema2.DeserializedManifest)
	manifestList, isManifestList := manifest.(*manifestlist.DeserializedManifestList)
	if isSchema2 {
		manifestType = manifestSchema2
	} else if _, isOCImanifest := manifest.(*ocischema.DeserializedManifest); isOCImanifest {
		manifestType = ociImageManifestSchema
	} else if isManifestList {
		if manifestList.MediaType == manifestlist.MediaTypeManifestList {
			manifestType = manifestlistSchema
		} else if manifestList.MediaType == v1.MediaTypeImageIndex {
			manifestType = ociImageIndexSchema
		}
	}

	if manifestType == manifestSchema1 {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithMessage("Schema 1 manifest not supported"))
		return
	}
	if manifestType == ociImageManifestSchema && !supports[ociImageManifestSchema] {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithMessage("OCI manifest found, but accept header does not support OCI manifests"))
		return
	}
	if manifestType == ociImageIndexSchema && !supports[ociImageIndexSchema] {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithMessage("OCI index found, but accept header does not support OCI indexes"))
		return
	}

	// Only rewrite manifests lists when they are being fetched by tag. If they
	// are being fetched by digest, we can't return something not matching the digest.
	if imh.Tag != "" && manifestType == manifestlistSchema && !supports[manifestlistSchema] {
		log := dcontext.GetLoggerWithFields(imh, map[interface{}]interface{}{
			"manifest_list_digest": imh.Digest.String(),
			"default_arch":         defaultArch,
			"default_os":           defaultOS})
		log.Info("client does not advertise support for manifest lists, selecting a manifest image for the default arch and os")

		// Find the image manifest corresponding to the default platform.
		var manifestDigest digest.Digest
		for _, manifestDescriptor := range manifestList.Manifests {
			if manifestDescriptor.Platform.Architecture == defaultArch && manifestDescriptor.Platform.OS == defaultOS {
				manifestDigest = manifestDescriptor.Digest
				break
			}
		}

		if manifestDigest == "" {
			imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithDetail(
				fmt.Errorf("manifest list %s does not contain a manifest image for the platform %s/%s",
					imh.Digest, defaultOS, defaultArch)))
			return
		}

		manifest, err = dbGetManifestFilesystemFallback(imh.Context, imh.App.db, manifestService, manifestDigest, "", imh.Repository.Named().Name(), imh.Config.Database.Enabled)
		if err != nil {
			if _, ok := err.(distribution.ErrManifestUnknownRevision); ok {
				imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithDetail(err))
			} else {
				imh.Errors = append(imh.Errors, errcode.FromUnknownError(err))
			}
			return
		}

		imh.Digest = manifestDigest
	}

	ct, p, err := manifest.Payload()
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Length", fmt.Sprint(len(p)))
	w.Header().Set("Docker-Content-Digest", imh.Digest.String())
	w.Header().Set("Etag", fmt.Sprintf(`"%s"`, imh.Digest))
	w.Write(p)
}

// dbGetManifestFilesystemFallback returns a distribution manifest by digest
// for the given repository. Reads from the database if enabled, otherwise the
// manifest will be retrieved from the filesytem.
func dbGetManifestFilesystemFallback(
	ctx context.Context,
	db datastore.Queryer,
	fsManifests distribution.ManifestService,
	dgst digest.Digest,
	tag, path string,
	dbEnabled bool) (distribution.Manifest, error) {
	if dbEnabled {
		return dbGetManifest(ctx, db, dgst, path)
	}

	var options []distribution.ManifestServiceOption
	if tag != "" {
		options = append(options, distribution.WithTag(tag))
	}

	return fsManifests.Get(ctx, dgst, options...)
}

func dbGetManifest(ctx context.Context, db datastore.Queryer, dgst digest.Digest, path string) (distribution.Manifest, error) {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": path, "digest": dgst})
	log.Debug("getting manifest by digest from database")

	repositoryStore := datastore.NewRepositoryStore(db)
	r, err := repositoryStore.FindByPath(ctx, path)
	if err != nil {
		return nil, err
	}
	if r == nil {
		log.Warn("repository not found in database")
		return nil, distribution.ErrManifestUnknownRevision{
			Name:     path,
			Revision: dgst,
		}
	}

	// Find manifest by its digest
	dbManifest, err := repositoryStore.FindManifestByDigest(ctx, r, dgst)
	if err != nil {
		return nil, err
	}
	if dbManifest == nil {
		return nil, distribution.ErrManifestUnknownRevision{
			Name:     path,
			Revision: dgst,
		}
	}

	return dbPayloadToManifest(dbManifest.Payload, dbManifest.MediaType, dbManifest.SchemaVersion)
}

func dbGetManifestByTag(ctx context.Context, db datastore.Queryer, tagName string, path string) (distribution.Manifest, digest.Digest, error) {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": path, "tag": tagName})
	log.Debug("getting manifest by tag from database")

	repositoryStore := datastore.NewRepositoryStore(db)
	r, err := repositoryStore.FindByPath(ctx, path)
	if err != nil {
		return nil, "", err
	}
	if r == nil {
		log.Warn("repository not found in database")
		return nil, "", distribution.ErrTagUnknown{Tag: tagName}
	}

	dbManifest, err := repositoryStore.FindManifestByTagName(ctx, r, tagName)
	if err != nil {
		return nil, "", err
	}
	// at the DB level a tag has a FK to manifests, so a tag cannot exist unless it points to an existing manifest
	if dbManifest == nil {
		log.Warn("tag not found in database")
		return nil, "", distribution.ErrTagUnknown{Tag: tagName}
	}

	manifest, err := dbPayloadToManifest(dbManifest.Payload, dbManifest.MediaType, dbManifest.SchemaVersion)
	if err != nil {
		return nil, "", err
	}

	return manifest, dbManifest.Digest, nil
}

func dbPayloadToManifest(payload []byte, mediaType string, schemaVersion int) (distribution.Manifest, error) {
	if schemaVersion == 1 {
		return nil, distribution.ErrSchemaV1Unsupported
	}

	if schemaVersion != 2 {
		return nil, fmt.Errorf("unrecognized manifest schema version %d", schemaVersion)
	}

	// TODO: Each case here is taken directly from the respective
	// registry/storage/*manifesthandler Unmarshal method. We cannot invoke them
	// directly as they are unexported, but they are relatively simple. We should
	// determine a single place for this logic during refactoring
	// https://gitlab.com/gitlab-org/container-registry/-/issues/135

	// This can be an image manifest or a manifest list
	switch mediaType {
	case schema2.MediaTypeManifest:
		m := &schema2.DeserializedManifest{}
		if err := m.UnmarshalJSON(payload); err != nil {
			return nil, err
		}

		return m, nil
	case v1.MediaTypeImageManifest:
		m := &ocischema.DeserializedManifest{}
		if err := m.UnmarshalJSON(payload); err != nil {
			return nil, err
		}

		return m, nil
	case manifestlist.MediaTypeManifestList, v1.MediaTypeImageIndex:
		m := &manifestlist.DeserializedManifestList{}
		if err := m.UnmarshalJSON(payload); err != nil {
			return nil, err
		}

		return m, nil
	case "":
		// OCI image or image index - no media type in the content

		// First see if it looks like an image index
		resIndex := &manifestlist.DeserializedManifestList{}
		if err := resIndex.UnmarshalJSON(payload); err != nil {
			return nil, err
		}
		if resIndex.Manifests != nil {
			return resIndex, nil
		}

		// Otherwise, assume it must be an image manifest
		m := &ocischema.DeserializedManifest{}
		if err := m.UnmarshalJSON(payload); err != nil {
			return nil, err
		}

		return m, nil
	default:
		return nil, distribution.ErrManifestVerification{fmt.Errorf("unrecognized manifest content type %s", mediaType)}
	}
}

func etagMatch(r *http.Request, etag string) bool {
	for _, headerVal := range r.Header["If-None-Match"] {
		if headerVal == etag || headerVal == fmt.Sprintf(`"%s"`, etag) { // allow quoted or unquoted
			return true
		}
	}
	return false
}

// PutManifest validates and stores a manifest in the registry.
func (imh *manifestHandler) PutManifest(w http.ResponseWriter, r *http.Request) {
	log := dcontext.GetLogger(imh)
	log.Debug("PutImageManifest")
	manifests, err := imh.Repository.Manifests(imh)
	if err != nil {
		imh.Errors = append(imh.Errors, err)
		return
	}

	var jsonBuf bytes.Buffer
	if err := copyFullPayload(imh, w, r, &jsonBuf, maxManifestBodySize, "image manifest PUT"); err != nil {
		// copyFullPayload reports the error if necessary
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithDetail(err.Error()))
		return
	}

	mediaType := r.Header.Get("Content-Type")
	manifest, desc, err := distribution.UnmarshalManifest(mediaType, jsonBuf.Bytes())
	if err != nil {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithDetail(err))
		return
	}

	if imh.Digest != "" {
		if desc.Digest != imh.Digest {
			log.Errorf("payload digest does match: %q != %q", desc.Digest, imh.Digest)
			imh.Errors = append(imh.Errors, v2.ErrorCodeDigestInvalid)
			return
		}
	} else if imh.Tag != "" {
		imh.Digest = desc.Digest
	} else {
		imh.Errors = append(imh.Errors, v2.ErrorCodeTagInvalid.WithDetail("no tag or digest specified"))
		return
	}

	isAnOCIManifest := mediaType == v1.MediaTypeImageManifest || mediaType == v1.MediaTypeImageIndex

	if isAnOCIManifest {
		log.Debug("Putting an OCI Manifest!")
	} else {
		log.Debug("Putting a Docker Manifest!")
	}

	var options []distribution.ManifestServiceOption
	if imh.Tag != "" {
		options = append(options, distribution.WithTag(imh.Tag))
	}

	if err := imh.applyResourcePolicy(manifest); err != nil {
		imh.Errors = append(imh.Errors, err)
		return
	}

	if !imh.App.Config.Migration.DisableMirrorFS {
		_, err = manifests.Put(imh, manifest, options...)
		if err != nil {
			imh.appendPutError(err)
			return
		}
	}

	if imh.Config.Database.Enabled {
		// We're using the database and mirroring writes to the filesystem. We'll run
		// a transaction so we can revert any changes to the database in case that
		// any part of this multi-phase database operation fails.
		tx, err := imh.App.db.BeginTx(imh.Context, nil)
		if err != nil {
			imh.Errors = append(imh.Errors,
				errcode.FromUnknownError(fmt.Errorf("failed to create database transaction: %w", err)))
			return
		}
		defer tx.Rollback()

		if err := dbPutManifest(imh, manifest, jsonBuf.Bytes()); err != nil {
			imh.appendPutError(err)
			return
		}

		if err := tx.Commit(); err != nil {
			imh.Errors = append(imh.Errors,
				errcode.FromUnknownError(fmt.Errorf("failed to commit manifest to database: %w", err)))
			return
		}
	}

	// Tag this manifest
	if imh.Tag != "" {
		if !imh.App.Config.Migration.DisableMirrorFS {
			tags := imh.Repository.Tags(imh)
			err = tags.Tag(imh, imh.Tag, desc)
			if err != nil {
				imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
				return
			}
		}

		// Associate tag with manifest in database.
		if imh.Config.Database.Enabled {
			tx, err := imh.App.db.BeginTx(imh.Context, nil)
			if err != nil {
				e := fmt.Errorf("failed to create database transaction: %w", err)
				imh.Errors = append(imh.Errors, errcode.FromUnknownError(e))
				return
			}
			defer tx.Rollback()

			if err := dbTagManifest(imh, tx, imh.Digest, imh.Tag, imh.Repository.Named().Name()); err != nil {
				e := fmt.Errorf("failed to create tag in database: %w", err)
				imh.Errors = append(imh.Errors, errcode.FromUnknownError(e))
				return
			}
			if err := tx.Commit(); err != nil {
				e := fmt.Errorf("failed to commit tag to database: %v", err)
				imh.Errors = append(imh.Errors, errcode.FromUnknownError(e))
				return
			}
		}
	}

	// Construct a canonical url for the uploaded manifest.
	ref, err := reference.WithDigest(imh.Repository.Named(), imh.Digest)
	if err != nil {
		imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	location, err := imh.urlBuilder.BuildManifestURL(ref)
	if err != nil {
		// NOTE(stevvooe): Given the behavior above, this absurdly unlikely to
		// happen. We'll log the error here but proceed as if it worked. Worst
		// case, we set an empty location header.
		log.Errorf("error building manifest url from digest: %v", err)
	}

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", imh.Digest.String())
	w.WriteHeader(http.StatusCreated)

	log.WithFields(logrus.Fields{
		"media_type": desc.MediaType,
		"size_bytes": desc.Size,
		"digest":     desc.Digest,
		"tag":        imh.Tag,
	}).Info("manifest uploaded")
}

func (imh *manifestHandler) appendPutError(err error) {
	if errors.Is(err, distribution.ErrUnsupported) {
		imh.Errors = append(imh.Errors, errcode.ErrorCodeUnsupported)
		return
	}
	if errors.Is(err, distribution.ErrAccessDenied) {
		imh.Errors = append(imh.Errors, errcode.ErrorCodeDenied)
		return
	}
	if errors.Is(err, distribution.ErrSchemaV1Unsupported) {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithDetail("manifest type unsupported"))
		return
	}

	switch err := err.(type) {
	case distribution.ErrManifestVerification:
		for _, verificationError := range err {
			switch verificationError := verificationError.(type) {
			case distribution.ErrManifestBlobUnknown:
				imh.Errors = append(imh.Errors, v2.ErrorCodeManifestBlobUnknown.WithDetail(verificationError.Digest))
			case distribution.ErrManifestNameInvalid:
				imh.Errors = append(imh.Errors, v2.ErrorCodeNameInvalid.WithDetail(err))
			case distribution.ErrManifestUnverified:
				imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnverified)
			default:
				if verificationError == digest.ErrDigestInvalidFormat {
					imh.Errors = append(imh.Errors, v2.ErrorCodeDigestInvalid)
				} else {
					imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown, verificationError)
				}
			}
		}
	case errcode.Error:
		imh.Errors = append(imh.Errors, err)
	default:
		imh.Errors = append(imh.Errors, errcode.FromUnknownError(err))
	}
}

func dbPutManifest(imh *manifestHandler, manifest distribution.Manifest, payload []byte) error {
	switch reqManifest := manifest.(type) {
	case *schema2.DeserializedManifest:
		return dbPutManifestSchema2(imh, reqManifest, payload)
	case *ocischema.DeserializedManifest:
		return dbPutManifestOCI(imh, reqManifest, payload)
	case *manifestlist.DeserializedManifestList:
		return dbPutManifestList(imh, reqManifest, payload)
	default:
		return v2.ErrorCodeManifestInvalid.WithDetail("manifest type unsupported")
	}
}

func dbTagManifest(ctx context.Context, db datastore.Queryer, dgst digest.Digest, tagName, path string) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": path, "manifest_digest": dgst, "tag": tagName})
	log.Debug("tagging manifest")

	repositoryStore := datastore.NewRepositoryStore(db)
	dbRepo, err := repositoryStore.FindByPath(ctx, path)
	if err != nil {
		return err
	}

	// TODO: If we return the manifest ID from the putDatabase methods, we can
	// avoid looking up the manifest by digest.
	dbManifest, err := repositoryStore.FindManifestByDigest(ctx, dbRepo, dgst)
	if err != nil {
		return err
	}
	if dbManifest == nil {
		return fmt.Errorf("manifest %s not found in database", dgst)
	}

	tagStore := datastore.NewTagStore(db)

	log.Debug("creating tag")
	return tagStore.CreateOrUpdate(ctx, &models.Tag{
		Name:         tagName,
		RepositoryID: dbRepo.ID,
		ManifestID:   dbManifest.ID,
	})
}

func dbPutManifestOCI(imh *manifestHandler, manifest *ocischema.DeserializedManifest, payload []byte) error {
	repoReader := datastore.NewRepositoryStore(imh.db)
	repoPath := imh.Repository.Named().Name()

	v := validation.NewOCIValidator(
		&datastore.RepositoryManifestService{RepositoryReader: repoReader, RepositoryPath: repoPath},
		&datastore.RepositoryBlobService{RepositoryReader: repoReader, RepositoryPath: repoPath},
		imh.App.isCache,
		imh.App.manifestURLs,
	)

	if err := v.Validate(imh, manifest); err != nil {
		return err
	}

	return dbPutManifestOCIOrSchema2(imh, manifest.Versioned, manifest.Layers, manifest.Config, payload)
}

func dbPutManifestSchema2(imh *manifestHandler, manifest *schema2.DeserializedManifest, payload []byte) error {
	repoReader := datastore.NewRepositoryStore(imh.db)
	repoPath := imh.Repository.Named().Name()

	v := validation.NewSchema2Validator(
		&datastore.RepositoryManifestService{RepositoryReader: repoReader, RepositoryPath: repoPath},
		&datastore.RepositoryBlobService{RepositoryReader: repoReader, RepositoryPath: repoPath},
		imh.App.isCache,
		imh.App.manifestURLs,
	)

	if err := v.Validate(imh.Context, manifest); err != nil {
		return err
	}

	return dbPutManifestOCIOrSchema2(imh, manifest.Versioned, manifest.Layers, manifest.Config, payload)
}

func dbPutManifestOCIOrSchema2(imh *manifestHandler, versioned manifest.Versioned, layers []distribution.Descriptor, cfgDesc distribution.Descriptor, payload []byte) error {
	repoPath := imh.Repository.Named().Name()

	log := dcontext.GetLoggerWithFields(imh, map[interface{}]interface{}{"repository": repoPath, "manifest_digest": imh.Digest, "schema_version": versioned.SchemaVersion})
	log.Debug("putting manifest")

	// create or find target repository
	repositoryStore := datastore.NewRepositoryStore(imh.App.db)
	dbRepo, err := repositoryStore.CreateOrFindByPath(imh, repoPath)
	if err != nil {
		return err
	}

	// Find the config now to ensure that the config's blob is associated with the repository.
	dbCfgBlob, err := dbFindRepositoryBlob(imh.Context, imh.App.db, cfgDesc, dbRepo.Path)
	if err != nil {
		return err
	}
	// TODO: update the config blob media_type here, it was set to "application/octet-stream" during the upload
	// 		 but now we know its concrete type (cfgDesc.MediaType).

	// Since filesystem writes may be optional, We cannot be sure that the
	// repository scoped filesystem blob service will have a link to the
	// configuration blob; however, since we check for repository scoped access
	// via the database above, we may retrieve the blob directly common storage.
	blobService, ok := imh.App.registry.Blobs().(distribution.BlobProvider)
	if !ok {
		return fmt.Errorf("unable to convert BlobEnumerator into BlobService")
	}

	cfgPayload, err := blobService.Get(imh, dbCfgBlob.Digest)
	if err != nil {
		return err
	}

	dbManifest, err := repositoryStore.FindManifestByDigest(imh.Context, dbRepo, imh.Digest)
	if err != nil {
		return err
	}
	if dbManifest == nil {
		log.Debug("manifest not found in database")

		m := &models.Manifest{
			RepositoryID:  dbRepo.ID,
			SchemaVersion: versioned.SchemaVersion,
			MediaType:     versioned.MediaType,
			Digest:        imh.Digest,
			Payload:       payload,
			Configuration: &models.Configuration{
				MediaType: dbCfgBlob.MediaType,
				Digest:    dbCfgBlob.Digest,
				Payload:   cfgPayload,
			},
		}

		mStore := datastore.NewManifestStore(imh.App.db)
		if err := mStore.Create(imh, m); err != nil {
			return err
		}

		dbManifest = m

		// find and associate manifest layer blobs
		for _, reqLayer := range layers {
			dbBlob, err := dbFindRepositoryBlob(imh.Context, imh.App.db, reqLayer, dbRepo.Path)
			if err != nil {
				return err
			}

			// TODO: update the layer blob media_type here, it was set to "application/octet-stream" during the upload
			// 		 but now we know its concrete type (reqLayer.MediaType).

			if err := mStore.AssociateLayerBlob(imh.Context, dbManifest, dbBlob); err != nil {
				return err
			}
		}
	}

	return nil
}

// dbFindRepositoryBlob finds a blob which is linked to the repository.
func dbFindRepositoryBlob(ctx context.Context, db datastore.Queryer, desc distribution.Descriptor, repoPath string) (*models.Blob, error) {
	rStore := datastore.NewRepositoryStore(db)

	r, err := rStore.FindByPath(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("source repository not found in database")
	}

	b, err := rStore.FindBlob(ctx, r, desc.Digest)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("blob not found in database")
	}

	return b, nil
}

// dbFindManifestListManifest finds a manifest which is linked to the manifest list.
func dbFindManifestListManifest(
	ctx context.Context,
	db datastore.Queryer,
	dbRepo *models.Repository,
	dgst digest.Digest,
	repoPath string) (*models.Manifest, error) {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": repoPath, "manifest_digest": dgst})
	log.Debug("finding manifest list manifest")

	var dbManifest *models.Manifest

	rStore := datastore.NewRepositoryStore(db)
	dbManifest, err := rStore.FindManifestByDigest(ctx, dbRepo, dgst)
	if err != nil {
		return nil, err
	}
	if dbManifest == nil {
		return nil, fmt.Errorf("manifest %s not found", dgst)
	}

	return dbManifest, nil
}

func dbPutManifestList(imh *manifestHandler, manifestList *manifestlist.DeserializedManifestList, payload []byte) error {
	repoPath := imh.Repository.Named().Name()

	log := dcontext.GetLoggerWithFields(imh, map[interface{}]interface{}{"repository": repoPath, "manifest_digest": imh.Digest})
	log.Debug("putting manifest list")

	repoReader := datastore.NewRepositoryStore(imh.db)

	v := validation.NewManifestListValidator(&datastore.RepositoryManifestService{RepositoryReader: repoReader, RepositoryPath: repoPath}, imh.App.isCache)

	if err := v.Validate(imh, manifestList); err != nil {
		return err
	}

	// create or find target repository
	repositoryStore := datastore.NewRepositoryStore(imh.App.db)
	dbRepo, err := repositoryStore.CreateOrFindByPath(imh.Context, repoPath)
	if err != nil {
		return err
	}

	dbManifestList, err := repositoryStore.FindManifestByDigest(imh.Context, dbRepo, imh.Digest)
	if err != nil {
		return err
	}

	if dbManifestList == nil {
		log.Debug("manifest list not found in database")

		// Media type can be either Docker (`application/vnd.docker.distribution.manifest.list.v2+json`) or OCI (empty).
		// We need to make it explicit if empty, otherwise we're not able to distinguish between media types.
		mediaType := manifestList.MediaType
		if mediaType == "" {
			mediaType = v1.MediaTypeImageIndex
		}

		dbManifestList = &models.Manifest{
			RepositoryID:  dbRepo.ID,
			SchemaVersion: manifestList.SchemaVersion,
			MediaType:     mediaType,
			Digest:        imh.Digest,
			Payload:       payload,
		}
		mStore := datastore.NewManifestStore(imh.App.db)
		if err := mStore.Create(imh, dbManifestList); err != nil {
			return err
		}

		// Associate manifests to the manifest list.
		for _, m := range manifestList.Manifests {
			dbManifest, err := dbFindManifestListManifest(imh.Context, imh.App.db, dbRepo, m.Digest, dbRepo.Path)
			if err != nil {
				return err
			}

			if err := mStore.AssociateManifest(imh.Context, dbManifestList, dbManifest); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyResourcePolicy checks whether the resource class matches what has
// been authorized and allowed by the policy configuration.
func (imh *manifestHandler) applyResourcePolicy(manifest distribution.Manifest) error {
	allowedClasses := imh.App.Config.Policy.Repository.Classes
	if len(allowedClasses) == 0 {
		return nil
	}

	var class string
	switch m := manifest.(type) {
	case *schema2.DeserializedManifest:
		switch m.Config.MediaType {
		case schema2.MediaTypeImageConfig:
			class = imageClass
		case schema2.MediaTypePluginConfig:
			class = "plugin"
		default:
			return errcode.ErrorCodeDenied.WithMessage("unknown manifest class for " + m.Config.MediaType)
		}
	case *ocischema.DeserializedManifest:
		switch m.Config.MediaType {
		case v1.MediaTypeImageConfig:
			class = imageClass
		default:
			return errcode.ErrorCodeDenied.WithMessage("unknown manifest class for " + m.Config.MediaType)
		}
	}

	if class == "" {
		return nil
	}

	// Check to see if class is allowed in registry
	var allowedClass bool
	for _, c := range allowedClasses {
		if class == c {
			allowedClass = true
			break
		}
	}
	if !allowedClass {
		return errcode.ErrorCodeDenied.WithMessage(fmt.Sprintf("registry does not allow %s manifest", class))
	}

	resources := auth.AuthorizedResources(imh)
	n := imh.Repository.Named().Name()

	var foundResource bool
	for _, r := range resources {
		if r.Name == n {
			if r.Class == "" {
				r.Class = imageClass
			}
			if r.Class == class {
				return nil
			}
			foundResource = true
		}
	}

	// resource was found but no matching class was found
	if foundResource {
		return errcode.ErrorCodeDenied.WithMessage(fmt.Sprintf("repository not authorized for %s manifest", class))
	}

	return nil
}

// TODO: Placeholder until https://gitlab.com/gitlab-org/container-registry/-/issues/109
var errManifestNotFoundDB = errors.New("manifest not found in database")

// dbDeleteManifest replicates the DeleteManifest action in the metadata database. This method doesn't actually delete
// a manifest from the database (that's a task for GC, if a manifest is unreferenced), it only deletes the record that
// associates the manifest with a digest d with the repository with path repoPath. Any tags that reference the manifest
// within the repository are also deleted.
func dbDeleteManifest(ctx context.Context, db datastore.Queryer, repoPath string, d digest.Digest) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": repoPath, "digest": d})
	log.Debug("deleting manifest from repository in database")

	rStore := datastore.NewRepositoryStore(db)
	r, err := rStore.FindByPath(ctx, repoPath)
	if err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("repository not found in database: %w", err)
	}

	found, err := rStore.DeleteManifest(ctx, r, d)
	if err != nil {
		return err
	}
	if !found {
		return errManifestNotFoundDB
	}

	return nil
}

// DeleteManifest removes the manifest with the given digest from the registry.
func (imh *manifestHandler) DeleteManifest(w http.ResponseWriter, r *http.Request) {
	dcontext.GetLogger(imh).Debug("DeleteImageManifest")

	if !imh.App.Config.Migration.DisableMirrorFS {
		manifests, err := imh.Repository.Manifests(imh)
		if err != nil {
			imh.Errors = append(imh.Errors, err)
			return
		}

		err = manifests.Delete(imh, imh.Digest)
		if err != nil {
			imh.appendManifestDeleteError(err)
			return
		}

		tagService := imh.Repository.Tags(imh)
		referencedTags, err := tagService.Lookup(imh, distribution.Descriptor{Digest: imh.Digest})
		if err != nil {
			imh.Errors = append(imh.Errors, err)
			return
		}

		for _, tag := range referencedTags {
			if err = tagService.Untag(imh, tag); err != nil {
				imh.Errors = append(imh.Errors, err)
				return
			}
		}
	}

	if imh.App.Config.Database.Enabled {
		if !deleteEnabled(imh.App.Config) {
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnsupported)
			return
		}

		tx, err := imh.db.BeginTx(imh.Context, nil)
		if err != nil {
			e := fmt.Errorf("failed to create database transaction: %w", err)
			imh.Errors = append(imh.Errors, errcode.FromUnknownError(e))
			return
		}
		defer tx.Rollback()

		if err = dbDeleteManifest(imh.Context, tx, imh.Repository.Named().String(), imh.Digest); err != nil {
			imh.appendManifestDeleteError(err)
			return
		}

		if err = tx.Commit(); err != nil {
			e := fmt.Errorf("failed to commit database transaction: %w", err)
			imh.Errors = append(imh.Errors, errcode.FromUnknownError(e))
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (imh *manifestHandler) appendManifestDeleteError(err error) {
	switch err {
	case digest.ErrDigestUnsupported, digest.ErrDigestInvalidFormat:
		imh.Errors = append(imh.Errors, v2.ErrorCodeDigestInvalid)
		return
	case distribution.ErrBlobUnknown, errManifestNotFoundDB:
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown)
		return
	case distribution.ErrUnsupported:
		imh.Errors = append(imh.Errors, errcode.ErrorCodeUnsupported)
		return
	default:
		imh.Errors = append(imh.Errors, errcode.FromUnknownError(err))
		return
	}
}
