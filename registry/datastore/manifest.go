package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
)

// ManifestReader is the interface that defines read operations for a Manifest store.
type ManifestReader interface {
	FindAll(ctx context.Context) (models.Manifests, error)
	FindByID(ctx context.Context, id int) (*models.Manifest, error)
	FindByDigest(ctx context.Context, digest string) (*models.Manifest, error)
	Count(ctx context.Context) (int, error)
	Layers(ctx context.Context, m *models.Manifest) (models.Layers, error)
	Lists(ctx context.Context, m *models.Manifest) (models.ManifestLists, error)
	Repositories(ctx context.Context, m *models.Manifest) (models.Repositories, error)
}

// ManifestWriter is the interface that defines write operations for a Manifest store.
type ManifestWriter interface {
	Create(ctx context.Context, m *models.Manifest) error
	Update(ctx context.Context, m *models.Manifest) error
	Mark(ctx context.Context, m *models.Manifest) error
	AssociateLayer(ctx context.Context, m *models.Manifest, l *models.Layer) error
	DissociateLayer(ctx context.Context, m *models.Manifest, l *models.Layer) error
	SoftDelete(ctx context.Context, m *models.Manifest) error
	Delete(ctx context.Context, id int) error
}

// ManifestStore is the interface that a Manifest store should conform to.
type ManifestStore interface {
	ManifestReader
	ManifestWriter
}

// manifestStore is the concrete implementation of a ManifestStore.
type manifestStore struct {
	db Queryer
}

// NewManifestStore builds a new manifest store.
func NewManifestStore(db Queryer) *manifestStore {
	return &manifestStore{db: db}
}

func scanFullManifest(row *sql.Row) (*models.Manifest, error) {
	m := new(models.Manifest)

	err := row.Scan(&m.ID, &m.SchemaVersion, &m.MediaType, &m.Digest, &m.ConfigurationID, &m.Payload,
		&m.CreatedAt, &m.MarkedAt, &m.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("manifest not found")
		}
		return nil, fmt.Errorf("error scaning manifest: %w", err)
	}

	return m, nil
}

func scanFullManifests(rows *sql.Rows) (models.Manifests, error) {
	mm := make(models.Manifests, 0)
	defer rows.Close()

	for rows.Next() {
		m := new(models.Manifest)

		err := rows.Scan(&m.ID, &m.SchemaVersion, &m.MediaType, &m.Digest, &m.ConfigurationID,
			&m.Payload, &m.CreatedAt, &m.MarkedAt, &m.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning manifest: %w", err)
		}
		mm = append(mm, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning manifests: %w", err)
	}

	return mm, nil
}

// FindByID finds a Manifest by ID.
func (s *manifestStore) FindByID(ctx context.Context, id int) (*models.Manifest, error) {
	q := `SELECT id, schema_version, media_type, digest, configuration_id, payload, created_at, marked_at, deleted_at
		FROM manifests WHERE id = $1`

	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullManifest(row)
}

// FindByDigest finds a Manifest by the digest.
func (s *manifestStore) FindByDigest(ctx context.Context, digest string) (*models.Manifest, error) {
	q := `SELECT id, schema_version, media_type, digest, configuration_id, payload, created_at, marked_at, deleted_at
		FROM manifests WHERE digest = $1`

	row := s.db.QueryRowContext(ctx, q, digest)

	return scanFullManifest(row)
}

// FindAll finds all manifests.
func (s *manifestStore) FindAll(ctx context.Context) (models.Manifests, error) {
	q := `SELECT id, schema_version, media_type, digest, configuration_id, payload, created_at, marked_at, deleted_at
		FROM manifests`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error finding manifests: %w", err)
	}

	return scanFullManifests(rows)
}

// Count counts all manifests.
func (s *manifestStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM manifests"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting manifests: %w", err)
	}

	return count, nil
}

// Layers finds layers associated with a manifest, through the ManifestLayer relationship entity.
func (s *manifestStore) Layers(ctx context.Context, m *models.Manifest) (models.Layers, error) {
	q := `SELECT l.id, l.media_type, l.digest, l.size, l.created_at, l.marked_at, l.deleted_at FROM layers as l
		JOIN manifest_layers as ml ON ml.layer_id = l.id
		JOIN manifests as m ON m.id = ml.manifest_id
		WHERE m.id = $1`

	rows, err := s.db.QueryContext(ctx, q, m.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding layers: %w", err)
	}

	return scanFullLayers(rows)
}

