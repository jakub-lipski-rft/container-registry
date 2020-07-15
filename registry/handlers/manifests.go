package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/docker/distribution"
	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/ocischema"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/errcode"
	v2 "github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/auth"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/libtrust"
	"github.com/gorilla/handlers"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// These constants determine which architecture and OS to choose from a
// manifest list when downconverting it to a schema1 manifest.
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
	reference := getReference(ctx)
	dgst, err := digest.Parse(reference)
	if err != nil {
		// We just have a tag
		manifestHandler.Tag = reference
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

	return mhandler
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
	manifests, err := imh.Repository.Manifests(imh)
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

		if imh.Config.Database.Enabled {
			manifest, dgst, err = dbGetManifestByTag(imh, imh.App.db, imh.Tag, imh.App.trustKey, imh.Repository.Named().Name())
		} else {
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
				imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
			}
			return
		}
		imh.Digest = dgst
	}

	if etagMatch(r, imh.Digest.String()) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if imh.Config.Database.Enabled && imh.Tag == "" {
		manifest, err = dbGetManifest(imh, imh.App.db, imh.Digest, imh.App.trustKey, imh.Repository.Named().Name())
	} else {
		var options []distribution.ManifestServiceOption
		if imh.Tag != "" {
			options = append(options, distribution.WithTag(imh.Tag))
		}
		manifest, err = manifests.Get(imh, imh.Digest, options...)
	}
	if err != nil {
		if _, ok := err.(distribution.ErrManifestUnknownRevision); ok {
			imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithDetail(err))
		} else {
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
		}
		return
	}

	// determine the type of the returned manifest
	manifestType := manifestSchema1
	schema2Manifest, isSchema2 := manifest.(*schema2.DeserializedManifest)
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

	if manifestType == ociImageManifestSchema && !supports[ociImageManifestSchema] {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithMessage("OCI manifest found, but accept header does not support OCI manifests"))
		return
	}
	if manifestType == ociImageIndexSchema && !supports[ociImageIndexSchema] {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithMessage("OCI index found, but accept header does not support OCI indexes"))
		return
	}
	// Only rewrite schema2 manifests when they are being fetched by tag.
	// If they are being fetched by digest, we can't return something not
	// matching the digest.
	if imh.Tag != "" && manifestType == manifestSchema2 && !supports[manifestSchema2] {
		// Rewrite manifest in schema1 format
		dcontext.GetLogger(imh).Infof("rewriting manifest %s in schema1 format to support old client", imh.Digest.String())

		manifest, err = imh.convertSchema2Manifest(schema2Manifest)
		if err != nil {
			return
		}
	} else if imh.Tag != "" && manifestType == manifestlistSchema && !supports[manifestlistSchema] {
		// Rewrite manifest in schema1 format
		dcontext.GetLogger(imh).Infof("rewriting manifest list %s in schema1 format to support old client", imh.Digest.String())

		// Find the image manifest corresponding to the default
		// platform
		var manifestDigest digest.Digest
		for _, manifestDescriptor := range manifestList.Manifests {
			if manifestDescriptor.Platform.Architecture == defaultArch && manifestDescriptor.Platform.OS == defaultOS {
				manifestDigest = manifestDescriptor.Digest
				break
			}
		}

		if manifestDigest == "" {
			imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown)
			return
		}

		if imh.Config.Database.Enabled {
			manifest, err = dbGetManifest(imh, imh.App.db, manifestDigest, imh.App.trustKey, imh.Repository.Named().Name())
		} else {
			manifest, err = manifests.Get(imh, manifestDigest)
		}
		if err != nil {
			if _, ok := err.(distribution.ErrManifestUnknownRevision); ok {
				imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown.WithDetail(err))
			} else {
				imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
			}
			return
		}

		// If necessary, convert the image manifest
		if schema2Manifest, isSchema2 := manifest.(*schema2.DeserializedManifest); isSchema2 && !supports[manifestSchema2] {
			manifest, err = imh.convertSchema2Manifest(schema2Manifest)
			if err != nil {
				return
			}
		} else {
			imh.Digest = manifestDigest
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

func dbGetManifest(ctx context.Context, db datastore.Queryer, dgst digest.Digest, schema1SigningKey libtrust.PrivateKey, path string) (distribution.Manifest, error) {
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

	return dbPayloadToManifest(dbManifest.Payload, dbManifest.MediaType, dbManifest.SchemaVersion, schema1SigningKey)
}

func dbGetManifestByTag(ctx context.Context, db datastore.Queryer, tagName string, schema1SigningKey libtrust.PrivateKey, path string) (distribution.Manifest, digest.Digest, error) {
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

	dbTag, err := repositoryStore.FindTagByName(ctx, r, tagName)
	if err != nil {
		return nil, "", err
	}
	if dbTag == nil {
		log.Warn("tag not found in database")
		return nil, "", distribution.ErrTagUnknown{Tag: tagName}
	}

	// Find manifest by its digest
	mStore := datastore.NewManifestStore(db)
	dbManifest, err := mStore.FindByID(ctx, dbTag.ManifestID)
	if err != nil {
		return nil, "", err
	}
	if dbManifest == nil {
		return nil, "", distribution.ErrManifestUnknown{Name: r.Name, Tag: dbTag.Name}
	}

	manifest, err := dbPayloadToManifest(dbManifest.Payload, dbManifest.MediaType, dbManifest.SchemaVersion, schema1SigningKey)
	if err != nil {
		return nil, "", err
	}

	return manifest, dbManifest.Digest, nil
}

func dbPayloadToManifest(payload []byte, mediaType string, schemaVersion int, schema1SigningKey libtrust.PrivateKey) (distribution.Manifest, error) {
	// TODO: Each case here is taken directly from the respective
	// registry/storage/*manifesthandler Unmarshal method. These are all relatively
	// simple with the exception of schema1. We cannot invoke them directly as
	// they are unexported. We should determine a single place for this logic
	// during refactoring https://gitlab.com/gitlab-org/container-registry/-/issues/135
	switch schemaVersion {
	case 1:
		var (
			signatures [][]byte
			err        error
		)

		jsig, err := libtrust.NewJSONSignature(payload, signatures...)
		if err != nil {
			return nil, err
		}

		if schema1SigningKey != nil {
			if err := jsig.Sign(schema1SigningKey); err != nil {
				return nil, err
			}
		}

		// Extract the pretty JWS
		raw, err := jsig.PrettySignature("signatures")
		if err != nil {
			return nil, err
		}

		var sm schema1.SignedManifest
		if err := json.Unmarshal(raw, &sm); err != nil {
			return nil, err
		}

		return &sm, nil
	case 2:
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

	return nil, fmt.Errorf("unrecognized manifest schema version %d", schemaVersion)
}

func (imh *manifestHandler) convertSchema2Manifest(schema2Manifest *schema2.DeserializedManifest) (distribution.Manifest, error) {
	targetDescriptor := schema2Manifest.Target()
	blobs := imh.Repository.Blobs(imh)
	configJSON, err := blobs.Get(imh, targetDescriptor.Digest)
	if err != nil {
		if err == distribution.ErrBlobUnknown {
			imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithDetail(err))
		} else {
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
		}
		return nil, err
	}

	ref := imh.Repository.Named()

	if imh.Tag != "" {
		ref, err = reference.WithTag(ref, imh.Tag)
		if err != nil {
			imh.Errors = append(imh.Errors, v2.ErrorCodeTagInvalid.WithDetail(err))
			return nil, err
		}
	}

	builder := schema1.NewConfigManifestBuilder(imh.Repository.Blobs(imh), imh.Context.App.trustKey, ref, configJSON)
	for _, d := range schema2Manifest.Layers {
		if err := builder.AppendReference(d); err != nil {
			imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithDetail(err))
			return nil, err
		}
	}
	manifest, err := builder.Build(imh)
	if err != nil {
		imh.Errors = append(imh.Errors, v2.ErrorCodeManifestInvalid.WithDetail(err))
		return nil, err
	}
	imh.Digest = digest.FromBytes(manifest.(*schema1.SignedManifest).Canonical)

	return manifest, nil
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
	dcontext.GetLogger(imh).Debug("PutImageManifest")
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
			dcontext.GetLogger(imh).Errorf("payload digest does match: %q != %q", desc.Digest, imh.Digest)
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
		dcontext.GetLogger(imh).Debug("Putting an OCI Manifest!")
	} else {
		dcontext.GetLogger(imh).Debug("Putting a Docker Manifest!")
	}

	var options []distribution.ManifestServiceOption
	if imh.Tag != "" {
		options = append(options, distribution.WithTag(imh.Tag))
	}

	if err := imh.applyResourcePolicy(manifest); err != nil {
		imh.Errors = append(imh.Errors, err)
		return
	}

	_, err = manifests.Put(imh, manifest, options...)
	if err != nil {
		// TODO(stevvooe): These error handling switches really need to be
		// handled by an app global mapper.
		if err == distribution.ErrUnsupported {
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnsupported)
			return
		}
		if err == distribution.ErrAccessDenied {
			imh.Errors = append(imh.Errors, errcode.ErrorCodeDenied)
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
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
		}
		return
	}

	// We're using the database and mirroring writes to the filesystem. We'll run
	// a transaction so we can revert any changes to the database in case that
	// any part of this multi-phase database operation fails.
	if imh.Config.Database.Enabled {
		tx, err := imh.App.db.Begin()
		if err != nil {
			imh.Errors = append(imh.Errors,
				errcode.ErrorCodeUnknown.WithDetail(fmt.Errorf("failed to create database transaction: %v", err)))
			return
		}
		defer tx.Rollback()

		switch reqManifest := manifest.(type) {
		case *schema1.SignedManifest:
			if err = dbPutManifestSchema1(imh, tx, imh.Digest, reqManifest, jsonBuf.Bytes(), imh.Repository.Named()); err != nil {
				imh.Errors = append(imh.Errors,
					errcode.ErrorCodeUnknown.WithDetail(fmt.Errorf("failed to write manifest to database: %v", err)))
				return
			}
		case *schema2.DeserializedManifest:
			// The config payload should already be pushed up and stored as a blob.
			cfgPayload, err := imh.Repository.Blobs(imh).Get(imh, reqManifest.Config.Digest)
			if err != nil {
				imh.Errors = append(imh.Errors,
					errcode.ErrorCodeUnknown.WithDetail(fmt.Errorf("failed to obtain configuration payload: %v", err)))
				return
			}

			if err = dbPutManifestSchema2(imh, tx, imh.Digest, reqManifest, jsonBuf.Bytes(), cfgPayload, imh.Repository.Named()); err != nil {
				imh.Errors = append(imh.Errors,
					errcode.ErrorCodeUnknown.WithDetail(fmt.Errorf("failed to write manifest to database: %v", err)))
				return
			}
		case *ocischema.DeserializedManifest:
			// The config payload should already be pushed up and stored as a blob.
			cfgPayload, err := imh.Repository.Blobs(imh).Get(imh, reqManifest.Config.Digest)
			if err != nil {
				imh.Errors = append(imh.Errors,
					errcode.ErrorCodeUnknown.WithDetail(fmt.Errorf("failed to obtain manifest configuration payload: %v", err)))
				return
			}

			if err = dbPutManifestOCI(imh, tx, imh.Digest, reqManifest, jsonBuf.Bytes(), cfgPayload, imh.Repository.Named()); err != nil {
				imh.Errors = append(imh.Errors,
					errcode.ErrorCodeUnknown.WithDetail(fmt.Errorf("failed to write manifest to database: %v", err)))
				return
			}
		case *manifestlist.DeserializedManifestList:
			if err = dbPutManifestList(imh, tx, imh.Digest, reqManifest, jsonBuf.Bytes(), imh.Repository.Named()); err != nil {
				imh.Errors = append(imh.Errors,
					errcode.ErrorCodeUnknown.WithDetail(fmt.Errorf("failed to write manifest list to database: %v", err)))
				return
			}
		default:
			dcontext.GetLoggerWithField(imh, "manifest_class", fmt.Sprintf("%T", reqManifest)).Warn("database does not support manifest class")
		}
		if err := tx.Commit(); err != nil {
			imh.Errors = append(imh.Errors,
				errcode.ErrorCodeUnknown.WithDetail(fmt.Errorf("failed to commit manifest to database: %v", err)))
			return
		}
	}

	// Tag this manifest
	if imh.Tag != "" {
		tags := imh.Repository.Tags(imh)
		err = tags.Tag(imh, imh.Tag, desc)
		if err != nil {
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
			return
		}

		// Associate tag with manifest in database.
		if imh.Config.Database.Enabled {
			tx, err := imh.App.db.Begin()
			if err != nil {
				e := fmt.Errorf("failed to create database transaction: %v", err)
				imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(e))
				return
			}
			defer tx.Rollback()

			switch reqManifest := manifest.(type) {
			case *schema2.DeserializedManifest, *schema1.SignedManifest, *ocischema.DeserializedManifest, *manifestlist.DeserializedManifestList:
				if err := dbTagManifest(imh, tx, imh.Digest, imh.Tag, imh.Repository.Named().Name()); err != nil {
					e := fmt.Errorf("failed to create tag in database: %v", err)
					imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(e))
					return
				}
			default:
				dcontext.GetLoggerWithField(imh, "manifest_class", fmt.Sprintf("%T", reqManifest)).
					Warn("database does not support manifest class")
			}
			if err := tx.Commit(); err != nil {
				e := fmt.Errorf("failed to commit tag to database: %v", err)
				imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(e))
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
		dcontext.GetLogger(imh).Errorf("error building manifest url from digest: %v", err)
	}

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", imh.Digest.String())
	w.WriteHeader(http.StatusCreated)

	dcontext.GetLogger(imh).Debug("Succeeded in putting manifest!")
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
	manifestStore := datastore.NewManifestStore(db)
	dbManifest, err := manifestStore.FindByDigest(ctx, dgst)
	if err != nil {
		return err
	}
	if dbManifest == nil {
		return fmt.Errorf("manifest %s not found in database", dgst)
	}

	tagStore := datastore.NewTagStore(db)

	dbTag, err := repositoryStore.FindTagByName(ctx, dbRepo, tagName)
	if err != nil {
		return err
	}

	if dbTag != nil {
		log.Debug("tag already exists in database")

		// Tag exists and already points to the current manifest.
		if dbTag.ManifestID == dbManifest.ID {
			log.Debug("tag already associated with current manifest")
			return nil
		}

		// Tag exists, but refers to another manifest, update the manifest to which the tag refers.
		log.Debug("updating tag with manifest ID")
		dbTag.ManifestID = dbManifest.ID

		return tagStore.Update(ctx, dbTag)
	}

	// Tag does not exist, create it.
	log.Debug("creating new tag")
	return tagStore.Create(ctx, &models.Tag{
		Name:         tagName,
		RepositoryID: dbRepo.ID,
		ManifestID:   dbManifest.ID,
	})
}

