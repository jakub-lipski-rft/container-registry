package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
)

// ManifestConfigurationReader is the interface that defines read operations for a manifest configuration store.
type ManifestConfigurationReader interface {
	FindAll(ctx context.Context) (models.ManifestConfigurations, error)
	FindByID(ctx context.Context, id int) (*models.ManifestConfiguration, error)
	FindByDigest(ctx context.Context, digest string) (*models.ManifestConfiguration, error)
	Count(ctx context.Context) (int, error)
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
	c := new(models.ManifestConfiguration)

	if err := row.Scan(&c.ID, &c.MediaType, &c.Digest, &c.Size, &c.Payload, &c.CreatedAt, &c.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("manifest configuration not found")
		}
		return nil, fmt.Errorf("error scaning manifest configuration: %w", err)
	}

	return c, nil
}

func scanFullManifestConfigurations(rows *sql.Rows) (models.ManifestConfigurations, error) {
	cc := make(models.ManifestConfigurations, 0)
	defer rows.Close()

	for rows.Next() {
		c := new(models.ManifestConfiguration)
		if err := rows.Scan(&c.ID, &c.MediaType, &c.Digest, &c.Size, &c.Payload, &c.CreatedAt, &c.DeletedAt); err != nil {
			return nil, fmt.Errorf("error scanning manifest configuration: %w", err)
		}
		cc = append(cc, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning manifest configurations: %w", err)
	}

	return cc, nil
}

// FindByID finds a manifest configuration by ID.
func (s *manifestConfigurationStore) FindByID(ctx context.Context, id int) (*models.ManifestConfiguration, error) {
	q := `SELECT id, media_type, digest, size, payload, created_at, deleted_at
		FROM manifest_configurations WHERE id = $1`
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullManifestConfiguration(row)
}

// FindByDigest finds a manifest configuration by the digest.
func (s *manifestConfigurationStore) FindByDigest(ctx context.Context, digest string) (*models.ManifestConfiguration, error) {
	q := `SELECT id, media_type, digest, size, payload, created_at, deleted_at
		FROM manifest_configurations WHERE digest = $1`
	row := s.db.QueryRowContext(ctx, q, digest)

	return scanFullManifestConfiguration(row)
}

// FindAll finds all manifest configurations.
func (s *manifestConfigurationStore) FindAll(ctx context.Context) ([]*models.ManifestConfiguration, error) {
	q := "SELECT id, media_type, digest, size, payload, created_at, deleted_at FROM manifest_configurations"
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

// Create saves a new manifest configuration.
func (s *manifestConfigurationStore) Create(ctx context.Context, c *models.ManifestConfiguration) error {
	q := `INSERT INTO manifest_configurations (media_type, digest, size, payload)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at`

	row := s.db.QueryRowContext(ctx, q, c.MediaType, c.Digest, c.Size, c.Payload)
	if err := row.Scan(&c.ID, &c.CreatedAt); err != nil {
		return fmt.Errorf("error creating manifest configuration: %w", err)
	}

	return nil
}

// Update updates an existing manifest configuration.
func (s *manifestConfigurationStore) Update(ctx context.Context, c *models.ManifestConfiguration) error {
	q := "UPDATE manifest_configurations SET (media_type, digest, size, payload) = ($1, $2, $3, $4) WHERE id = $5"

	res, err := s.db.ExecContext(ctx, q, c.MediaType, c.Digest, c.Size, c.Payload, c.ID)
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
