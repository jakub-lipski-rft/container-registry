// +build integration

package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/stretchr/testify/require"
)

// This file is intended to test the HTTP API tolerance and behaviour under scenarios that are prone to race conditions
// due to online GC.

func findAndLockGCManifestTask(t *testing.T, env *testEnv, repoName reference.Named, tagName string) (*models.GCManifestTask, datastore.Transactor) {
	tx, err := env.db.BeginTx(env.ctx, nil)
	require.NoError(t, err)

	rStore := datastore.NewRepositoryStore(tx)
	r, err := rStore.FindByPath(env.ctx, repoName.Name())
	require.NoError(t, err)
	require.NotNil(t, r)

	tag, err := rStore.FindTagByName(env.ctx, r, tagName)
	require.NoError(t, err)
	require.NotNil(t, tag)

	mts := datastore.NewGCManifestTaskStore(tx)
	mt, err := mts.FindAndLockBefore(env.ctx, r.ID, tag.ManifestID, time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	require.NotNil(t, mt)

	return mt, tx
}

// TestTagsAPI_Delete_OnlineGC_BlocksAndResumesAfterGCReview tests that when we try to delete a tag that points to a
// manifest that is being reviewed by the online GC, the API is not able to delete the tag until GC completes.
// https://gitlab.com/gitlab-org/container-registry/-/blob/master/docs-gitlab/db/online-garbage-collection.md#deleting-the-last-referencing-tag
func TestTagsAPI_Delete_OnlineGC_BlocksAndResumesAfterGCReview(t *testing.T) {
	env := newTestEnv(t, withDelete)
	defer env.Shutdown()

	if !env.config.Database.Enabled {
		t.Skip("skipping test because the metadata database is not enabled")
	}

	// create test repo with a single manifest and tag
	repoName, err := reference.WithName("test")
	require.NoError(t, err)

	tagName := "1.0.0"
	createRepository(t, env, repoName.Name(), tagName)

	// simulate GC process by locking the manifest review record
	mt, tx := findAndLockGCManifestTask(t, env, repoName, tagName)
	defer tx.Rollback()

	// simulate GC manifest review happening in the background while we make the API request
	lockDuration := 2 * time.Second
	time.AfterFunc(lockDuration, func() {
		// the manifest is not dangling, so we delete the GC task and commit transaction, as the GC would do
		mts := datastore.NewGCManifestTaskStore(tx)
		require.NoError(t, mts.Delete(env.ctx, mt))
		require.NoError(t, tx.Rollback())
	})

	// attempt to delete tag through the API, this should succeed after waiting for lockDuration
	ref, err := reference.WithTag(repoName, tagName)
	require.NoError(t, err)
	tagDeleteURL, err := env.builder.BuildTagURL(ref)
	require.NoError(t, err)

	start := time.Now()
	resp, err := httpDelete(tagDeleteURL)
	end := time.Now()
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	require.WithinDuration(t, start, end, lockDuration+100*time.Millisecond)
}

// TestTagsAPI_Delete_OnlineGC_TimeoutOnProlongedReview tests that when we try to delete a tag that points to a
// manifest that is being reviewed by the online GC, and for some reason the review does not end within
// tagDeleteGCLockTimeout, the API request is aborted and a 503 Service Unavailable response is returned to clients.
// https://gitlab.com/gitlab-org/container-registry/-/blob/master/docs-gitlab/db/online-garbage-collection.md#deleting-the-last-referencing-tag
func TestTagsAPI_Delete_OnlineGC_TimeoutOnProlongedReview(t *testing.T) {
	env := newTestEnv(t, withDelete)
	defer env.Shutdown()

	if !env.config.Database.Enabled {
		t.Skip("skipping test because the metadata database is not enabled")
	}

	// create test repo and tag
	repoName, err := reference.WithName("test")
	require.NoError(t, err)

	tagName := "1.0.0"
	createRepository(t, env, repoName.Name(), tagName)

	// simulate GC process by locking the manifest review record indefinitely
	_, tx := findAndLockGCManifestTask(t, env, repoName, tagName)
	defer tx.Rollback()

	// attempt to delete tag through the API, this should fail after waiting for tagDeleteGCLockTimeout (5 seconds)
	ref, err := reference.WithTag(repoName, tagName)
	require.NoError(t, err)

	tagDeleteURL, err := env.builder.BuildTagURL(ref)
	require.NoError(t, err)

	start := time.Now()
	resp, err := httpDelete(tagDeleteURL)
	end := time.Now()
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	require.WithinDuration(t, start, end, 5*time.Second+100*time.Millisecond)
}