// Lists finds all manifest lists which reference a manifest, through the ManifestListItem relationship entity.
func (s *manifestStore) Lists(ctx context.Context, m *models.Manifest) (models.ManifestLists, error) {
	q := `SELECT ml.id, ml.schema_version, ml.media_type, ml.payload, ml.created_at,
		ml.marked_at, ml.deleted_at FROM manifest_lists as ml
		JOIN manifest_list_items as mli ON mli.manifest_list_id = ml.id
		JOIN manifests as m ON m.id = mli.manifest_id
		WHERE m.id = $1`

	rows, err := s.db.QueryContext(ctx, q, m.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding manifest lists: %w", err)
	}

	return scanFullManifestLists(rows)
}

// Repositories finds all repositories which reference a manifest.
func (s *manifestStore) Repositories(ctx context.Context, m *models.Manifest) (models.Repositories, error) {
	q := `SELECT r.id, r.name, r.path, r.parent_id, r.created_at, r.deleted_at FROM repositories as r
		JOIN repository_manifests as rm ON rm.repository_id = r.id
		JOIN manifests as m ON m.id = rm.manifest_id
		WHERE m.id = $1`

	rows, err := s.db.QueryContext(ctx, q, m.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding repositories: %w", err)
	}

	return scanFullRepositories(rows)
}

// Create saves a new Manifest.
func (s *manifestStore) Create(ctx context.Context, m *models.Manifest) error {
	q := `INSERT INTO manifests (schema_version, media_type, digest, configuration_id, payload)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`

	row := s.db.QueryRowContext(ctx, q, m.SchemaVersion, m.MediaType, m.Digest, m.ConfigurationID, m.Payload)
	if err := row.Scan(&m.ID, &m.CreatedAt); err != nil {
		return fmt.Errorf("error creating manifest: %w", err)
	}

	return nil
}

// Update updates an existing Manifest.
func (s *manifestStore) Update(ctx context.Context, m *models.Manifest) error {
	q := `UPDATE manifests
		SET (schema_version, media_type, digest, configuration_id, payload) = ($1, $2, $3, $4, $5)
		WHERE id = $6`

	res, err := s.db.ExecContext(ctx, q, m.SchemaVersion, m.MediaType, m.Digest, m.ConfigurationID, m.Payload, m.ID)
	if err != nil {
		return fmt.Errorf("error updating manifest: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error updating manifest: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("manifest not found")
	}

	return nil
}

// Mark marks a Manifest during garbage collection.
func (s *manifestStore) Mark(ctx context.Context, m *models.Manifest) error {
	q := "UPDATE manifests SET marked_at = NOW() WHERE id = $1 RETURNING marked_at"

	if err := s.db.QueryRowContext(ctx, q, m.ID).Scan(&m.MarkedAt); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("manifest not found")
		}
		return fmt.Errorf("error soft deleting manifest: %w", err)
	}

	return nil
}

// AssociateLayer associates a layer and a manifest.
func (s *manifestStore) AssociateLayer(ctx context.Context, m *models.Manifest, l *models.Layer) error {
	q := "INSERT INTO manifest_layers (manifest_id, layer_id) VALUES ($1, $2)"

	if _, err := s.db.ExecContext(ctx, q, m.ID, l.ID); err != nil {
		return fmt.Errorf("error associating layer: %w", err)
	}

	return nil
}

// DissociateLayer dissociates a layer and a manifest.
func (s *manifestStore) DissociateLayer(ctx context.Context, m *models.Manifest, l *models.Layer) error {
	q := "DELETE FROM manifest_layers WHERE manifest_id = $1 AND layer_id = $2"

	res, err := s.db.ExecContext(ctx, q, m.ID, l.ID)
	if err != nil {
		return fmt.Errorf("error dissociating layer: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error dissociating layer: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("layer association not found")
	}

	return nil
}

// SoftDelete soft deletes a Manifest.
func (s *manifestStore) SoftDelete(ctx context.Context, m *models.Manifest) error {
	q := "UPDATE manifests SET deleted_at = NOW() WHERE id = $1 RETURNING deleted_at"

	if err := s.db.QueryRowContext(ctx, q, m.ID).Scan(&m.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("manifest not found")
		}
		return fmt.Errorf("error soft deleting manifest: %w", err)
	}

	return nil
}

// Delete deletes a Manifest.
func (s *manifestStore) Delete(ctx context.Context, id int) error {
	q := "DELETE FROM manifests WHERE id = $1"

	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error deleting manifest: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error deleting manifest: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("manifest not found")
	}

	return nil
}
