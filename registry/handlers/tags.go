package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/docker/distribution/registry/datastore"

	"github.com/docker/distribution"
	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/api/errcode"
	v2 "github.com/docker/distribution/registry/api/v2"
	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/gorilla/handlers"
)

// tagsDispatcher constructs the tags handler api endpoint.
func tagsDispatcher(ctx *Context, r *http.Request) http.Handler {
	tagsHandler := &tagsHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"GET": http.HandlerFunc(tagsHandler.GetTags),
	}
}

// tagsHandler handles requests for lists of tags under a repository name.
type tagsHandler struct {
	*Context
}

type tagsAPIResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func dbGetTags(ctx context.Context, db datastore.Queryer, repoPath string, n int, last string) ([]string, bool, error) {
	log := dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{"repository": repoPath, "limit": n, "marker": last})
	log.Debug("finding tags in database")
	
	rStore := datastore.NewRepositoryStore(db)
	r, err := rStore.FindByPath(ctx, repoPath)
	if err != nil {
		return nil, false, errcode.ErrorCodeUnknown.WithDetail(err)
	}
	if r == nil {
		log.Warn("repository not found in database")
		return nil, false, v2.ErrorCodeNameUnknown.WithDetail(map[string]string{"name": repoPath})
	}

	tt, err := rStore.TagsPaginated(ctx, r, n, last)
	if err != nil {
		return nil, false, errcode.ErrorCodeUnknown.WithDetail(err)
	}

	tags := make([]string, 0, len(tt))
	for _, t := range tt {
		tags = append(tags, t.Name)
	}

	var moreEntries bool
	if len(tt) > 0 {
		n, err := rStore.TagsCountAfterName(ctx, r, tt[len(tt)-1].Name)
		if err != nil {
			return nil, false, errcode.ErrorCodeUnknown.WithDetail(err)
		}
		moreEntries = n > 0
	}

	return tags, moreEntries, nil
}

// GetTags returns a json list of tags for a specific image name.
func (th *tagsHandler) GetTags(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Pagination headers are currently only supported by the metadata database backend
	q := r.URL.Query()
	lastEntry := q.Get("last")
	maxEntries, err := strconv.Atoi(q.Get("n"))
	if err != nil || maxEntries <= 0 {
		maxEntries = maximumReturnedEntries
	}

	var tags []string
	var moreEntries bool

	if th.Config.Database.Enabled {
		tags, moreEntries, err = dbGetTags(th.Context, th.db, th.Repository.Named().Name(), maxEntries, lastEntry)
		if err != nil {
			th.Errors = append(th.Errors, err)
			return
		}
		if len(tags) == 0 {
			// If no tags are found, the current implementation (`else`) returns a nil slice instead of an empty one,
			// so we have to enforce the same behavior here, for consistency.
			tags = nil
		}
	} else {
		tagService := th.Repository.Tags(th)
		tags, err = tagService.All(th)
		if err != nil {
			switch err := err.(type) {
			case distribution.ErrRepositoryUnknown:
				th.Errors = append(th.Errors, v2.ErrorCodeNameUnknown.WithDetail(map[string]string{"name": th.Repository.Named().Name()}))
			case errcode.Error:
				th.Errors = append(th.Errors, err)
			default:
				th.Errors = append(th.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
			}
			return
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Add a link header if there are more entries to retrieve (only supported by the metadata database backend)
	if moreEntries {
		lastEntry = tags[len(tags)-1]
		urlStr, err := createLinkEntry(r.URL.String(), maxEntries, lastEntry)
		if err != nil {
			th.Errors = append(th.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
			return
		}
		w.Header().Set("Link", urlStr)
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(tagsAPIResponse{
		Name: th.Repository.Named().Name(),
		Tags: tags,
	}); err != nil {
		th.Errors = append(th.Errors, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}
}

// tagDispatcher constructs the tag handler api endpoint.
func tagDispatcher(ctx *Context, r *http.Request) http.Handler {
	thandler := handlers.MethodHandler{}

	tagHandler := &tagHandler{
		Context: ctx,
		Tag:     getTag(ctx),
	}

	if !ctx.readOnly {
		thandler["DELETE"] = http.HandlerFunc(tagHandler.DeleteTag)
	}

	return thandler
}

// tagHandler handles requests for a specific tag under a repository name.
type tagHandler struct {
	*Context
	Tag string
}

func dbDeleteTag(ctx context.Context, db datastore.Queryer, repoPath string, tagName string) error {
	rStore := datastore.NewRepositoryStore(db)
	r, err := rStore.FindByPath(ctx, repoPath)
	if err != nil {
		return err
	}
	if r == nil {
		return errors.New("repository not found in database")
	}

	t, err := rStore.FindTagByName(ctx, r, tagName)
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New("tag not found in database")
	}

	tStore := datastore.NewTagStore(db)
	return tStore.Delete(ctx, t.ID)
}

// DeleteTag deletes a tag for a specific image name.
func (th *tagHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	dcontext.GetLogger(th).Debug("DeleteTag")

	if th.App.isCache {
		th.Errors = append(th.Errors, errcode.ErrorCodeUnsupported)
		return
	}

	tagService := th.Repository.Tags(th)
	if err := tagService.Untag(th.Context, th.Tag); err != nil {
		switch err.(type) {
		case distribution.ErrTagUnknown:
		case storagedriver.PathNotFoundError:
			th.Errors = append(th.Errors, v2.ErrorCodeManifestUnknown)
		default:
			th.Errors = append(th.Errors, errcode.ErrorCodeUnknown)
		}
		return
	}

	if th.App.Config.Database.Enabled {
		if err := dbDeleteTag(th, th.db, th.Repository.Named().Name(), th.Tag); err != nil {
			e := fmt.Errorf("failed to delete tag in database: %v", err)
			th.Errors = append(th.Errors, errcode.ErrorCodeUnknown.WithDetail(e))
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}
