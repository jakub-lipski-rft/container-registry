package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	dcontext "github.com/docker/distribution/context"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/opencontainers/go-digest"
)

// GCOpts contains options for garbage collector
type GCOpts struct {
	DryRun         bool
	RemoveUntagged bool
}

// ManifestDel contains manifest structure which will be deleted
type ManifestDel struct {
	Name   string
	Digest digest.Digest
	Tags   []string
}

// syncManifestDelContainer provides thead-safe appends of ManifestDel
type syncManifestDelContainer struct {
	sync.Mutex
	manifestDels []ManifestDel
}

func (c *syncManifestDelContainer) append(md ManifestDel) {
	c.Lock()
	defer c.Unlock()

	c.manifestDels = append(c.manifestDels, md)
}

// syncDigestSet provides thread-safe set operations on digests.
type syncDigestSet struct {
	sync.Mutex
	members map[digest.Digest]struct{}
}

func newSyncDigestSet() syncDigestSet {
	return syncDigestSet{sync.Mutex{}, make(map[digest.Digest]struct{})}
}

// idempotently adds a digest to the set.
func (s *syncDigestSet) add(d digest.Digest) {
	s.Lock()
	defer s.Unlock()

	s.members[d] = struct{}{}
}

// contains reports the digest's membership within the set.
func (s *syncDigestSet) contains(d digest.Digest) bool {
	s.Lock()
	defer s.Unlock()

	_, ok := s.members[d]

	return ok
}

// len returns the number of members within the set.
func (s *syncDigestSet) len() int {
	s.Lock()
	defer s.Unlock()

	return len(s.members)
}

