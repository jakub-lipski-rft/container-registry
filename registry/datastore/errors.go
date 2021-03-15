package datastore

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when a row is not found on the metadata database.
	ErrNotFound = errors.New("not found")
	// ErrManifestNotFound is returned when a manifest is not found on the metadata database.
	ErrManifestNotFound = fmt.Errorf("manifest %w", ErrNotFound)
	// ErrManifestReferencedInList is returned when attempting to delete a manifest referenced in at least one list.
	ErrManifestReferencedInList = errors.New("manifest referenced by manifest list")
)
