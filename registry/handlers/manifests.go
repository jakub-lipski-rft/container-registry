package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"

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

	manifestGetter, err := imh.newManifestGetter(r)
	if err != nil {
		imh.Errors = append(imh.Errors, err)
		return
	}

	var (
		manifest distribution.Manifest
		getErr   error
	)

	if imh.Tag != "" {
		manifest, imh.Digest, getErr = manifestGetter.GetByTag(imh.Context, imh.Tag)
	} else {
		manifest, getErr = manifestGetter.GetByDigest(imh.Context, imh.Digest)
	}
	if getErr != nil {
		switch {
		case errors.Is(getErr, errETagMatches):
			w.WriteHeader(http.StatusNotModified)
		case errors.As(getErr, &distribution.ErrManifestUnknownRevision{}),
			errors.As(getErr, &distribution.ErrManifestUnknown{}),
			errors.Is(getErr, digest.ErrDigestInvalidFormat),
			errors.As(getErr, &distribution.ErrTagUnknown{}):
			imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithDetail(getErr))
		case errors.Is(getErr, distribution.ErrSchemaV1Unsupported):
			imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithMessage("Schema 1 manifest not supported"))
		default:
			imh.Errors = append(imh.Errors, errcode.FromUnknownError(getErr))
		}
		return
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
	if manifestType == ociImageManifestSchema && !supports(r, ociImageManifestSchema) {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithMessage("OCI manifest found, but accept header does not support OCI manifests"))
		return
	}
	if manifestType == ociImageIndexSchema && !supports(r, ociImageIndexSchema) {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithMessage("OCI index found, but accept header does not support OCI indexes"))
		return
	}

	// Only rewrite manifests lists when they are being fetched by tag. If they
	// are being fetched by digest, we can't return something not matching the digest.
	if imh.Tag != "" && manifestType == manifestlistSchema && !supports(r, manifestlistSchema) {
		manifest, err = imh.rewriteManifestList(manifestList)
		if err != nil {
			switch err := err.(type) {
			case distribution.ErrManifestUnknownRevision:
				imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithDetail(err))
			case errcode.Error:
				imh.Errors = append(imh.Errors, err)
			default:
				imh.Errors = append(imh.Errors, errcode.FromUnknownError(err))
			}
			return
		}
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

func supports(req *http.Request, st storageType) bool {
	// this parsing of Accept headers is not quite as full-featured as godoc.org's parser, but we don't care about "q=" values
	// https://github.com/golang/gddo/blob/e91d4165076d7474d20abda83f92d15c7ebc3e81/httputil/header/header.go#L165-L202
	for _, acceptHeader := range req.Header["Accept"] {
		// r.Header[...] is a slice in case the request contains the same header more than once
		// if the header isn't set, we'll get the zero value, which "range" will handle gracefully

		// we need to split each header value on "," to get the full list of "Accept" values (per RFC 2616)
		// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.1
		for _, rawMT := range strings.Split(acceptHeader, ",") {
			mediaType, _, err := mime.ParseMediaType(rawMT)
			if err != nil {
				continue
			}

			switch st {
			// Schema2 manifests are supported by default, so there's no need to
			// confirm support for them.
			case manifestSchema2:
				return true
			case manifestlistSchema:
				if mediaType == manifestlist.MediaTypeManifestList {
					return true
				}
			case ociImageManifestSchema:
				if mediaType == v1.MediaTypeImageManifest {
					return true
				}
			case ociImageIndexSchema:
				if mediaType == v1.MediaTypeImageIndex {
					return true
				}
			}
		}
	}

	return false
}

func (imh *manifestHandler) rewriteManifestList(manifestList *manifestlist.DeserializedManifestList) (distribution.Manifest, error) {
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
		return nil, v2.ErrorCodeManifestUnknown.WithDetail(
			fmt.Errorf("manifest list %s does not contain a manifest image for the platform %s/%s",
				imh.Digest, defaultOS, defaultArch))
	}

	// TODO: We're passing an empty request here to skip etag matching logic.
	// This should be handled more cleanly.
	manifestGetter, err := imh.newManifestGetter(&http.Request{})
	if err != nil {
		return nil, err
	}

	manifest, err := manifestGetter.GetByDigest(imh.Context, manifestDigest)
	if err != nil {
		return nil, err
	}

	imh.Digest = manifestDigest

	return manifest, nil
}