func dbPutManifestOCI(ctx context.Context, db datastore.Queryer, dgst digest.Digest, manifest *ocischema.DeserializedManifest, payload, cfgPayload []byte, path reference.Named) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": path.Name(), "manifest_digest": dgst, "schema_version": manifest.Versioned.SchemaVersion})
	log.Debug("putting manifest")

	// Find or create the configuration.
	cfgStore := datastore.NewConfigurationStore(db)
	blobStore := datastore.NewBlobStore(db)

	dbCfg, err := cfgStore.FindByDigest(ctx, manifest.Config.Digest)
	if err != nil {
		return err
	}

	if dbCfg == nil {
		log.Debug("manifest config not found in database")

		dbCfgBlob, err := blobStore.FindByDigest(ctx, manifest.Config.Digest)
		if err != nil {
			return err
		}
		if dbCfgBlob == nil {
			return fmt.Errorf("config blob %s not found in database", manifest.Config.Digest)
		}

		dbCfg = &models.Configuration{BlobID: dbCfgBlob.ID, Payload: cfgPayload}
		if err := cfgStore.Create(ctx, dbCfg); err != nil {
			return err
		}
	}
	// TODO: update the config blob media_type here, it was set to "application/octect-stream" during the upload
	// 		 but now we know its concrete type (manifest.Config.MediaType).

	mStore := datastore.NewManifestStore(db)
	dbManifest, err := mStore.FindByDigest(ctx, dgst)
	if err != nil {
		return err
	}
	if dbManifest == nil {
		log.Debug("manifest not found in database")

		m := &models.Manifest{
			ConfigurationID: sql.NullInt64{Int64: dbCfg.ID, Valid: true},
			SchemaVersion:   manifest.SchemaVersion,
			MediaType:       manifest.MediaType,
			Digest:          dgst,
			Payload:         payload,
		}

		if err := mStore.Create(ctx, m); err != nil {
			return err
		}

		dbManifest = m

		// find and associate manifest layer blobs
		for _, reqLayer := range manifest.Layers {
			dbBlob, err := blobStore.FindByDigest(ctx, reqLayer.Digest)
			if err != nil {
				return err
			}

			if dbBlob == nil {
				return fmt.Errorf("layer blob %s not found in database", reqLayer.Digest)
			}

			// TODO: update the layer blob media_type here, it was set to "application/octect-stream" during the upload
			// 		 but now we know its concrete type (reqLayer.MediaType).

			if err := mStore.AssociateLayerBlob(ctx, dbManifest, dbBlob); err != nil {
				return err
			}
		}
	}

	// Associate manifest and repository.
	repositoryStore := datastore.NewRepositoryStore(db)
	dbRepo, err := repositoryStore.CreateOrFindByPath(ctx, path.Name())
	if err != nil {
		return err
	}

	if err := repositoryStore.AssociateManifest(ctx, dbRepo, dbManifest); err != nil {
		return err
	}
	return nil
}

