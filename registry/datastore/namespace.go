package datastore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/docker/distribution/registry/datastore/metrics"
	"github.com/docker/distribution/registry/datastore/models"
)

// NamespaceReader is the interface that defines read operations for a namespace store.
type NamespaceReader interface {
	FindByName(ctx context.Context, name string) (*models.Namespace, error)
}

// NamespaceWriter is the interface that defines write operations for a namespace store.
type NamespaceWriter interface {
	CreateOrFind(ctx context.Context, r *models.Namespace) error
}

// NamespaceStore is the interface that a namespace store should conform to.
type NamespaceStore interface {
	NamespaceReader
	NamespaceWriter
}

// namespaceStore is the concrete implementation of a NamespaceStore.
type namespaceStore struct {
	// db can be either a *sql.DB or *sql.Tx
	db Queryer
}

// NewNamespaceStore builds a new repositoryStore.
func NewNamespaceStore(db Queryer) *namespaceStore {
	return &namespaceStore{db: db}
}

func scanFullNamespace(row *sql.Row) (*models.Namespace, error) {
	n := new(models.Namespace)

	if err := row.Scan(&n.ID, &n.Name, &n.CreatedAt, &n.UpdatedAt); err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("scanning namespace: %w", err)
		}
		return nil, nil
	}

	return n, nil
}

// FindByName finds a namespace by name.
func (s *namespaceStore) FindByName(ctx context.Context, name string) (*models.Namespace, error) {
	defer metrics.InstrumentQuery("namespace_find_by_name")()
	q := `SELECT
			id,
			name,
			created_at,
			updated_at
		FROM
			top_level_namespaces
		WHERE
			name = $1`
	row := s.db.QueryRowContext(ctx, q, name)

	return scanFullNamespace(row)
}

// CreateOrFind attempts to create a namespace. If the namespace already exists (same name) that record is loaded from
// the database into n. This is similar to a FindByName followed by a Create, but without being prone to race conditions
// on write operations between the corresponding read (FindByName) and write (Create) operations. Separate Find* and
// Create method calls should be preferred to this when race conditions are not a concern.
func (s *namespaceStore) CreateOrFind(ctx context.Context, n *models.Namespace) error {
	defer metrics.InstrumentQuery("namespace_create_or_find")()
	q := `INSERT INTO top_level_namespaces (name)
			VALUES ($1)
		ON CONFLICT (name)
			DO NOTHING
		RETURNING
			id, created_at`

	row := s.db.QueryRowContext(ctx, q, n.Name)
	if err := row.Scan(&n.ID, &n.CreatedAt); err != nil {
		if err != sql.ErrNoRows {
			return fmt.Errorf("creating namespace: %w", err)
		}
		// if the result set has no rows, then the namespace already exists
		tmp, err := s.FindByName(ctx, n.Name)
		if err != nil {
			return err
		}
		*n = *tmp
	}

	return nil
}
