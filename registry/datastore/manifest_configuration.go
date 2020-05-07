package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
	"github.com/opencontainers/go-digest"
)

// ManifestConfigurationReader is the interface that defines read operations for a manifest configuration store.
type ManifestConfigurationReader interface {
	FindAll(ctx context.Context) (models.ManifestConfigurations, error)
	FindByID(ctx context.Context, id int) (*models.ManifestConfiguration, error)
	FindByDigest(ctx context.Context, d digest.Digest) (*models.ManifestConfiguration, error)
	Count(ctx context.Context) (int, error)
	Manifest(ctx context.Context, c *models.ManifestConfiguration) (*models.Manifest, error)
}

// ManifestConfigurationWriter is the interface that defines write operations for a manifest configuration store.
type ManifestConfigurationWriter interface {
	Create(ctx context.Context, c *models.ManifestConfiguration) error
	Update(ctx context.Context, c *models.ManifestConfiguration) error
	SoftDelete(ctx context.Context, c *models.ManifestConfiguration) error
	Delete(ctx context.Context, id int) error
}

// ManifestConfigurationStore is the interface that a manifest configuration store should conform to.
type ManifestConfigurationStore interface {
	ManifestConfigurationReader
	ManifestConfigurationWriter
}

// manifestConfigurationStore is the concrete implementation of a ManifestConfigurationStore.
type manifestConfigurationStore struct {
	db Queryer
}

// NewManifestConfigurationStore builds a new repository store.
func NewManifestConfigurationStore(db Queryer) *manifestConfigurationStore {
	return &manifestConfigurationStore{db: db}
}

func scanFullManifestConfiguration(row *sql.Row) (*models.ManifestConfiguration, error) {
	var digestHex []byte
	c := new(models.ManifestConfiguration)
	err := row.Scan(&c.ID, &c.ManifestID, &c.MediaType, &digestHex, &c.Size, &c.Payload, &c.CreatedAt, &c.DeletedAt)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("error scaning manifest configuration: %w", err)
		}
		return nil, nil
	}
	c.Digest = digest.NewDigestFromBytes(digest.SHA256, digestHex)

	return c, nil
}

func scanFullManifestConfigurations(rows *sql.Rows) (models.ManifestConfigurations, error) {
	cc := make(models.ManifestConfigurations, 0)
	defer rows.Close()

	for rows.Next() {
		var digestHex []byte
		c := new(models.ManifestConfiguration)
		err := rows.Scan(&c.ID, &c.ManifestID, &c.MediaType, &digestHex, &c.Size, &c.Payload, &c.CreatedAt, &c.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning manifest configuration: %w", err)
		}
		c.Digest = digest.NewDigestFromBytes(digest.SHA256, digestHex)
		cc = append(cc, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning manifest configurations: %w", err)
	}

	return cc, nil
}

// FindByID finds a manifest configuration by ID.
func (s *manifestConfigurationStore) FindByID(ctx context.Context, id int) (*models.ManifestConfiguration, error) {
	q := `SELECT id, manifest_id, media_type, digest_hex, size, payload, created_at, deleted_at
		FROM manifest_configurations WHERE id = $1`
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullManifestConfiguration(row)
}

// FindByDigest finds a manifest configuration by the digest.
func (s *manifestConfigurationStore) FindByDigest(ctx context.Context, d digest.Digest) (*models.ManifestConfiguration, error) {
	q := `SELECT id, manifest_id, media_type, digest_hex, size, payload, created_at, deleted_at
		FROM manifest_configurations WHERE digest_hex = decode($1, 'hex')`
	row := s.db.QueryRowContext(ctx, q, d.Hex())

	return scanFullManifestConfiguration(row)
}

// FindAll finds all manifest configurations.
func (s *manifestConfigurationStore) FindAll(ctx context.Context) ([]*models.ManifestConfiguration, error) {
	q := "SELECT id, manifest_id, media_type, digest_hex, size, payload, created_at, deleted_at FROM manifest_configurations"
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error finding manifest configurations: %w", err)
	}

	return scanFullManifestConfigurations(rows)
}

// Count counts all manifest configurations.
func (s *manifestConfigurationStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM manifest_configurations"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting manifest configurations: %w", err)
	}

	return count, nil
}

// Manifest finds the manifest that the configuration belongs to.
func (s *manifestConfigurationStore) Manifest(ctx context.Context, c *models.ManifestConfiguration) (*models.Manifest, error) {
	q := `SELECT id, schema_version, media_type, digest_hex, payload, created_at, marked_at, deleted_at
		FROM manifests WHERE id = $1`

	row := s.db.QueryRowContext(ctx, q, c.ManifestID)

	return scanFullManifest(row)
}

// Create saves a new manifest configuration.
func (s *manifestConfigurationStore) Create(ctx context.Context, c *models.ManifestConfiguration) error {
	q := `INSERT INTO manifest_configurations (manifest_id, media_type, digest_hex, size, payload)
		VALUES ($1, $2, decode($3, 'hex'), $4, $5) RETURNING id, created_at`

	row := s.db.QueryRowContext(ctx, q, c.ManifestID, c.MediaType, c.Digest.Hex(), c.Size, c.Payload)
	if err := row.Scan(&c.ID, &c.CreatedAt); err != nil {
		return fmt.Errorf("error creating manifest configuration: %w", err)
	}

	return nil
}

// Update updates an existing manifest configuration.
func (s *manifestConfigurationStore) Update(ctx context.Context, c *models.ManifestConfiguration) error {
	q := `UPDATE manifest_configurations
		SET (manifest_id, media_type, digest_hex, size, payload) = ($1, $2, decode($3, 'hex'), $4, $5) WHERE id = $6`

	res, err := s.db.ExecContext(ctx, q, c.ManifestID, c.MediaType, c.Digest.Hex(), c.Size, c.Payload, c.ID)
	if err != nil {
		return fmt.Errorf("error updating manifest configuration: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error updating manifest configuration: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("manifest configuration not found")
	}

	return nil
}

// SoftDelete soft deletes a manifest configuration.
func (s *manifestConfigurationStore) SoftDelete(ctx context.Context, c *models.ManifestConfiguration) error {
	q := "UPDATE manifest_configurations SET deleted_at = NOW() WHERE id = $1 RETURNING deleted_at"

	if err := s.db.QueryRowContext(ctx, q, c.ID).Scan(&c.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("manifest configuration not found")
		}
		return fmt.Errorf("error soft deleting manifest configuration: %w", err)
	}

	return nil
}

// Delete deletes a manifest configuration.
func (s *manifestConfigurationStore) Delete(ctx context.Context, id int) error {
	q := "DELETE FROM manifest_configurations WHERE id = $1"

	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error deleting manifest configuration: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error deleting manifest configuration: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("manifest configuration not found")
	}

	return nil
}
