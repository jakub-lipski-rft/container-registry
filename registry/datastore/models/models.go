package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/opencontainers/go-digest"
)

type Repository struct {
	ID        int64
	Name      string
	Path      string
	ParentID  sql.NullInt64
	CreatedAt time.Time
	UpdatedAt sql.NullTime
}

// Repositories is a slice of Repository pointers.
type Repositories []*Repository

type ManifestConfiguration struct {
	ID         int64
	ManifestID int64
	BlobID     int64
	MediaType  string
	Digest     digest.Digest
	Size       int64
	Payload    json.RawMessage
	CreatedAt  time.Time
}

// ManifestConfigurations is a slice of ManifestConfiguration pointers.
type ManifestConfigurations []*ManifestConfiguration

type Manifest struct {
	ID            int64
	SchemaVersion int
	MediaType     string
	Digest        digest.Digest
	Payload       json.RawMessage
	CreatedAt     time.Time
	MarkedAt      sql.NullTime
}

// Manifests is a slice of Manifest pointers.
type Manifests []*Manifest

type Tag struct {
	ID           int64
	Name         string
	RepositoryID int64
	// a tag can be associated with either a manifest or a manifest list
	ManifestID     sql.NullInt64
	ManifestListID sql.NullInt64
	CreatedAt      time.Time
	UpdatedAt      sql.NullTime
}

// Tags is a slice of Tag pointers.
type Tags []*Tag

type Blob struct {
	ID        int64
	MediaType string
	Digest    digest.Digest
	Size      int64
	CreatedAt time.Time
	MarkedAt  sql.NullTime
}

// Blobs is a slice of Blob pointers.
type Blobs []*Blob

type ManifestList struct {
	ID            int64
	SchemaVersion int
	MediaType     string
	Digest        digest.Digest
	Payload       json.RawMessage
	CreatedAt     time.Time
	MarkedAt      sql.NullTime

	Repository *Repository
}

// ManifestLists is a slice of ManifestList pointers.
type ManifestLists []*ManifestList
