package datastore

import (
	"fmt"
	"strconv"

	"github.com/opencontainers/go-digest"
)

// DigestAlgorithm is the database representation of a digest algorithm, stored as a smallint.
type DigestAlgorithm int

const (
	Unknown DigestAlgorithm = iota // 0
	SHA256                         // 1
	SHA512                         // 2
)

// String implements the Stringer interface.
func (alg DigestAlgorithm) String() string {
	return strconv.Itoa(int(alg))
}

// NewDigestAlgorithm maps a digest.Digest to a DigestAlgorithm.
func NewDigestAlgorithm(alg digest.Algorithm) (DigestAlgorithm, error) {
	switch alg {
	case digest.SHA256:
		return SHA256, nil
	case digest.SHA512:
		return SHA512, nil
	default:
		return Unknown, fmt.Errorf("unknown digest algorithm %q", alg)
	}
}

// Parse maps a DigestAlgorithm to a digest.Digest.
func (alg DigestAlgorithm) Parse() (digest.Algorithm, error) {
	switch alg {
	case SHA256:
		return digest.SHA256, nil
	case SHA512:
		return digest.SHA512, nil
	default:
		return "", fmt.Errorf("unknown digest algorithm %q", alg)
	}
}
