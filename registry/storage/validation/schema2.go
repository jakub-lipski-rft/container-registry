package validation

import (
	"context"
	"fmt"
	"net/url"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
)

// Schema2Validator ensures that a schema2 manifest is valid and optionally
// verifies all manifest references.
type Schema2Validator struct {
	ManifestExister            ManifestExister
	BlobStatter                distribution.BlobStatter
	SkipDependencyVerification bool
	ManifestURLs               ManifestURLs
}

// Validate ensures that the manifest content is valid from the
// perspective of the registry. As a policy, the registry only tries to store
// valid content, leaving trust policies of that content up to consumers.
func (v *Schema2Validator) Validate(ctx context.Context, mnfst *schema2.DeserializedManifest) error {
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
		case schema2.MediaTypeForeignLayer:
			// Clients download this layer from an external URL, so do not check for
			// its presence.
			if len(descriptor.URLs) == 0 {
				err = errMissingURL
			}
			allow := v.ManifestURLs.Allow
			deny := v.ManifestURLs.Deny
			for _, u := range descriptor.URLs {
				var pu *url.URL
				pu, err = url.Parse(u)
				if err != nil || (pu.Scheme != "http" && pu.Scheme != "https") || pu.Fragment != "" || (allow != nil && !allow.MatchString(u)) || (deny != nil && deny.MatchString(u)) {
					err = errInvalidURL
					break
				}
			}
		case schema2.MediaTypeManifest, schema1.MediaTypeManifest:
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
