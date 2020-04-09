package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Repository struct {
	ID        int
	Name      string
	Path      string
	ParentID  sql.NullInt64
	CreatedAt time.Time
	DeletedAt sql.NullTime

	Parent *Repository
}

// Repositories is a slice of Repository pointers.
type Repositories []*Repository

type ManifestConfiguration struct {
	ID        int
	MediaType string
	Digest    string
	Size      int64
	Payload   json.RawMessage
	CreatedAt time.Time
	DeletedAt sql.NullTime
}

// ManifestConfigurations is a slice of ManifestConfiguration pointers.
type ManifestConfigurations []*ManifestConfiguration

type Manifest struct {
	ID              int
	SchemaVersion   int
	MediaType       string
	Digest          string
	ConfigurationID int
	Payload         json.RawMessage
	CreatedAt       time.Time
	MarkedAt        sql.NullTime
	DeletedAt       sql.NullTime

	Configuration *ManifestConfiguration
}

// Manifests is a slice of Manifest pointers.
type Manifests []*Manifest

type Tag struct {
	ID           int
	Name         string
	RepositoryID int
	ManifestID   int
	CreatedAt    time.Time
	UpdatedAt    sql.NullTime
	DeletedAt    sql.NullTime

	Repository *Repository
	Manifest   *Manifest
}

// Tags is a slice of Tag pointers.
type Tags []*Tag

type Layer struct {
	ID        int
	MediaType string
	Digest    string
	Size      int64
	CreatedAt time.Time
	MarkedAt  sql.NullTime
	DeletedAt sql.NullTime
}

// Layers is a slice of Layer pointers.
type Layers []*Layer

type ManifestLayer struct {
	ID         int
	ManifestID int
	LayerID    int
	CreatedAt  time.Time
	DeletedAt  sql.NullTime

	Layer    *Layer
	Manifest *Manifest
}

// ManifestLayers is a slice of ManifestLayer pointers.
type ManifestLayers []*ManifestLayer

type ManifestList struct {
	ID            int
	SchemaVersion int
	MediaType     sql.NullString
	Payload       json.RawMessage
	CreatedAt     time.Time
	MarkedAt      sql.NullTime
	DeletedAt     sql.NullTime

	Repository *Repository
}

// ManifestLists is a slice of ManifestList pointers.
type ManifestLists []*ManifestList

type ManifestListItem struct {
	ID             int
	ManifestListID int
	ManifestID     int
	CreatedAt      time.Time
	DeletedAt      sql.NullTime

	Manifest     *Manifest
	ManifestList *ManifestList
}

// ManifestListItems is a slice of ManifestListItem pointers.
type ManifestListItems []*ManifestListItem
