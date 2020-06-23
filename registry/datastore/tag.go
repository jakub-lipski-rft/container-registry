package datastore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
)

// TagReader is the interface that defines read operations for a tag store.
type TagReader interface {
	FindAll(ctx context.Context) (models.Tags, error)
	FindByID(ctx context.Context, id int64) (*models.Tag, error)
	Count(ctx context.Context) (int, error)
	Repository(ctx context.Context, t *models.Tag) (*models.Repository, error)
	Manifest(ctx context.Context, t *models.Tag) (*models.Manifest, error)
	ManifestList(ctx context.Context, t *models.Tag) (*models.ManifestList, error)
}

// TagWriter is the interface that defines write operations for a tag store.
type TagWriter interface {
	Create(ctx context.Context, t *models.Tag) error
	Update(ctx context.Context, t *models.Tag) error
	Delete(ctx context.Context, id int64) error
}

// TagStore is the interface that a tag store should conform to.
type TagStore interface {
	TagReader
	TagWriter
}

// tagStore is the concrete implementation of a TagStore.
type tagStore struct {
	db Queryer
}

// NewTagStore builds a new tag store.
func NewTagStore(db Queryer) *tagStore {
	return &tagStore{db: db}
}

func scanFullTag(row *sql.Row) (*models.Tag, error) {
	t := new(models.Tag)

	if err := row.Scan(&t.ID, &t.Name, &t.RepositoryID, &t.ManifestID, &t.ManifestListID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("error scaning tag: %w", err)
		}
		return nil, nil
	}

	return t, nil
}

func scanFullTags(rows *sql.Rows) (models.Tags, error) {
	tt := make(models.Tags, 0)
	defer rows.Close()

	for rows.Next() {
		t := new(models.Tag)
		if err := rows.Scan(&t.ID, &t.Name, &t.RepositoryID, &t.ManifestID, &t.ManifestListID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning tag: %w", err)
		}
		tt = append(tt, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning tags: %w", err)
	}

	return tt, nil
}

// FindByID finds a Tag by ID.
func (s *tagStore) FindByID(ctx context.Context, id int64) (*models.Tag, error) {
	q := "SELECT id, name, repository_id, manifest_id, manifest_list_id, created_at, updated_at FROM tags WHERE id = $1"
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullTag(row)
}

// FindAll finds all tags.
func (s *tagStore) FindAll(ctx context.Context) (models.Tags, error) {
	q := "SELECT id, name, repository_id, manifest_id, manifest_list_id, created_at, updated_at FROM tags"
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error finding tags: %w", err)
	}

	return scanFullTags(rows)
}

// Count counts all tags.
func (s *tagStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM tags"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting tags: %w", err)
	}

	return count, nil
}

// Repository finds a tag repository.
func (s *tagStore) Repository(ctx context.Context, t *models.Tag) (*models.Repository, error) {
	q := "SELECT id, name, path, parent_id, created_at FROM repositories WHERE id = $1"
	row := s.db.QueryRowContext(ctx, q, t.RepositoryID)

	return scanFullRepository(row)
}

// Manifest finds a tag manifest. A tag can be associated with either a manifest or a manifest list.
func (s *tagStore) Manifest(ctx context.Context, t *models.Tag) (*models.Manifest, error) {
	q := `SELECT id, schema_version, media_type, digest_hex, payload, created_at, marked_at
		FROM manifests WHERE id = $1`

	row := s.db.QueryRowContext(ctx, q, t.ManifestID)

	return scanFullManifest(row)
}

// ManifestList finds a tag manifest list. A tag can be associated with either a manifest or a manifest list.
func (s *tagStore) ManifestList(ctx context.Context, t *models.Tag) (*models.ManifestList, error) {
	q := `SELECT id, schema_version, media_type, digest_hex, payload, created_at, marked_at
		FROM manifest_lists WHERE id = $1`

	row := s.db.QueryRowContext(ctx, q, t.ManifestListID)

	return scanFullManifestList(row)
}

// Create saves a new Tag.
func (s *tagStore) Create(ctx context.Context, t *models.Tag) error {
	q := `INSERT INTO tags (name, repository_id, manifest_id, manifest_list_id) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	row := s.db.QueryRowContext(ctx, q, t.Name, t.RepositoryID, t.ManifestID, t.ManifestListID)
	if err := row.Scan(&t.ID, &t.CreatedAt); err != nil {
		return fmt.Errorf("error creating tag: %w", err)
	}

	return nil
}

// Update updates an existing Tag.
func (s *tagStore) Update(ctx context.Context, t *models.Tag) error {
	q := "UPDATE tags SET (name, repository_id, manifest_id, manifest_list_id) = ($1, $2, $3, $4) WHERE id = $5"

	res, err := s.db.ExecContext(ctx, q, t.Name, t.RepositoryID, t.ManifestID, t.ManifestListID, t.ID)
	if err != nil {
		return fmt.Errorf("error updating tag: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error updating tag: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("tag not found")
	}

	return nil
}

// Delete deletes a Tag.
func (s *tagStore) Delete(ctx context.Context, id int64) error {
	q := "DELETE FROM tags WHERE id = $1"

	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error deleting tag: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error deleting tag: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("tag not found")
	}

	return nil
}
