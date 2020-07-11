package datastore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
	"github.com/opencontainers/go-digest"
)

// ConfigurationReader is the interface that defines read operations for a configuration store.
type ConfigurationReader interface {
	FindAll(ctx context.Context) (models.Configurations, error)
	FindByID(ctx context.Context, id int64) (*models.Configuration, error)
	FindByDigest(ctx context.Context, d digest.Digest) (*models.Configuration, error)
	Count(ctx context.Context) (int, error)
	Manifest(ctx context.Context, c *models.Configuration) (*models.Manifest, error)
}

// ConfigurationWriter is the interface that defines write operations for a configuration store.
type ConfigurationWriter interface {
	Create(ctx context.Context, c *models.Configuration) error
	Update(ctx context.Context, c *models.Configuration) error
	Delete(ctx context.Context, id int64) error
}

// ConfigurationStore is the interface that a configuration store should conform to.
type ConfigurationStore interface {
	ConfigurationReader
	ConfigurationWriter
}

// configurationStore is the concrete implementation of a ConfigurationStore.
type configurationStore struct {
	db Queryer
}

// NewConfigurationStore builds a new repository store.
func NewConfigurationStore(db Queryer) *configurationStore {
	return &configurationStore{db: db}
}

func scanFullConfiguration(row *sql.Row) (*models.Configuration, error) {
	var digestAlgorithm DigestAlgorithm
	var digestHex []byte
	c := new(models.Configuration)
	err := row.Scan(&c.ID, &c.ManifestID, &c.BlobID, &c.MediaType, &digestAlgorithm, &digestHex, &c.Size, &c.Payload, &c.CreatedAt)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("error scaning configuration: %w", err)
		}
		return nil, nil
	}

	alg, err := digestAlgorithm.Parse()
	if err != nil {
		return nil, err
	}
	c.Digest = digest.NewDigestFromBytes(alg, digestHex)

	return c, nil
}

func scanFullConfigurations(rows *sql.Rows) (models.Configurations, error) {
	cc := make(models.Configurations, 0)
	defer rows.Close()

	for rows.Next() {
		var digestAlgorithm DigestAlgorithm
		var digestHex []byte
		c := new(models.Configuration)
		err := rows.Scan(&c.ID, &c.ManifestID, &c.BlobID, &c.MediaType, &digestAlgorithm, &digestHex, &c.Size, &c.Payload, &c.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning configuration: %w", err)
		}

		alg, err := digestAlgorithm.Parse()
		if err != nil {
			return nil, err
		}
		c.Digest = digest.NewDigestFromBytes(alg, digestHex)

		cc = append(cc, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning configurations: %w", err)
	}

	return cc, nil
}

// FindByID finds a configuration by ID.
func (s *configurationStore) FindByID(ctx context.Context, id int64) (*models.Configuration, error) {
	q := `SELECT c.id, c.manifest_id, c.blob_id, b.media_type, b.digest_algorithm, b.digest_hex, b.size, c.payload, c.created_at
		FROM configurations AS c
		JOIN blobs AS b ON c.blob_id = b.id
		WHERE c.id = $1`
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullConfiguration(row)
}

// FindByDigest finds a configuration by the digest.
func (s *configurationStore) FindByDigest(ctx context.Context, d digest.Digest) (*models.Configuration, error) {
	q := `SELECT c.id, c.manifest_id, c.blob_id, b.media_type, b.digest_algorithm, b.digest_hex, b.size, c.payload, c.created_at
		FROM configurations AS c
		JOIN blobs AS b ON c.blob_id = b.id
		WHERE b.digest_algorithm = $1 AND b.digest_hex = decode($2, 'hex')`

	alg, err := NewDigestAlgorithm(d.Algorithm())
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, q, alg, d.Hex())

	return scanFullConfiguration(row)
}

// FindAll finds all configurations.
func (s *configurationStore) FindAll(ctx context.Context) (models.Configurations, error) {
	q := `SELECT c.id, c.manifest_id, c.blob_id, b.media_type, b.digest_algorithm, b.digest_hex, b.size, c.payload, c.created_at
		FROM configurations AS c
		JOIN blobs AS b ON c.blob_id = b.id`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error finding configurations: %w", err)
	}

	return scanFullConfigurations(rows)
}

// Count counts all configurations.
func (s *configurationStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM configurations"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting configurations: %w", err)
	}

	return count, nil
}

// Manifest finds the manifest that the configuration belongs to.
func (s *configurationStore) Manifest(ctx context.Context, c *models.Configuration) (*models.Manifest, error) {
	q := `SELECT id, schema_version, media_type, digest_algorithm, digest_hex, payload, created_at, marked_at
		FROM manifests WHERE id = $1`

	row := s.db.QueryRowContext(ctx, q, c.ManifestID)

	return scanFullManifest(row)
}

// Create saves a new configuration.
func (s *configurationStore) Create(ctx context.Context, c *models.Configuration) error {
	q := `INSERT INTO configurations (manifest_id, blob_id, payload)
		VALUES ($1, $2, $3) RETURNING id, created_at`

	row := s.db.QueryRowContext(ctx, q, c.ManifestID, c.BlobID, c.Payload)
	if err := row.Scan(&c.ID, &c.CreatedAt); err != nil {
		return fmt.Errorf("error creating configuration: %w", err)
	}

	return nil
}

// Update updates an existing configuration.
func (s *configurationStore) Update(ctx context.Context, c *models.Configuration) error {
	q := `UPDATE configurations
		SET (manifest_id, blob_id, payload) = ($1, $2, $3) WHERE id = $4`

	res, err := s.db.ExecContext(ctx, q, c.ManifestID, c.BlobID, c.Payload, c.ID)
	if err != nil {
		return fmt.Errorf("error updating configuration: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error updating configuration: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("configuration not found")
	}

	return nil
}

// Delete deletes a configuration.
func (s *configurationStore) Delete(ctx context.Context, id int64) error {
	q := "DELETE FROM configurations WHERE id = $1"

	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error deleting configuration: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error deleting configuration: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("configuration not found")
	}

	return nil
}