func dbPutManifestSchema2(ctx context.Context, db datastore.Queryer, dgst digest.Digest, manifest *schema2.DeserializedManifest, payload, cfgPayload []byte, path reference.Named) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": path.Name(), "manifest_digest": dgst, "schema_version": manifest.Versioned.SchemaVersion})
	log.Debug("putting manifest")

	// Find or create the configuration.
	cfgStore := datastore.NewConfigurationStore(db)
	blobStore := datastore.NewBlobStore(db)

	dbCfg, err := cfgStore.FindByDigest(ctx, manifest.Config.Digest)
	if err != nil {
		return err
	}

	if dbCfg == nil {
		log.Debug("manifest config not found in database")

		dbCfgBlob, err := blobStore.FindByDigest(ctx, manifest.Config.Digest)
		if err != nil {
			return err
		}
		if dbCfgBlob == nil {
			return fmt.Errorf("config blob %s not found in database", manifest.Config.Digest)
		}

		dbCfg = &models.Configuration{BlobID: dbCfgBlob.ID, Payload: cfgPayload}
		if err := cfgStore.Create(ctx, dbCfg); err != nil {
			return err
		}
	}
	// TODO: update the config blob media_type here, it was set to "application/octect-stream" during the upload
	// 		 but now we know its concrete type (manifest.Config.MediaType).

	mStore := datastore.NewManifestStore(db)
	dbManifest, err := mStore.FindByDigest(ctx, dgst)
	if err != nil {
		return err
	}
	if dbManifest == nil {
		log.Debug("manifest not found in database")

		m := &models.Manifest{
			ConfigurationID: sql.NullInt64{Int64: dbCfg.ID, Valid: true},
			SchemaVersion:   manifest.SchemaVersion,
			MediaType:       manifest.MediaType,
			Digest:          dgst,
			Payload:         payload,
		}

		if err := mStore.Create(ctx, m); err != nil {
			return err
		}

		dbManifest = m

		// find and associate manifest layer blobs
		for _, reqLayer := range manifest.Layers {
			dbBlob, err := blobStore.FindByDigest(ctx, reqLayer.Digest)
			if err != nil {
				return err
			}

			if dbBlob == nil {
				return fmt.Errorf("layer blob %s not found in database", reqLayer.Digest)
			}

			// TODO: update the layer blob media_type here, it was set to "application/octect-stream" during the upload
			// 		 but now we know its concrete type (reqLayer.MediaType).

			if err := mStore.AssociateLayerBlob(ctx, dbManifest, dbBlob); err != nil {
				return err
			}
		}
	}

	// Associate manifest and repository.
	repositoryStore := datastore.NewRepositoryStore(db)
	dbRepo, err := repositoryStore.CreateOrFindByPath(ctx, path.Name())
	if err != nil {
		return err
	}

	if err := repositoryStore.AssociateManifest(ctx, dbRepo, dbManifest); err != nil {
		return err
	}
	return nil
}