// MarkAndSweep performs a mark and sweep of registry data
func MarkAndSweep(ctx context.Context, storageDriver driver.StorageDriver, registry distribution.Namespace, opts GCOpts) error {
	repositoryEnumerator, ok := registry.(distribution.RepositoryEnumerator)
	if !ok {
		return fmt.Errorf("unable to convert Namespace to RepositoryEnumerator")
	}

	// mark
	markStart := time.Now()
	dcontext.GetLogger(ctx).Info("starting mark stage")

	markSet := newSyncDigestSet()
	manifestArr := syncManifestDelContainer{sync.Mutex{}, make([]ManifestDel, 0)}

	err := repositoryEnumerator.Enumerate(ctx, func(repoName string) error {
		dcontext.GetLoggerWithField(ctx, "repo", repoName).Info("marking repository")

		var err error
		named, err := reference.WithName(repoName)
		if err != nil {
			return fmt.Errorf("failed to parse repo name %s: %v", repoName, err)
		}
		repository, err := registry.Repository(ctx, named)
		if err != nil {
			return fmt.Errorf("failed to construct repository: %v", err)
		}

		manifestService, err := repository.Manifests(ctx)
		if err != nil {
			return fmt.Errorf("failed to construct manifest service: %v", err)
		}

		manifestEnumerator, ok := manifestService.(distribution.ManifestEnumerator)
		if !ok {
			return fmt.Errorf("unable to convert ManifestService into ManifestEnumerator")
		}

		t, ok := repository.Tags(ctx).(*tagStore)
		if !ok {
			return fmt.Errorf("unable to convert tagService into tagStore")
		}

		cachedTagStore := newCachedTagStore(t)

		// Since we're removing untagged images, retrieving all tags primes the
		// cache. This isn't strictly necessary, but it prevents a potentially large
		// amount of goroutines being spawned only to wait for priming to complete
		// and allows us to report the number of primed tags.
		if opts.RemoveUntagged {
			primeStart := time.Now()
			dcontext.GetLoggerWithField(ctx, "repo", repoName).Info("priming tags cache")

			allTags, err := cachedTagStore.All(ctx)
			if err != nil {
				switch err := err.(type) {
				case distribution.ErrRepositoryUnknown:
					// Ignore path not found error on missing tags folder
				default:
					return fmt.Errorf("failed to retrieve tags %v", err)
				}
			}

			dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{
				"repo":        repoName,
				"tags_primed": len(allTags),
				"duration_s":  time.Since(primeStart).Seconds(),
			}).Info("tags cache primed")
		}

		err = manifestEnumerator.Enumerate(ctx, func(dgst digest.Digest) error {
			if opts.RemoveUntagged {
				// fetch all tags where this manifest is the latest one
				tags, err := cachedTagStore.Lookup(ctx, distribution.Descriptor{Digest: dgst})
				if err != nil {
					return fmt.Errorf("failed to retrieve tags for digest %v: %v", dgst, err)
				}
				if len(tags) == 0 {
					dcontext.GetLoggerWithField(ctx, "digest", dgst).Infof("manifest eligible for deletion")
					// fetch all tags from repository
					// all of these tags could contain manifest in history
					// which means that we need check (and delete) those references when deleting manifest
					allTags, err := cachedTagStore.All(ctx)
					if err != nil {
						switch err := err.(type) {
						case distribution.ErrRepositoryUnknown:
							// Ignore path not found error on missing tags folder
						default:
							return fmt.Errorf("failed to retrieve tags %v", err)
						}
					}
					manifestArr.append(ManifestDel{Name: repoName, Digest: dgst, Tags: allTags})
					return nil
				}
			}
			// Mark the manifest's blob
			dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{
				"digest_type": "manifest",
				"digest":      dgst,
				"repository":  repoName,
			}).Info("marking manifest")
			markSet.add(dgst)

			manifest, err := manifestService.Get(ctx, dgst)
			if err != nil {
				return fmt.Errorf("failed to retrieve manifest for digest %v: %v", dgst, err)
			}

			descriptors := manifest.References()
			for _, descriptor := range descriptors {
				dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{
					"digest_type": "layer",
					"digest":      descriptor.Digest,
					"repository":  repoName,
				}).Info("marking manifest")
				markSet.add(descriptor.Digest)
			}

			return nil
		})

		if err != nil {
			// In certain situations such as unfinished uploads, deleting all
			// tags in S3 or removing the _manifests folder manually, this
			// error may be of type PathNotFound.
			//
			// In these cases we can continue marking other manifests safely.
			//
			// If we encounter a MultiError, check each underlying error, returning
			// nil only if all errors are of type PathNotFound.
			switch err := err.(type) {
			case driver.MultiError:
				for _, e := range err {
					if _, ok := e.(driver.PathNotFoundError); !ok {
						return err
					}
				}
			case driver.PathNotFoundError:
			default:
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to mark: %v", err)
	}

	blobService := registry.Blobs()
	deleteSet := newSyncDigestSet()

	dcontext.GetLogger(ctx).Info("finding blobs eligible for deletion. This may take some time...")

	sizeChan := make(chan int64)
	sizeDone := make(chan struct{})
	var totalSizeBytes int64
	go func() {
		for size := range sizeChan {
			totalSizeBytes += size
		}
		sizeDone <- struct{}{}
	}()

	err = blobService.Enumerate(ctx, func(desc distribution.Descriptor) error {
		// check if digest is in markSet. If not, delete it!
		if !markSet.contains(desc.Digest) {
			dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{
				"digest":     desc.Digest,
				"size_bytes": desc.Size,
			}).Info("blob eligible for deletion")

			sizeChan <- desc.Size
			deleteSet.add(desc.Digest)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error enumerating blobs: %v", err)
	}

	close(sizeChan)
	<-sizeDone

	dcontext.GetLoggerWithFields(ctx, map[interface{}]interface{}{
		"blobs_marked":               markSet.len(),
		"blobs_to_delete":            deleteSet.len(),
		"manifests_to_delete":        len(manifestArr.manifestDels),
		"storage_use_estimate_bytes": totalSizeBytes,
		"duration_s":                 time.Since(markStart).Seconds(),
	}).Info("mark stage complete")

	// sweep
	if opts.DryRun {
		return nil
	}
	sweepStart := time.Now()
	dcontext.GetLogger(ctx).Info("starting sweep stage")

	vacuum := NewVacuum(ctx, storageDriver)

	if len(manifestArr.manifestDels) > 0 {
		if err := vacuum.RemoveManifests(manifestArr.manifestDels); err != nil {
			return fmt.Errorf("failed to delete manifests: %v", err)
		}
	}

	// Lock and unlock manually and access members directly to reduce lock operations.
	deleteSet.Lock()
	defer deleteSet.Unlock()

	dgsts := make([]digest.Digest, 0, len(deleteSet.members))
	for dgst := range deleteSet.members {
		dgsts = append(dgsts, dgst)
	}
	if len(dgsts) > 0 {
		if err := vacuum.RemoveBlobs(dgsts); err != nil {
			return fmt.Errorf("failed to delete blobs: %v", err)
		}
	}
	dcontext.GetLoggerWithField(ctx, "duration_s", time.Since(sweepStart).Seconds()).Info("sweep stage complete")

	return err
}
