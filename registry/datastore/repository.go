package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
)

// RepositoryReader is the interface that defines read operations for a repository store.
type RepositoryReader interface {
	FindAll(ctx context.Context) (models.Repositories, error)
	FindByID(ctx context.Context, id int) (*models.Repository, error)
	FindByPath(ctx context.Context, path string) (*models.Repository, error)
	FindDescendantsOf(ctx context.Context, id int) (models.Repositories, error)
	FindAncestorsOf(ctx context.Context, id int) (models.Repositories, error)
	FindSiblingsOf(ctx context.Context, id int) (models.Repositories, error)
	Count(ctx context.Context) (int, error)
	Tags(ctx context.Context, r *models.Repository) (models.Tags, error)
	ManifestTags(ctx context.Context, r *models.Repository, m *models.Manifest) (models.Tags, error)
}

// RepositoryWriter is the interface that defines write operations for a repository store.
type RepositoryWriter interface {
	Create(ctx context.Context, r *models.Repository) error
	Update(ctx context.Context, r *models.Repository) error
	SoftDelete(ctx context.Context, r *models.Repository) error
	Delete(ctx context.Context, id int) error
}

// RepositoryStore is the interface that a repository store should conform to.
type RepositoryStore interface {
	RepositoryReader
	RepositoryWriter
}

// repositoryStore is the concrete implementation of a RepositoryStore.
type repositoryStore struct {
	// db can be either a *sql.DB or *sql.Tx
	db Queryer
}

// NewRepositoryStore builds a new repositoryStore.
func NewRepositoryStore(db Queryer) *repositoryStore {
	return &repositoryStore{db: db}
}

func scanFullRepository(row *sql.Row) (*models.Repository, error) {
	r := new(models.Repository)

	if err := row.Scan(&r.ID, &r.Name, &r.Path, &r.ParentID, &r.CreatedAt, &r.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("repository not found")
		}
		return nil, fmt.Errorf("error scanning repository: %w", err)
	}

	return r, nil
}

func scanFullRepositories(rows *sql.Rows) (models.Repositories, error) {
	rr := make(models.Repositories, 0)
	defer rows.Close()

	for rows.Next() {
		r := new(models.Repository)
		if err := rows.Scan(&r.ID, &r.Name, &r.Path, &r.ParentID, &r.CreatedAt, &r.DeletedAt); err != nil {
			return nil, fmt.Errorf("error scanning repository: %w", err)
		}
		rr = append(rr, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning repositories: %w", err)
	}

	return rr, nil
}

// FindByID finds a repository by ID.
func (s *repositoryStore) FindByID(ctx context.Context, id int) (*models.Repository, error) {
	q := "SELECT id, name, path, parent_id, created_at, deleted_at FROM repositories WHERE id = $1"
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullRepository(row)
}

// FindByPath finds a repository by path.
func (s *repositoryStore) FindByPath(ctx context.Context, path string) (*models.Repository, error) {
	q := "SELECT id, name, path, parent_id, created_at, deleted_at FROM repositories WHERE path = $1"
	row := s.db.QueryRowContext(ctx, q, path)

	return scanFullRepository(row)
}

// FindAll finds all repositories.
func (s *repositoryStore) FindAll(ctx context.Context) (models.Repositories, error) {
	q := "SELECT id, name, path, parent_id, created_at, deleted_at FROM repositories"
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error finding repositories: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindDescendantsOf finds all descendants of a given repository.
func (s *repositoryStore) FindDescendantsOf(ctx context.Context, id int) (models.Repositories, error) {
	q := `WITH RECURSIVE descendants AS (
		SELECT id, name, path, parent_id, created_at, deleted_at FROM repositories WHERE id = $1
		UNION ALL
		SELECT repositories.* FROM repositories
		JOIN descendants ON descendants.id = repositories.parent_id
		) SELECT * FROM descendants WHERE descendants.id != $1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("error finding descendants of repository: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindAncestorsOf finds all ancestors of a given repository.
func (s *repositoryStore) FindAncestorsOf(ctx context.Context, id int) (models.Repositories, error) {
	q := `WITH RECURSIVE ancestors AS (
		SELECT id, name, path, parent_id, created_at, deleted_at FROM repositories  WHERE id = $1
		UNION ALL
		SELECT repositories.* FROM repositories
		JOIN ancestors ON ancestors.parent_id = repositories.id
		) SELECT * FROM ancestors WHERE ancestors.id != $1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("error finding ancestors of repository: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindSiblingsOf finds all siblings of a given repository.
func (s *repositoryStore) FindSiblingsOf(ctx context.Context, id int) (models.Repositories, error) {
	q := `SELECT siblings.id, siblings.name, siblings.path, siblings.parent_id, siblings.created_at, siblings.deleted_at
		FROM repositories siblings
		LEFT JOIN repositories anchor ON siblings.parent_id = anchor.parent_id
		WHERE anchor.id = $1 AND siblings.id != $1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("error finding siblings of repository: %w", err)
	}

	return scanFullRepositories(rows)
}

// Tags finds all tags of a given repository.
func (s *repositoryStore) Tags(ctx context.Context, r *models.Repository) (models.Tags, error) {
	q := `SELECT id, name, repository_id, manifest_id, created_at, updated_at, deleted_at
		FROM tags WHERE repository_id = $1`

	rows, err := s.db.QueryContext(ctx, q, r.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding tags: %w", err)
	}

	return scanFullTags(rows)
}

// ManifestTags finds all tags of a given repository manifest.
func (s *repositoryStore) ManifestTags(ctx context.Context, r *models.Repository, m *models.Manifest) (models.Tags, error) {
	q := `SELECT id, name, repository_id, manifest_id, created_at, updated_at, deleted_at
		FROM tags WHERE repository_id = $1 AND manifest_id = $2`

	rows, err := s.db.QueryContext(ctx, q, r.ID, m.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding tags: %w", err)
	}

	return scanFullTags(rows)
}

// Count counts all repositories.
func (s *repositoryStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM repositories"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting repositories: %w", err)
	}

	return count, nil
}

// Create saves a new repository.
func (s *repositoryStore) Create(ctx context.Context, r *models.Repository) error {
	q := "INSERT INTO repositories (name, path, parent_id) VALUES ($1, $2, $3) RETURNING id, created_at"

	row := s.db.QueryRowContext(ctx, q, r.Name, r.Path, r.ParentID)
	if err := row.Scan(&r.ID, &r.CreatedAt); err != nil {
		return fmt.Errorf("error creating repository: %w", err)
	}

	return nil
}

// Update updates an existing repository.
func (s *repositoryStore) Update(ctx context.Context, r *models.Repository) error {
	q := "UPDATE repositories SET (name, path, parent_id) = ($1, $2, $3) WHERE id = $4"

	res, err := s.db.ExecContext(ctx, q, r.Name, r.Path, r.ParentID, r.ID)
	if err != nil {
		return fmt.Errorf("error updating repository: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error updating repository: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("repository not found")
	}

	return nil
}

// SoftDelete soft deletes a repository.
func (s *repositoryStore) SoftDelete(ctx context.Context, r *models.Repository) error {
	q := "UPDATE repositories SET deleted_at = NOW() WHERE id = $1 RETURNING deleted_at"

	if err := s.db.QueryRowContext(ctx, q, r.ID).Scan(&r.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("repository not found")
		}
		return fmt.Errorf("error soft deleting repository: %w", err)
	}

	return nil
}

// Delete deletes a repository.
func (s *repositoryStore) Delete(ctx context.Context, id int) error {
	q := "DELETE FROM repositories WHERE id = $1"

	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("repository not found")
	}

	return nil
}
