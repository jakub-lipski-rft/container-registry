package storage

import (
	"context"
	"fmt"
	"time"

	dcontext "github.com/docker/distribution/context"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/opencontainers/go-digest"
)

func emit(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
}

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

// MarkAndSweep performs a mark and sweep of registry data
func MarkAndSweep(ctx context.Context, storageDriver driver.StorageDriver, registry distribution.Namespace, opts GCOpts) error {
	repositoryEnumerator, ok := registry.(distribution.RepositoryEnumerator)
	if !ok {
		return fmt.Errorf("unable to convert Namespace to RepositoryEnumerator")
	}

	// mark
	markSet := make(map[digest.Digest]struct{})
	manifestArr := make([]ManifestDel, 0)
	err := repositoryEnumerator.Enumerate(ctx, func(repoName string) error {
		emit(repoName)

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

		err = manifestEnumerator.Enumerate(ctx, func(dgst digest.Digest) error {
			if opts.RemoveUntagged {
				// fetch all tags where this manifest is the latest one
				tags, err := repository.Tags(ctx).Lookup(ctx, distribution.Descriptor{Digest: dgst})
				if err != nil {
					return fmt.Errorf("failed to retrieve tags for digest %v: %v", dgst, err)
				}
				if len(tags) == 0 {
					emit("manifest eligible for deletion: %s", dgst)
					// fetch all tags from repository
					// all of these tags could contain manifest in history
					// which means that we need check (and delete) those references when deleting manifest
					allTags, err := repository.Tags(ctx).All(ctx)
					if err != nil {
						return fmt.Errorf("failed to retrieve tags %v", err)
					}
					manifestArr = append(manifestArr, ManifestDel{Name: repoName, Digest: dgst, Tags: allTags})
					return nil
				}
			}
			// Mark the manifest's blob
			emit("%s: marking manifest %s ", repoName, dgst)
			markSet[dgst] = struct{}{}

			manifest, err := manifestService.Get(ctx, dgst)
			if err != nil {
				return fmt.Errorf("failed to retrieve manifest for digest %v: %v", dgst, err)
			}

			descriptors := manifest.References()
			for _, descriptor := range descriptors {
				markSet[descriptor.Digest] = struct{}{}
				emit("%s: marking blob %s", repoName, descriptor.Digest)
			}

			return nil
		})

		if err != nil {
			// In certain situations such as unfinished uploads, deleting all
			// tags in S3 or removing the _manifests folder manually, this
			// error may be of type PathNotFound.
			//
			// In these cases we can continue marking other manifests safely.
			if _, ok := err.(driver.PathNotFoundError); ok {
				return nil
			}
		}

		return err
	})

	if err != nil {
		return fmt.Errorf("failed to mark: %v", err)
	}

	blobService := registry.Blobs()
	deleteSet := make(map[digest.Digest]struct{})
	err = blobService.Enumerate(ctx, func(dgst digest.Digest) error {
		// check if digest is in markSet. If not, delete it!
		if _, ok := markSet[dgst]; !ok {
			deleteSet[dgst] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error enumerating blobs: %v", err)
	}
	emit("\n%d blobs marked, %d blobs and %d manifests eligible for deletion", len(markSet), len(deleteSet), len(manifestArr))

	// sweep
	sweepStart := time.Now()
	dcontext.GetLogger(ctx).Info("starting sweep stage")

	vacuum := NewVacuum(ctx, storageDriver)
	if !opts.DryRun && len(manifestArr) > 0 {
		if err := vacuum.RemoveManifests(manifestArr); err != nil {
			return fmt.Errorf("failed to delete manifests: %v", err)
		}
	}

	dgsts := make([]digest.Digest, 0, len(deleteSet))
	for dgst := range deleteSet {
		dgsts = append(dgsts, dgst)
	}
	if !opts.DryRun && len(dgsts) > 0 {
		if err := vacuum.RemoveBlobs(dgsts); err != nil {
			return fmt.Errorf("failed to delete blobs: %v", err)
		}
	}
	dcontext.GetLoggerWithField(ctx, "duration_s", time.Since(sweepStart).Seconds()).Info("sweep stage complete")

	return err
}