var errETagMatches = errors.New("etag matches")

func (imh *manifestHandler) newManifestGetter(req *http.Request) (manifestGetter, error) {
	if imh.Config.Database.Enabled {
		return newDBManifestGetter(imh, req)
	}

	return newFSManifestGetter(imh, req)
}

type manifestGetter interface {
	GetByTag(context.Context, string) (distribution.Manifest, digest.Digest, error)
	GetByDigest(context.Context, digest.Digest) (distribution.Manifest, error)
}

type dbManifestGetter struct {
	datastore.RepositoryStore
	repoPath string
	req      *http.Request
}

func newDBManifestGetter(imh *manifestHandler, req *http.Request) (*dbManifestGetter, error) {
	return &dbManifestGetter{
		RepositoryStore: datastore.NewRepositoryStore(imh.App.db),
		repoPath:        imh.Repository.Named().Name(),
		req:             req,
	}, nil
}

func (g *dbManifestGetter) GetByTag(ctx context.Context, tagName string) (distribution.Manifest, digest.Digest, error) {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": g.repoPath, "tag": tagName})
	log.Debug("getting manifest by tag from database")

	dbRepo, err := g.FindByPath(ctx, g.repoPath)
	if err != nil {
		return nil, "", err
	}

	if dbRepo == nil {
		log.Warn("repository not found in database")
		return nil, "", distribution.ErrTagUnknown{Tag: tagName}
	}

	dbManifest, err := g.FindManifestByTagName(ctx, dbRepo, tagName)
	if err != nil {
		return nil, "", err
	}

	// at the DB level a tag has a FK to manifests, so a tag cannot exist unless it points to an existing manifest
	if dbManifest == nil {
		log.Warn("tag not found in database")
		return nil, "", distribution.ErrTagUnknown{Tag: tagName}
	}

	if etagMatch(g.req, dbManifest.Digest.String()) {
		return nil, dbManifest.Digest, errETagMatches
	}

	manifest, err := dbPayloadToManifest(dbManifest.Payload, dbManifest.MediaType, dbManifest.SchemaVersion)
	if err != nil {
		return nil, "", err
	}

	return manifest, dbManifest.Digest, nil
}

func (g *dbManifestGetter) GetByDigest(ctx context.Context, dgst digest.Digest) (distribution.Manifest, error) {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": g.repoPath, "digest": dgst})
	log.Debug("getting manifest by digest from database")

	if etagMatch(g.req, dgst.String()) {
		return nil, errETagMatches
	}

	dbRepo, err := g.FindByPath(ctx, g.repoPath)
	if err != nil {
		return nil, err
	}

	if dbRepo == nil {
		log.Warn("repository not found in database")
		return nil, distribution.ErrManifestUnknownRevision{
			Name:     g.repoPath,
			Revision: dgst,
		}
	}

	// Find manifest by its digest
	dbManifest, err := g.FindManifestByDigest(ctx, dbRepo, dgst)
	if err != nil {
		return nil, err
	}
	if dbManifest == nil {
		return nil, distribution.ErrManifestUnknownRevision{
			Name:     g.repoPath,
			Revision: dgst,
		}
	}

	return dbPayloadToManifest(dbManifest.Payload, dbManifest.MediaType, dbManifest.SchemaVersion)
}

type fsManifestGetter struct {
	ms  distribution.ManifestService
	ts  distribution.TagService
	req *http.Request
}

func newFSManifestGetter(imh *manifestHandler, r *http.Request) (*fsManifestGetter, error) {
	manifestService, err := imh.Repository.Manifests(imh)
	if err != nil {
		return nil, err
	}

	return &fsManifestGetter{
		ts:  imh.Repository.Tags(imh),
		ms:  manifestService,
		req: r,
	}, nil
}

func (g *fsManifestGetter) GetByTag(ctx context.Context, tagName string) (distribution.Manifest, digest.Digest, error) {
	desc, err := g.ts.Get(ctx, tagName)
	if err != nil {
		return nil, "", err
	}

	if etagMatch(g.req, desc.Digest.String()) {
		return nil, desc.Digest, errETagMatches
	}

	mfst, err := g.GetByDigest(ctx, desc.Digest)
	if err != nil {
		return nil, "", err
	}

	return mfst, desc.Digest, nil
}

