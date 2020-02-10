package storage

import (
	"context"
	"path"
	"time"

	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/opencontainers/go-digest"
)

// vacuum contains functions for cleaning up repositories and blobs
// These functions will only reliably work on strongly consistent
// storage systems.
// https://en.wikipedia.org/wiki/Consistency_model

// NewVacuum creates a new Vacuum
func NewVacuum(ctx context.Context, driver driver.StorageDriver) Vacuum {
	return Vacuum{
		ctx:    ctx,
		driver: driver,
	}
}

// Vacuum removes content from the filesystem
type Vacuum struct {
	driver driver.StorageDriver
	ctx    context.Context
}

// RemoveBlob removes a blob from the filesystem
func (v Vacuum) RemoveBlob(dgst digest.Digest) error {
	blobPath, err := pathFor(blobPathSpec{digest: dgst})
	if err != nil {
		return err
	}

	dcontext.GetLogger(v.ctx).Infof("Deleting blob: %s", blobPath)

	err = v.driver.Delete(v.ctx, blobPath)
	if err != nil {
		return err
	}

	return nil
}

// RemoveBlobs removes a list of blobs from the filesystem. This is used exclusively by the garbage collector and
// the intention is to leverage on bulk delete requests whenever supported by the storage backend.
func (v Vacuum) RemoveBlobs(dgsts []digest.Digest) error {
	start := time.Now()
	blobPaths := make([]string, 0, len(dgsts))
	for _, d := range dgsts {
		// get the full path of the blob's data file
		p, err := pathFor(blobDataPathSpec{digest: d})
		if err != nil {
			return err
		}
		dcontext.GetLoggerWithFields(v.ctx, map[interface{}]interface{}{
			"digest": d,
			"path":   p,
		}).Info("preparing to delete blob")
		blobPaths = append(blobPaths, p)
	}

	total := len(blobPaths)
	dcontext.GetLoggerWithField(v.ctx, "count", total).Info("deleting blobs")

	count, err := v.driver.DeleteFiles(v.ctx, blobPaths)

	l := dcontext.GetLoggerWithFields(v.ctx, map[interface{}]interface{}{
		"count":      count,
		"duration_s": time.Since(start).Seconds(),
	})
	if count < total {
		l.Warn("blobs partially deleted")
	} else {
		l.Info("blobs deleted")
	}

	return err
}

// RemoveManifests removes a series of manifests from the filesystem. Unlike RemoveManifest, this bundles all related
// tag index and manifest link files in a single driver.DeleteFiles request. The link files full path is used instead of
// their parent directory path (which always contains a single file, the link itself).
func (v Vacuum) RemoveManifests(mm []ManifestDel) error {
	start := time.Now()
	var manifestLinks, tagLinks, allLinks []string
	for _, m := range mm {
		// get manifest revision link full path
		p, err := pathFor(manifestRevisionLinkPathSpec{name: m.Name, revision: m.Digest})
		if err != nil {
			return err
		}
		dcontext.GetLoggerWithFields(v.ctx, map[interface{}]interface{}{
			"digest": m.Digest,
			"path":   p,
		}).Info("preparing to delete manifest")

		manifestLinks = append(manifestLinks, p)

		for _, t := range m.Tags {
			// get tag index link full path
			p, err := pathFor(manifestTagIndexEntryLinkPathSpec{name: m.Name, revision: m.Digest, tag: t})
			if err != nil {
				return err
			}
			dcontext.GetLoggerWithFields(v.ctx, map[interface{}]interface{}{
				"tag":  t,
				"path": p,
			}).Info("preparing to delete manifest tag reference")

			tagLinks = append(tagLinks, p)
		}
	}

	allLinks = append(manifestLinks, tagLinks...)
	total := len(allLinks)
	if total == 0 {
		return nil
	}

	dcontext.GetLoggerWithFields(v.ctx, map[interface{}]interface{}{
		"manifests": len(manifestLinks),
		"tags":      len(tagLinks),
		"total":     total,
	}).Info("deleting manifests")

	count, err := v.driver.DeleteFiles(v.ctx, allLinks)

	l := dcontext.GetLoggerWithFields(v.ctx, map[interface{}]interface{}{
		"count":      count,
		"duration_s": time.Since(start).Seconds(),
	})
	if count < total {
		l.Warn("manifests partially deleted")
	} else {
		l.Info("manifests deleted")
	}

	return err
}

// RemoveRepository removes a repository directory from the
// filesystem
func (v Vacuum) RemoveRepository(repoName string) error {
	rootForRepository, err := pathFor(repositoriesRootPathSpec{})
	if err != nil {
		return err
	}
	repoDir := path.Join(rootForRepository, repoName)
	dcontext.GetLogger(v.ctx).Infof("Deleting repo: %s", repoDir)
	err = v.driver.Delete(v.ctx, repoDir)
	if err != nil {
		return err
	}

	return nil
}
