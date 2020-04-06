package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
)

// TagReader is the interface that defines read operations for a tag store.
type TagReader interface {
	FindAll(ctx context.Context) (models.Tags, error)
	FindByID(ctx context.Context, id int) (*models.Tag, error)
	FindByNameAndManifestID(ctx context.Context, name string, manifestID int) (*models.Tag, error)
	Count(ctx context.Context) (int, error)
}

// TagWriter is the interface that defines write operations for a tag store.
type TagWriter interface {
	Create(ctx context.Context, t *models.Tag) error
	Update(ctx context.Context, t *models.Tag) error
	Mark(ctx context.Context, t *models.Tag) error
	SoftDelete(ctx context.Context, t *models.Tag) error
	Delete(ctx context.Context, id int) error
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

	if err := row.Scan(&t.ID, &t.Name, &t.ManifestID, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("error scaning tag: %w", err)
	}

	return t, nil
}

func scanFullTags(rows *sql.Rows) (models.Tags, error) {
	tt := make(models.Tags, 0)
	defer rows.Close()

	for rows.Next() {
		t := new(models.Tag)
		if err := rows.Scan(&t.ID, &t.Name, &t.ManifestID, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt); err != nil {
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
func (s *tagStore) FindByID(ctx context.Context, id int) (*models.Tag, error) {
	q := "SELECT id, name, manifest_id, created_at, updated_at, deleted_at FROM tags WHERE id = $1"
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullTag(row)
}

// FindByName finds a Tag by name and manifest ID.
func (s *tagStore) FindByNameAndManifestID(ctx context.Context, name string, manifestID int) (*models.Tag, error) {
	q := "SELECT id, name, manifest_id, created_at, updated_at, deleted_at FROM tags WHERE name = $1 AND manifest_id = $2"
	row := s.db.QueryRowContext(ctx, q, name, manifestID)

	return scanFullTag(row)
}

// FindAll finds all tags.
func (s *tagStore) FindAll(ctx context.Context) (models.Tags, error) {
	q := "SELECT id, name, manifest_id, created_at, updated_at, deleted_at FROM tags"
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

// Create saves a new Tag.
func (s *tagStore) Create(ctx context.Context, t *models.Tag) error {
	q := "INSERT INTO tags (name, manifest_id) VALUES ($1, $2) RETURNING id, created_at"

	row := s.db.QueryRowContext(ctx, q, t.Name, t.ManifestID)
	if err := row.Scan(&t.ID, &t.CreatedAt); err != nil {
		return fmt.Errorf("error creating tag: %w", err)
	}

	return nil
}

// Update updates an existing Tag.
func (s *tagStore) Update(ctx context.Context, t *models.Tag) error {
	q := "UPDATE tags SET (name, manifest_id) = ($1, $2) WHERE id = $3"

	res, err := s.db.ExecContext(ctx, q, t.Name, t.ManifestID, t.ID)
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

// SoftDelete soft deletes a Tag.
func (s *tagStore) SoftDelete(ctx context.Context, t *models.Tag) error {
	q := "UPDATE tags SET deleted_at = NOW() WHERE id = $1 RETURNING deleted_at"

	if err := s.db.QueryRowContext(ctx, q, t.ID).Scan(&t.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("tag not found")
		}
		return fmt.Errorf("error soft deleting tag: %w", err)
	}

	return nil
}

// Delete deletes a Tag.
func (s *tagStore) Delete(ctx context.Context, id int) error {
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