func dbPutManifestSchema1(ctx context.Context, db datastore.Queryer, dgst digest.Digest, manifest *schema1.SignedManifest, payload []byte, path reference.Named) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": path.Name(), "manifest_digest": dgst, "schema_version": manifest.Versioned.SchemaVersion})
	log.Debug("putting manifest")

	mStore := datastore.NewManifestStore(db)
	dbManifest, err := mStore.FindByDigest(ctx, dgst)
	if err != nil {
		return err
	}
	if dbManifest == nil {
		log.Debug("manifest not found in database")

		m := &models.Manifest{
			SchemaVersion: manifest.SchemaVersion,
			MediaType:     manifest.MediaType,
			Digest:        dgst,
			Payload:       manifest.Canonical,
		}

		if err := mStore.Create(ctx, m); err != nil {
			return err
		}

		dbManifest = m

		// find and associate manifest layer blobs
		blobStore := datastore.NewBlobStore(db)
		for _, layer := range manifest.FSLayers {
			dbBlob, err := blobStore.FindByDigest(ctx, layer.BlobSum)
			if err != nil {
				return err
			}

			if dbBlob == nil {
				return fmt.Errorf("layer blob %s not found in database", layer.BlobSum)
			}

			// TODO: update the layer blob media_type here, it was set to "application/octect-stream" during the upload
			// 		 but now we know its concrete type (reqLayer.MediaType).

			if err := mStore.AssociateLayerBlob(ctx, dbManifest, dbBlob); err != nil {
				return err
			}
		}
	}

	// Associate manifest and repository.
	repositoryStore := datastore.NewRepositoryStore(db)
	dbRepo, err := repositoryStore.CreateOrFindByPath(ctx, path.Name())
	if err != nil {
		return err
	}

	if err := repositoryStore.AssociateManifest(ctx, dbRepo, dbManifest); err != nil {
		return err
	}
	return nil
}

