package validation

import (
	"context"
	"errors"
	"regexp"

	"net/url"

	digest "github.com/opencontainers/go-digest"
)

var (
	errUnexpectedURL = errors.New("unexpected URL on layer")
	errMissingURL    = errors.New("missing URL on layer")
	errInvalidURL    = errors.New("invalid URL on layer")
)

// ManifestExister checks for the existance of a manifest.
type ManifestExister interface {
	// Exists returns true if the manifest exists.
	Exists(ctx context.Context, dgst digest.Digest) (bool, error)
}

// ManifestURLs holds regular expressions for controlling manifest URL allowlisting
type ManifestURLs struct {
	Allow *regexp.Regexp
	Deny  *regexp.Regexp
}

func validURL(s string, u ManifestURLs) bool {
	pu, err := url.Parse(s)
	if err != nil || (pu.Scheme != "http" && pu.Scheme != "https") || pu.Fragment != "" || (u.Allow != nil && !u.Allow.MatchString(s)) || (u.Deny != nil && u.Deny.MatchString(s)) {
		return false
	}

	return true
}
