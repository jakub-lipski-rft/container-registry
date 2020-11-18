package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/docker/distribution/configuration"

	dcontext "github.com/docker/distribution/context"

	"github.com/docker/distribution/registry/datastore"

	"github.com/docker/distribution"
	"github.com/docker/distribution/registry/api/errcode"
	v2 "github.com/docker/distribution/registry/api/v2"
	"github.com/gorilla/handlers"
	"github.com/opencontainers/go-digest"
)

// blobDispatcher uses the request context to build a blobHandler.
func blobDispatcher(ctx *Context, r *http.Request) http.Handler {
	dgst, err := getDigest(ctx)
	if err != nil {
		if err == errDigestNotAvailable {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx.Errors = append(ctx.Errors, v2.ErrorCodeDigestInvalid.WithDetail(err))
			})
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx.Errors = append(ctx.Errors, v2.ErrorCodeDigestInvalid.WithDetail(err))
		})
	}

	blobHandler := &blobHandler{
		Context: ctx,
		Digest:  dgst,
	}

	mhandler := handlers.MethodHandler{
		"GET":  http.HandlerFunc(blobHandler.GetBlob),
		"HEAD": http.HandlerFunc(blobHandler.GetBlob),
	}

	if !ctx.readOnly {
		mhandler["DELETE"] = http.HandlerFunc(blobHandler.DeleteBlob)
	}

	return mhandler
}

// blobHandler serves http blob requests.
type blobHandler struct {
	*Context

	Digest digest.Digest
}

func dbGetBlob(ctx context.Context, db datastore.Queryer, repoPath string, dgst digest.Digest) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": repoPath, "digest": dgst})
	log.Debug("finding blob in database")

	rStore := datastore.NewRepositoryStore(db)
	r, err := rStore.FindByPath(ctx, repoPath)
	if err != nil {
		return errcode.ErrorCodeUnknown.WithDetail(err)
	}
	if r == nil {
		log.Warn("repository not found in database")
		return v2.ErrorCodeBlobUnknown.WithDetail(dgst)
	}

	bb, err := rStore.Blobs(ctx, r)
	if err != nil {
		return errcode.ErrorCodeUnknown.WithDetail(err)
	}

	var found bool
	for _, b := range bb {
		if b.Digest == dgst {
			found = true
			break
		}
	}
	if !found {
		log.Warn("blob link not found in database")
		return v2.ErrorCodeBlobUnknown.WithDetail(dgst)
	}

	return nil
}

// GetBlob fetches the binary data from backend storage returns it in the
// response.
func (bh *blobHandler) GetBlob(w http.ResponseWriter, r *http.Request) {
	dcontext.GetLogger(bh).Debug("GetBlob")

	var dgst digest.Digest
	blobs := bh.Repository.Blobs(bh)

	if bh.Config.Database.Enabled {
		if err := dbGetBlob(bh.Context, bh.db, bh.Repository.Named().Name(), bh.Digest); err != nil {
			bh.Errors = append(bh.Errors, err)
			return
		}

		dgst = bh.Digest
	} else {
		desc, err := blobs.Stat(bh, bh.Digest)
		if err != nil {
			if err == distribution.ErrBlobUnknown {
				bh.Errors = append(bh.Errors, v2.ErrorCodeBlobUnknown.WithDetail(bh.Digest))
			} else {
				bh.Errors = append(bh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
			}
			return
		}

		dgst = desc.Digest
	}

	if err := blobs.ServeBlob(bh, w, r, dgst); err != nil {
		dcontext.GetLogger(bh).Debugf("unexpected error getting blob HTTP handler: %v", err)
		bh.Errors = append(bh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}
}

// dbDeleteBlob does not actually delete a blob from the database (that's GC's responsibility), it only unlinks it from
// a repository.
func dbDeleteBlob(ctx context.Context, config *configuration.Configuration, db datastore.Queryer, repoPath string, d digest.Digest) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": repoPath, "digest": d})
	log.Debug("deleting blob from repository in database")

	if !deleteEnabled(config) {
		return distribution.ErrUnsupported
	}

	rStore := datastore.NewRepositoryStore(db)
	r, err := rStore.FindByPath(ctx, repoPath)
	if err != nil {
		return err
	}
	if r == nil {
		return distribution.ErrRepositoryUnknown{Name: repoPath}
	}

	found, err := rStore.UnlinkBlob(ctx, r, d)
	if err != nil {
		return err
	}
	if !found {
		return distribution.ErrBlobUnknown
	}

	return nil
}

func deleteEnabled(config *configuration.Configuration) bool {
	if d, ok := config.Storage["delete"]; ok {
		e, ok := d["enabled"]
		if ok {
			if deleteEnabled, ok := e.(bool); ok && deleteEnabled {
				return true
			}
		}
	}
	return false
}

// DeleteBlob deletes a layer blob
func (bh *blobHandler) DeleteBlob(w http.ResponseWriter, r *http.Request) {
	dcontext.GetLogger(bh).Debug("DeleteBlob")

	err := bh.deleteBlob()
	if err != nil {
		switch err {
		case distribution.ErrUnsupported:
			bh.Errors = append(bh.Errors, errcode.ErrorCodeUnsupported)
			return
		case distribution.ErrBlobUnknown:
			bh.Errors = append(bh.Errors, v2.ErrorCodeBlobUnknown)
			return
		case distribution.ErrRepositoryUnknown{Name: bh.Repository.Named().Name()}:
			bh.Errors = append(bh.Errors, v2.ErrorCodeNameUnknown)
			return
		default:
			bh.Errors = append(bh.Errors, err)
			dcontext.GetLogger(bh).WithError(err).Error("Unknown error deleting blob")
			return
		}
	}

	w.Header().Set("Content-Length", "0")
	w.WriteHeader(http.StatusAccepted)
}

func (bh *blobHandler) deleteBlob() error {
	if !bh.App.Config.Migration.DisableMirrorFS {
		blobs := bh.Repository.Blobs(bh)
		if err := blobs.Delete(bh, bh.Digest); err != nil {
			return err
		}

		if !bh.App.Config.Database.Enabled {
			return nil
		}

	}

	if bh.App.Config.Database.Enabled {
		return dbDeleteBlob(bh, bh.App.Config, bh.db, bh.Repository.Named().Name(), bh.Digest)
	}

	// If we reach this point, we should have failed on an invalid config already,
	// but we'll write nominal response headers if neither of the above cases
	// executes, so we should avoid that.
	return errors.New("no metadata database and filesystem metadata is disabled")
}
