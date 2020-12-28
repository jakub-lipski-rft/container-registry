package validation

import (
	"context"
	"fmt"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
)

// ManifestListValidator ensures that a manifestlist is valid and optionally
// verifies all manifest references.
type ManifestListValidator struct {
	baseValidator
}

// NewManifestListValidator returns a new ManifestListValidator.
func NewManifestListValidator(exister ManifestExister, skipDependencyVerification bool) *ManifestListValidator {
	return &ManifestListValidator{
		baseValidator: baseValidator{
			manifestExister:            exister,
			blobStatter:                nil,
			skipDependencyVerification: skipDependencyVerification,
		},
	}
}

// Validate ensures that the manifest content is valid from the
// perspective of the registry. As a policy, the registry only tries to store
// valid content, leaving trust policies of that content up to consumers.
func (v *ManifestListValidator) Validate(ctx context.Context, mnfst *manifestlist.DeserializedManifestList) error {
	var errs distribution.ErrManifestVerification

	if mnfst.SchemaVersion != 2 {
		return fmt.Errorf("unrecognized manifest list schema version %d", mnfst.SchemaVersion)
	}

	if !v.skipDependencyVerification {
		for _, manifestDescriptor := range mnfst.References() {
			exists, err := v.manifestExister.Exists(ctx, manifestDescriptor.Digest)
			if err != nil && err != distribution.ErrBlobUnknown {
				errs = append(errs, err)
			}
			if err != nil || !exists {
				// On error here, we always append unknown blob errors.
				errs = append(errs, distribution.ErrManifestBlobUnknown{Digest: manifestDescriptor.Digest})
			}
		}
	}
	if len(errs) != 0 {
		return errs
	}

	return nil
}
