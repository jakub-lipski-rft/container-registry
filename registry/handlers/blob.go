package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/docker/distribution/registry/datastore/models"

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
	blobs := bh.Repository.Blobs(bh)
	desc, err := blobs.Stat(bh, bh.Digest)
	if err != nil {
		if err == distribution.ErrBlobUnknown {
			bh.Errors = append(bh.Errors, v2.ErrorCodeBlobUnknown.WithDetail(bh.Digest))
		} else {
			bh.Errors = append(bh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
		}
		return
	}

	if bh.Config.Database.Enabled {
		if err := dbGetBlob(bh.Context, bh.db, bh.Repository.Named().Name(), bh.Digest); err != nil {
			dcontext.GetLogger(bh).WithError(err).Warn("unable to fetch blob from database, falling back to filesystem")
		}
	}

	if err := blobs.ServeBlob(bh, w, r, desc.Digest); err != nil {
		dcontext.GetLogger(bh).Debugf("unexpected error getting blob HTTP handler: %v", err)
		bh.Errors = append(bh.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}
}

// dbDeleteBlob does not actually delete a blob from the database (that's GC's responsibility), it only unlinks it from
// a repository.
func dbDeleteBlob(ctx context.Context, db datastore.Queryer, repoPath string, d digest.Digest, fallback bool) error {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": repoPath, "digest": d})
	log.Debug("deleting blob from repository in database")

	rStore := datastore.NewRepositoryStore(db)
	r, err := rStore.FindByPath(ctx, repoPath)
	if err != nil {
		return err
	}
	if r == nil {
		if fallback {
			log.Warn("repository not found in database, no need to unlink from the blob")
			return nil
		}

		return errors.New("repository not found in database")
	}

	bb, err := rStore.Blobs(ctx, r)
	if err != nil {
		return err
	}

	var b *models.Blob
	for _, layer := range bb {
		if layer.Digest == d {
			b = layer
			break
		}
	}
	if b == nil {
		if fallback {
			log.Warn("blob not found in database, no need to unlink it from the repository")
			return nil
		}

		return errors.New("blob not found in database")
	}

	return rStore.UnlinkBlob(ctx, r, b)
}

// DeleteBlob deletes a layer blob
func (bh *blobHandler) DeleteBlob(w http.ResponseWriter, r *http.Request) {
	dcontext.GetLogger(bh).Debug("DeleteBlob")

	blobs := bh.Repository.Blobs(bh)
	err := blobs.Delete(bh, bh.Digest)
	if err != nil {
		switch err {
		case distribution.ErrUnsupported:
			bh.Errors = append(bh.Errors, errcode.ErrorCodeUnsupported)
			return
		case distribution.ErrBlobUnknown:
			bh.Errors = append(bh.Errors, v2.ErrorCodeBlobUnknown)
			return
		default:
			bh.Errors = append(bh.Errors, err)
			dcontext.GetLogger(bh).Errorf("Unknown error deleting blob: %s", err.Error())
			return
		}
	}

	if bh.App.Config.Database.Enabled {
		if err := dbDeleteBlob(bh, bh.db, bh.Repository.Named().Name(), bh.Digest, bh.App.Config.Database.Experimental.Fallback); err != nil {
			e := fmt.Errorf("failed to delete blob in database: %v", err)
			bh.Errors = append(bh.Errors, errcode.ErrorCodeUnknown.WithDetail(e))
			return
		}
	}

	w.Header().Set("Content-Length", "0")
	w.WriteHeader(http.StatusAccepted)
}
