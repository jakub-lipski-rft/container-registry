package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
	"github.com/opencontainers/go-digest"
)

// BlobReader is the interface that defines read operations for a blob store.
type BlobReader interface {
	FindAll(ctx context.Context) (models.Blobs, error)
	FindByID(ctx context.Context, id int64) (*models.Blob, error)
	FindByDigest(ctx context.Context, d digest.Digest) (*models.Blob, error)
	Count(ctx context.Context) (int, error)
}

// BlobWriter is the interface that defines write operations for a blob store.
type BlobWriter interface {
	Create(ctx context.Context, b *models.Blob) error
	CreateOrFind(ctx context.Context, b *models.Blob) error
	UpdateMediaType(ctx context.Context, b *models.Blob) error
	Mark(ctx context.Context, b *models.Blob) error
	Delete(ctx context.Context, id int64) error
}

// BlobStore is the interface that a blob store should conform to.
type BlobStore interface {
	BlobReader
	BlobWriter
}

// blobStore is the concrete implementation of a BlobStore.
type blobStore struct {
	db Queryer
}

// NewBlobStore builds a new blobStore.
func NewBlobStore(db Queryer) *blobStore {
	return &blobStore{db: db}
}

func scanFullBlob(row *sql.Row) (*models.Blob, error) {
	var digestAlgorithm DigestAlgorithm
	var digestHex []byte
	b := new(models.Blob)

	if err := row.Scan(&b.ID, &b.MediaType, &digestAlgorithm, &digestHex, &b.Size, &b.CreatedAt, &b.MarkedAt); err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("error scanning blob: %w", err)
		}
		return nil, nil
	}

	alg, err := digestAlgorithm.Parse()
	if err != nil {
		return nil, err
	}
	b.Digest = digest.NewDigestFromBytes(alg, digestHex)

	return b, nil
}

func scanFullBlobs(rows *sql.Rows) (models.Blobs, error) {
	bb := make(models.Blobs, 0)
	defer rows.Close()

	for rows.Next() {
		var digestAlgorithm DigestAlgorithm
		var digestHex []byte
		b := new(models.Blob)

		err := rows.Scan(&b.ID, &b.MediaType, &digestAlgorithm, &digestHex, &b.Size, &b.CreatedAt, &b.MarkedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning blob: %w", err)
		}

		alg, err := digestAlgorithm.Parse()
		if err != nil {
			return nil, err
		}
		b.Digest = digest.NewDigestFromBytes(alg, digestHex)

		bb = append(bb, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning blobs: %w", err)
	}

	return bb, nil
}

// FindByID finds a blob by ID.
func (s *blobStore) FindByID(ctx context.Context, id int64) (*models.Blob, error) {
	q := "SELECT id, media_type, digest_algorithm, digest_hex, size, created_at, marked_at FROM blobs WHERE id = $1"
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullBlob(row)
}

// FindByDigest finds a blob by digest.
func (s *blobStore) FindByDigest(ctx context.Context, d digest.Digest) (*models.Blob, error) {
	q := `SELECT id, media_type, digest_algorithm, digest_hex, size, created_at, marked_at
		FROM blobs WHERE digest_algorithm = $1 AND digest_hex = decode($2, 'hex')`

	alg, err := NewDigestAlgorithm(d.Algorithm())
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, q, alg, d.Hex())

	return scanFullBlob(row)
}

// FindAll finds all blobs.
func (s *blobStore) FindAll(ctx context.Context) (models.Blobs, error) {
	q := "SELECT id, media_type, digest_algorithm, digest_hex, size, created_at, marked_at FROM blobs"
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error finding blobs: %w", err)
	}

	return scanFullBlobs(rows)
}

// Count counts all blobs.
func (s *blobStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM blobs"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting blobs: %w", err)
	}

	return count, nil
}

// Create saves a new blob.
func (s *blobStore) Create(ctx context.Context, b *models.Blob) error {
	q := `INSERT INTO blobs (media_type, digest_algorithm, digest_hex, size) VALUES ($1, $2, decode($3, 'hex'), $4)
		RETURNING id, created_at`

	digestAlgorithm, err := NewDigestAlgorithm(b.Digest.Algorithm())
	if err != nil {
		return err
	}
	row := s.db.QueryRowContext(ctx, q, b.MediaType, digestAlgorithm, b.Digest.Hex(), b.Size)
	if err := row.Scan(&b.ID, &b.CreatedAt); err != nil {
		return fmt.Errorf("error creating blob: %w", err)
	}

	return nil
}

// CreateOrFind attempts to create a blob. If the blob already exists (same digest_hex) that record is loaded from the
// database into b. This is similar to a FindByDigest followed by a Create, but without being prone to race conditions
// on write operations between the corresponding read (FindByDigest) and write (Create) operations. Separate Find* and
// Create method calls should be preferred to this when race conditions are not a concern.
func (s *blobStore) CreateOrFind(ctx context.Context, b *models.Blob) error {
	q := `INSERT INTO blobs (media_type, digest_algorithm, digest_hex, size)
		VALUES ($1, $2, decode($3, 'hex'), $4)
		ON CONFLICT (digest_algorithm, digest_hex) DO NOTHING
		RETURNING id, created_at`

	digestAlgorithm, err := NewDigestAlgorithm(b.Digest.Algorithm())
	if err != nil {
		return err
	}
	row := s.db.QueryRowContext(ctx, q, b.MediaType, digestAlgorithm, b.Digest.Hex(), b.Size)

	if err := row.Scan(&b.ID, &b.CreatedAt); err != nil {
		if err != sql.ErrNoRows {
			return fmt.Errorf("creating blob: %w", err)
		}
		// if the result set has no rows, then the blob already exists
		tmp, err := s.FindByDigest(ctx, b.Digest)
		if err != nil {
			return err
		}
		*b = *tmp
	}

	return nil
}

// UpdateMediaType updates an existing blob media type. Blobs are usually received and stored with a generic media type
// of application/octet-stream. Once we know their concrete media type we can update them.
func (s *blobStore) UpdateMediaType(ctx context.Context, b *models.Blob) error {
	q := "UPDATE blobs SET media_type = $1 WHERE id = $2"

	res, err := s.db.ExecContext(ctx, q, b.MediaType, b.ID)
	if err != nil {
		return fmt.Errorf("error updating blob media type: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error updating blob media type: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("blob not found")
	}

	return nil
}

// Mark marks a blob during garbage collection.
func (s *blobStore) Mark(ctx context.Context, b *models.Blob) error {
	q := "UPDATE blobs SET marked_at = NOW() WHERE id = $1 RETURNING marked_at"

	if err := s.db.QueryRowContext(ctx, q, b.ID).Scan(&b.MarkedAt); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("blob not found")
		}
		return fmt.Errorf("error soft deleting blobs: %w", err)
	}

	return nil
}

// Delete deletes a blob.
func (s *blobStore) Delete(ctx context.Context, id int64) error {
	q := "DELETE FROM blobs WHERE id = $1"

	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error deleting blob: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error deleting blob: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("blob not found")
	}

	return nil
}