func dbPutManifestList(ctx context.Context, db datastore.Queryer, canonical digest.Digest, manifestList *manifestlist.DeserializedManifestList, payload []byte, path reference.Named) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": path.Name(), "manifest_digest": canonical})
	log.Debug("putting manifest list")

	mStore := datastore.NewManifestStore(db)
	dbManifestList, err := mStore.FindByDigest(ctx, canonical)
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
			SchemaVersion: manifestList.SchemaVersion,
			MediaType:     mediaType,
			Digest:        canonical,
			Payload:       payload,
		}
		if err := mStore.Create(ctx, dbManifestList); err != nil {
			return err
		}

		// Associate manifests to the manifest list.
		for _, m := range manifestList.Manifests {
			dbManifest, err := mStore.FindByDigest(ctx, m.Digest)
			if err != nil {
				return err
			}
			if dbManifest == nil {
				return fmt.Errorf("manifest %s not found", m.Digest)
			}
			if err := mStore.AssociateManifest(ctx, dbManifestList, dbManifest); err != nil {
				return err
			}
		}
	}

	// Associate manifest list and repository.
	repositoryStore := datastore.NewRepositoryStore(db)
	dbRepo, err := repositoryStore.CreateOrFindByPath(ctx, path.Name())
	if err != nil {
		return err
	}

	return repositoryStore.AssociateManifest(ctx, dbRepo, dbManifestList)
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
	case *schema1.SignedManifest:
		class = imageClass
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