func (g *fsManifestGetter) GetByDigest(ctx context.Context, dgst digest.Digest) (distribution.Manifest, error) {
	if etagMatch(g.req, dgst.String()) {
		return nil, errETagMatches
	}

	return g.ms.Get(ctx, dgst)
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

const (
	manifestDeleteGCReviewWindow = 1 * time.Hour
	manifestDeleteGCLockTimeout  = 5 * time.Second
)

// dbDeleteManifest replicates the DeleteManifest action in the metadata database. This method doesn't actually delete
// a manifest from the database (that's a task for GC, if a manifest is unreferenced), it only deletes the record that
// associates the manifest with a digest d with the repository with path repoPath. Any tags that reference the manifest
// within the repository are also deleted.
func dbDeleteManifest(ctx context.Context, db datastore.Handler, repoPath string, d digest.Digest) error {
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

	// We need to find the manifest first and then lookup for any manifest it references (if it's a manifest list). This
	// is needed to ensure we lock any related online GC tasks to prevent race conditions around the delete. See:
	// https://gitlab.com/gitlab-org/container-registry/-/blob/master/docs-gitlab/db/online-garbage-collection.md#deleting-the-last-referencing-manifest-list
	m, err := rStore.FindManifestByDigest(ctx, r, d)
	if err != nil {
		return err
	}
	if m == nil {
		return errManifestNotFoundDB
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create database transaction: %w", err)
	}
	defer tx.Rollback()

	switch m.MediaType {
	case manifestlist.MediaTypeManifestList, v1.MediaTypeImageIndex:
		mStore := datastore.NewManifestStore(tx)
		mm, err := mStore.References(ctx, m)
		if err != nil {
			return err
		}

		// This should never happen, as it's not possible to delete a child manifest if it's referenced by a list, which
		// means that we'll always have at least one child manifest here. Nevertheless, log error if this ever happens.
		if len(mm) == 0 {
			log.Error("stored manifest list has no references")
			break
		}
		ids := make([]int64, 0, len(mm))
		for _, m := range mm {
			ids = append(ids, m.ID)
		}

		// Prevent long running transactions by setting an upper limit of manifestDeleteGCLockTimeout. If the GC is
		// holding the lock of a related review record, the processing there should be fast enough to avoid this.
		// Regardless, we should not let transactions open (and clients waiting) for too long. If this sensible timeout
		// is exceeded, abort the manifest delete and let the client retry. This will bubble up and lead to a 503
		// Service Unavailable response.
		ctx, cancel := context.WithTimeout(ctx, manifestDeleteGCLockTimeout)
		defer cancel()

		mts := datastore.NewGCManifestTaskStore(tx)
		if _, err := mts.FindAndLockNBefore(ctx, r.ID, ids, time.Now().Add(manifestDeleteGCReviewWindow)); err != nil {
			return err
		}
	}

	rStore = datastore.NewRepositoryStore(tx)
	found, err := rStore.DeleteManifest(ctx, r, d)
	if err != nil {
		return err
	}
	if !found {
		return errManifestNotFoundDB
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit database transaction: %w", err)
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

		if err := dbDeleteManifest(imh.Context, imh.db, imh.Repository.Named().String(), imh.Digest); err != nil {
			imh.appendManifestDeleteError(err)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (imh *manifestHandler) appendManifestDeleteError(err error) {
	switch {
	case errors.Is(err, digest.ErrDigestUnsupported), errors.Is(err, digest.ErrDigestInvalidFormat):
		imh.Errors = append(imh.Errors, v2.ErrorCodeDigestInvalid)
	case errors.Is(err, distribution.ErrBlobUnknown), errors.Is(err, errManifestNotFoundDB):
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown)
	case errors.Is(err, distribution.ErrUnsupported):
		imh.Errors = append(imh.Errors, errcode.ErrorCodeUnsupported)
	case errors.Is(err, datastore.ErrManifestReferencedInList):
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestReferencedInList)
	default:
		imh.Errors = append(imh.Errors, errcode.FromUnknownError(err))
	}
}
