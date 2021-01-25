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

type Configuration struct {
	MediaType string
	Digest    digest.Digest
	Payload   json.RawMessage
}

type Manifest struct {
	ID            int64
	RepositoryID  int64
	SchemaVersion int
	MediaType     string
	Digest        digest.Digest
	Payload       json.RawMessage
	Configuration *Configuration
	CreatedAt     time.Time
}

// Manifests is a slice of Manifest pointers.
type Manifests []*Manifest

type Tag struct {
	ID           int64
	Name         string
	RepositoryID int64
	ManifestID   int64
	CreatedAt    time.Time
	UpdatedAt    sql.NullTime
}

// Tags is a slice of Tag pointers.
type Tags []*Tag

type Blob struct {
	MediaType string
	Digest    digest.Digest
	Size      int64
	CreatedAt time.Time
}

// Blobs is a slice of Blob pointers.
type Blobs []*Blob

// GCBlobTask represents a row in the gc_blob_review_queue table.
type GCBlobTask struct {
	ReviewAfter time.Time
	ReviewCount int
	Digest      digest.Digest
}

// GCConfigLink represents a row in the gc_blobs_configurations table.
type GCConfigLink struct {
	ID           int64
	RepositoryID int64
	ManifestID   int64
	Digest       digest.Digest
}