// dbDeleteManifest replicates the DeleteManifest action in the metadata database. This method doesn't actually delete
// a manifest from the database (that's a task for GC, if a manifest is unreferenced), it only deletes the record that
// associates the manifest with a digest d with the repository with path repoPath. Any tags that reference the manifest
// within the repository are also deleted.
func dbDeleteManifest(ctx context.Context, db datastore.Queryer, repoPath string, d digest.Digest) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": repoPath, "digest": d})

	rStore := datastore.NewRepositoryStore(db)
	r, err := rStore.FindByPath(ctx, repoPath)
	if err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("repository not found in database: %w", err)
	}

	m, err := rStore.FindManifestByDigest(ctx, r, d)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("no manifest found in database")
	}

	log.Debug("manifest found in database")
	if err := rStore.DissociateManifest(ctx, r, m); err != nil {
		return err
	}

	return rStore.UntagManifest(ctx, r, m)
}

// DeleteManifest removes the manifest with the given digest from the registry.
func (imh *manifestHandler) DeleteManifest(w http.ResponseWriter, r *http.Request) {
	dcontext.GetLogger(imh).Debug("DeleteImageManifest")

	manifests, err := imh.Repository.Manifests(imh)
	if err != nil {
		imh.Errors = append(imh.Errors, err)
		return
	}

	err = manifests.Delete(imh, imh.Digest)
	if err != nil {
		switch err {
		case digest.ErrDigestUnsupported:
		case digest.ErrDigestInvalidFormat:
			imh.Errors = append(imh.Errors, v2.ErrorCodeDigestInvalid)
			return
		case distribution.ErrBlobUnknown:
			imh.Errors = append(imh.Errors, v2.ErrorCodeManifestUnknown)
			return
		case distribution.ErrUnsupported:
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnsupported)
			return
		default:
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown)
			return
		}
	}

	tagService := imh.Repository.Tags(imh)
	referencedTags, err := tagService.Lookup(imh, distribution.Descriptor{Digest: imh.Digest})
	if err != nil {
		imh.Errors = append(imh.Errors, err)
		return
	}

	for _, tag := range referencedTags {
		if err := tagService.Untag(imh, tag); err != nil {
			imh.Errors = append(imh.Errors, err)
			return
		}
	}

	if imh.App.Config.Database.Enabled {
		tx, err := imh.db.BeginTx(r.Context(), nil)
		if err != nil {
			e := fmt.Errorf("failed to create database transaction: %v", err)
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(e))
			return
		}
		defer tx.Rollback()

		if err = dbDeleteManifest(imh, tx, imh.Repository.Named().String(), imh.Digest); err != nil {
			e := fmt.Errorf("failed to delete manifest in database: %v", err)
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(e))
			return
		}

		if err := tx.Commit(); err != nil {
			e := fmt.Errorf("failed to commit database transaction: %v", err)
			imh.Errors = append(imh.Errors, errcode.ErrorCodeUnknown.WithDetail(e))
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}
