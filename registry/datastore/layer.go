package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
)

// LayerReader is the interface that defines read operations for a layer store.
type LayerReader interface {
	FindAll(ctx context.Context) (models.Layers, error)
	FindByID(ctx context.Context, id int) (*models.Layer, error)
	FindByDigest(ctx context.Context, digest string) (*models.Layer, error)
	Count(ctx context.Context) (int, error)
	Manifests(ctx context.Context, l *models.Layer) (models.Manifests, error)
}

// LayerWriter is the interface that defines write operations for a layer store.
type LayerWriter interface {
	Create(ctx context.Context, l *models.Layer) error
	Update(ctx context.Context, l *models.Layer) error
	Mark(ctx context.Context, l *models.Layer) error
	SoftDelete(ctx context.Context, l *models.Layer) error
	Delete(ctx context.Context, id int) error
}

// LayerStore is the interface that a layer store should conform to.
type LayerStore interface {
	LayerReader
	LayerWriter
}

// layerStore is the concrete implementation of a LayerStore.
type layerStore struct {
	db Queryer
}

// NewLayerStore builds a new layerStore.
func NewLayerStore(db Queryer) *layerStore {
	return &layerStore{db: db}
}

func scanFullLayer(row *sql.Row) (*models.Layer, error) {
	l := new(models.Layer)

	if err := row.Scan(&l.ID, &l.MediaType, &l.Digest, &l.Size, &l.CreatedAt, &l.MarkedAt, &l.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("layer not found")
		}
		return nil, fmt.Errorf("error scanning layer: %w", err)
	}

	return l, nil
}

func scanFullLayers(rows *sql.Rows) (models.Layers, error) {
	ll := make(models.Layers, 0)
	defer rows.Close()

	for rows.Next() {
		l := new(models.Layer)
		if err := rows.Scan(&l.ID, &l.MediaType, &l.Digest, &l.Size, &l.CreatedAt, &l.MarkedAt, &l.DeletedAt); err != nil {
			return nil, fmt.Errorf("error scanning layer: %w", err)
		}
		ll = append(ll, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning layers: %w", err)
	}

	return ll, nil
}

// FindByID finds a layer by ID.
func (s *layerStore) FindByID(ctx context.Context, id int) (*models.Layer, error) {
	q := "SELECT id, media_type, digest, size, created_at, marked_at, deleted_at FROM layers WHERE id = $1"
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullLayer(row)
}

// FindByDigest finds a layer by digest.
func (s *layerStore) FindByDigest(ctx context.Context, digest string) (*models.Layer, error) {
	q := "SELECT id, media_type, digest, size, created_at, marked_at, deleted_at FROM layers WHERE digest = $1"
	row := s.db.QueryRowContext(ctx, q, digest)

	return scanFullLayer(row)
}

// FindAll finds all layers.
func (s *layerStore) FindAll(ctx context.Context) (models.Layers, error) {
	q := "SELECT id, media_type, digest, size, created_at, marked_at, deleted_at FROM layers"
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error finding layers: %w", err)
	}

	return scanFullLayers(rows)
}

// Count counts all layers.
func (s *layerStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM layers"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting layers: %w", err)
	}

	return count, nil
}

// Manifests finds all manifests that reference a layer.
func (s *layerStore) Manifests(ctx context.Context, l *models.Layer) (models.Manifests, error) {
	q := `SELECT m.id, m.schema_version, m.media_type, m.digest, m.configuration_id,
		m.payload, m.created_at, m.marked_at, m.deleted_at FROM manifests as m
		JOIN manifest_layers as ml ON ml.manifest_id = m.id
		JOIN layers as l ON l.id = ml.layer_id
		WHERE l.id = $1`

	rows, err := s.db.QueryContext(ctx, q, l.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding manifests: %w", err)
	}

	return scanFullManifests(rows)
}

// Create saves a new layer.
func (s *layerStore) Create(ctx context.Context, l *models.Layer) error {
	q := "INSERT INTO layers (media_type, digest, size) VALUES ($1, $2, $3) RETURNING id, created_at"

	row := s.db.QueryRowContext(ctx, q, l.MediaType, l.Digest, l.Size)
	if err := row.Scan(&l.ID, &l.CreatedAt); err != nil {
		return fmt.Errorf("error creating layer: %w", err)
	}

	return nil
}

// Update updates an existing layer.
func (s *layerStore) Update(ctx context.Context, l *models.Layer) error {
	q := "UPDATE layers SET (media_type, digest, size) = ($1, $2, $3) WHERE id = $4"

	res, err := s.db.ExecContext(ctx, q, l.MediaType, l.Digest, l.Size, l.ID)
	if err != nil {
		return fmt.Errorf("error updating layer: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error updating layer: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("layer not found")
	}

	return nil
}

// Mark marks a layer during garbage collection.
func (s *layerStore) Mark(ctx context.Context, l *models.Layer) error {
	q := "UPDATE layers SET marked_at = NOW() WHERE id = $1 RETURNING marked_at"

	if err := s.db.QueryRowContext(ctx, q, l.ID).Scan(&l.MarkedAt); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("layer not found")
		}
		return fmt.Errorf("error soft deleting layers: %w", err)
	}

	return nil
}

// SoftDelete soft deletes a layer.
func (s *layerStore) SoftDelete(ctx context.Context, l *models.Layer) error {
	q := "UPDATE layers SET deleted_at = NOW() WHERE id = $1 RETURNING deleted_at"

	if err := s.db.QueryRowContext(ctx, q, l.ID).Scan(&l.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("layer not found")
		}
		return fmt.Errorf("error soft deleting layer: %w", err)
	}

	return nil
}

// Delete deletes a layer.
func (s *layerStore) Delete(ctx context.Context, id int) error {
	q := "DELETE FROM layers WHERE id = $1"

	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error deleting layer: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error deleting layer: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("layer not found")
	}

	return nil
}
