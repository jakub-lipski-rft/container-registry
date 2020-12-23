package validation

import (
	"context"
	"fmt"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/ocischema"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

// OCIValidator ensures that a OCI image manifest is valid and optionally
// verifies all manifest references.
type OCIValidator struct {
	ManifestExister            ManifestExister
	BlobStatter                distribution.BlobStatter
	SkipDependencyVerification bool
	ManifestURLs               ManifestURLs
}

// Validate ensures that the manifest content is valid from the
// perspective of the registry. As a policy, the registry only tries to store
// valid content, leaving trust policies of that content up to consumers.
func (v *OCIValidator) Validate(ctx context.Context, mnfst *ocischema.DeserializedManifest) error {
	var errs distribution.ErrManifestVerification

	if mnfst.Manifest.SchemaVersion != 2 {
		return fmt.Errorf("unrecognized manifest schema version %d", mnfst.Manifest.SchemaVersion)
	}

	if v.SkipDependencyVerification {
		return nil
	}

	for _, descriptor := range mnfst.References() {
		var err error

		switch descriptor.MediaType {
		case v1.MediaTypeImageLayer, v1.MediaTypeImageLayerGzip, v1.MediaTypeImageLayerNonDistributable, v1.MediaTypeImageLayerNonDistributableGzip:
			for _, u := range descriptor.URLs {
				if !validURL(u, v.ManifestURLs) {
					err = errInvalidURL
					break
				}
			}

			if err == nil && len(descriptor.URLs) == 0 {
				// If no URLs, require that the blob exists
				_, err = v.BlobStatter.Stat(ctx, descriptor.Digest)
			}
		case v1.MediaTypeImageManifest:
			var exists bool
			exists, err = v.ManifestExister.Exists(ctx, descriptor.Digest)
			if err != nil || !exists {
				err = distribution.ErrBlobUnknown // just coerce to unknown.
			}

			fallthrough // double check the blob store.
		default:
			// forward all else to blob storage
			if len(descriptor.URLs) == 0 {
				_, err = v.BlobStatter.Stat(ctx, descriptor.Digest)
			}
		}

		if err != nil {
			if err != distribution.ErrBlobUnknown {
				errs = append(errs, err)
			}

			// On error here, we always append unknown blob errors.
			errs = append(errs, distribution.ErrManifestBlobUnknown{Digest: descriptor.Digest})
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}
