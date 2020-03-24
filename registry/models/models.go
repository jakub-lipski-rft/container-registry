package models

import (
	"encoding/json"
	"time"
)

type Repository struct {
	ID        int
	Name      string
	Path      string
	ParentID  *int
	CreatedAt time.Time
	DeletedAt *time.Time

	Parent *Repository
}

type ManifestConfiguration struct {
	ID        int
	MediaType string
	Digest    string
	Size      int64
	Payload   json.RawMessage
	CreatedAt time.Time
	DeletedAt *time.Time
}

type Manifest struct {
	ID              int
	RepositoryID    int
	SchemaVersion   int
	MediaType       string
	Digest          string
	ConfigurationID int
	Payload         json.RawMessage
	CreatedAt       time.Time
	MarkedAt        *time.Time
	DeletedAt       *time.Time

	Configuration *ManifestConfiguration
	Repository    *Repository
}

type Tag struct {
	ID         int
	Name       string
	ManifestID int
	CreatedAt  time.Time
	UpdatedAt  *time.Time
	DeletedAt  *time.Time

	Manifest *Manifest
}

type Layer struct {
	ID        int
	MediaType string
	Digest    string
	Size      int64
	CreatedAt time.Time
	MarkedAt  *time.Time
	DeletedAt *time.Time
}

type ManifestLayer struct {
	ID         int
	ManifestID int
	LayerID    int
	CreatedAt  time.Time
	MarkedAt   *time.Time
	DeletedAt  *time.Time

	Layer    *Layer
	Manifest *Manifest
}

type ManifestList struct {
	ID            int
	RepositoryID  int
	SchemaVersion int
	MediaType     *string
	Payload       json.RawMessage
	CreatedAt     time.Time
	MarkedAt      *time.Time
	DeletedAt     *time.Time

	Repository *Repository
}

type ManifestListItem struct {
	ID             int
	ManifestListID int
	ManifestID     int
	CreatedAt      time.Time
	DeletedAt      *time.Time

	Manifest     *Manifest
	ManifestList *ManifestList
}
