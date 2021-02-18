package datastore

import "errors"

// ErrNotFound is returned when a row is not found on the metadata database.
var ErrNotFound = errors.New("not found")
