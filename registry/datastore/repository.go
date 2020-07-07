package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/docker/distribution/registry/datastore/models"
)

// RepositoryReader is the interface that defines read operations for a repository store.
type RepositoryReader interface {
	FindAll(ctx context.Context) (models.Repositories, error)
	FindAllPaginated(ctx context.Context, limit int, lastPath string) (models.Repositories, error)
	FindByID(ctx context.Context, id int64) (*models.Repository, error)
	FindByPath(ctx context.Context, path string) (*models.Repository, error)
	FindDescendantsOf(ctx context.Context, id int64) (models.Repositories, error)
	FindAncestorsOf(ctx context.Context, id int64) (models.Repositories, error)
	FindSiblingsOf(ctx context.Context, id int64) (models.Repositories, error)
	Count(ctx context.Context) (int, error)
	CountAfterPath(ctx context.Context, path string) (int, error)
	Manifests(ctx context.Context, r *models.Repository) (models.Manifests, error)
	ManifestLists(ctx context.Context, r *models.Repository) (models.ManifestLists, error)
	Tags(ctx context.Context, r *models.Repository) (models.Tags, error)
	TagsPaginated(ctx context.Context, r *models.Repository, limit int, lastName string) (models.Tags, error)
	TagsCountAfterName(ctx context.Context, r *models.Repository, lastName string) (int, error)
	ManifestTags(ctx context.Context, r *models.Repository, m *models.Manifest) (models.Tags, error)
	ManifestListTags(ctx context.Context, r *models.Repository, m *models.ManifestList) (models.Tags, error)
	FindManifestByDigest(ctx context.Context, r *models.Repository, d digest.Digest) (*models.Manifest, error)
	FindManifestListByDigest(ctx context.Context, r *models.Repository, d digest.Digest) (*models.ManifestList, error)
	FindTagByName(ctx context.Context, r *models.Repository, name string) (*models.Tag, error)
	Blobs(ctx context.Context, r *models.Repository) (models.Blobs, error)
}

// RepositoryWriter is the interface that defines write operations for a repository store.
type RepositoryWriter interface {
	Create(ctx context.Context, r *models.Repository) error
	CreateByPath(ctx context.Context, path string) (*models.Repository, error)
	CreateOrFind(ctx context.Context, r *models.Repository) error
	CreateOrFindByPath(ctx context.Context, path string) (*models.Repository, error)
	Update(ctx context.Context, r *models.Repository) error
	AssociateManifest(ctx context.Context, r *models.Repository, m *models.Manifest) error
	DissociateManifest(ctx context.Context, r *models.Repository, m *models.Manifest) error
	AssociateManifestList(ctx context.Context, r *models.Repository, ml *models.ManifestList) error
	DissociateManifestList(ctx context.Context, r *models.Repository, ml *models.ManifestList) error
	UntagManifest(ctx context.Context, r *models.Repository, m *models.Manifest) error
	UntagManifestList(ctx context.Context, r *models.Repository, ml *models.ManifestList) error
	LinkBlob(ctx context.Context, r *models.Repository, b *models.Blob) error
	UnlinkBlob(ctx context.Context, r *models.Repository, b *models.Blob) error
	Delete(ctx context.Context, id int64) error
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

	if err := row.Scan(&r.ID, &r.Name, &r.Path, &r.ParentID, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("error scanning repository: %w", err)
		}
		return nil, nil
	}

	return r, nil
}

func scanFullRepositories(rows *sql.Rows) (models.Repositories, error) {
	rr := make(models.Repositories, 0)
	defer rows.Close()

	for rows.Next() {
		r := new(models.Repository)
		if err := rows.Scan(&r.ID, &r.Name, &r.Path, &r.ParentID, &r.CreatedAt, &r.UpdatedAt); err != nil {
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
func (s *repositoryStore) FindByID(ctx context.Context, id int64) (*models.Repository, error) {
	q := "SELECT id, name, path, parent_id, created_at, updated_at FROM repositories WHERE id = $1"
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullRepository(row)
}

// FindByPath finds a repository by path.
func (s *repositoryStore) FindByPath(ctx context.Context, path string) (*models.Repository, error) {
	q := "SELECT id, name, path, parent_id, created_at, updated_at FROM repositories WHERE path = $1"
	row := s.db.QueryRowContext(ctx, q, path)

	return scanFullRepository(row)
}

// FindAll finds all repositories.
func (s *repositoryStore) FindAll(ctx context.Context) (models.Repositories, error) {
	q := "SELECT id, name, path, parent_id, created_at, updated_at FROM repositories"
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error finding repositories: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindAllPaginated finds up to limit repositories with path lexicographically after lastPath. This is used exclusively
// for the GET /v2/_catalog API route, where pagination is done with a marker (lastPath). Empty repositories (which do
// not have at least a manifest or a manifest list) are ignored. Also, even if there is no repository with a path
// of lastPath, the returned repositories will always be those with a path lexicographically after lastPath. Finally,
// repositories are lexicographically sorted. These constraints exists to preserve the existing API behaviour (when
// doing a filesystem walk based pagination).
func (s *repositoryStore) FindAllPaginated(ctx context.Context, limit int, lastPath string) (models.Repositories, error) {
	q := `SELECT DISTINCT r.id, r.name, r.path, r.parent_id, r.created_at, r.updated_at
		FROM repositories AS r
	 	LEFT JOIN repository_manifests AS rm ON r.id = rm.repository_id
	 	LEFT JOIN repository_manifest_lists AS rml ON r.id = rml.repository_id
		WHERE COALESCE(rm.repository_id, rml.repository_id) IS NOT NULL
		AND r.path > $1
		ORDER BY r.path
		LIMIT $2`
	rows, err := s.db.QueryContext(ctx, q, lastPath, limit)
	if err != nil {
		return nil, fmt.Errorf("error finding repositories with pagination: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindDescendantsOf finds all descendants of a given repository.
func (s *repositoryStore) FindDescendantsOf(ctx context.Context, id int64) (models.Repositories, error) {
	q := `WITH RECURSIVE descendants AS (
		SELECT id, name, path, parent_id, created_at, updated_at FROM repositories WHERE id = $1
		UNION ALL
		SELECT r.id, r.name, r.path, r.parent_id, r.created_at, r.updated_at FROM repositories AS r
		JOIN descendants ON descendants.id = r.parent_id
		) SELECT * FROM descendants WHERE descendants.id != $1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("error finding descendants of repository: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindAncestorsOf finds all ancestors of a given repository.
func (s *repositoryStore) FindAncestorsOf(ctx context.Context, id int64) (models.Repositories, error) {
	q := `WITH RECURSIVE ancestors AS (
		SELECT id, name, path, parent_id, created_at, updated_at FROM repositories WHERE id = $1
		UNION ALL
		SELECT r.id, r.name, r.path, r.parent_id, r.created_at, r.updated_at FROM repositories AS r
		JOIN ancestors ON ancestors.parent_id = r.id
		) SELECT * FROM ancestors WHERE ancestors.id != $1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("error finding ancestors of repository: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindSiblingsOf finds all siblings of a given repository.
func (s *repositoryStore) FindSiblingsOf(ctx context.Context, id int64) (models.Repositories, error) {
	q := `SELECT siblings.id, siblings.name, siblings.path, siblings.parent_id, siblings.created_at, siblings.updated_at
		FROM repositories AS siblings
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
	q := `SELECT id, name, repository_id, manifest_id, manifest_list_id, created_at, updated_at
		FROM tags WHERE repository_id = $1`

	rows, err := s.db.QueryContext(ctx, q, r.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding tags: %w", err)
	}

	return scanFullTags(rows)
}

// TagsPaginated finds up to limit tags of a given repository with name lexicographically after lastName. This is used
// exclusively for the GET /v2/<name>/tags/list API route, where pagination is done with a marker (lastName). Even if
// there is no tag with a name of lastName, the returned tags will always be those with a path lexicographically after
// lastName. Finally, tags are lexicographically sorted. These constraints exists to preserve the existing API behaviour
// (when doing a filesystem walk based pagination).
func (s *repositoryStore) TagsPaginated(ctx context.Context, r *models.Repository, limit int, lastName string) (models.Tags, error) {
	q := `SELECT id, name, repository_id, manifest_id, manifest_list_id, created_at, updated_at
		FROM tags WHERE repository_id = $1 AND name > $2 ORDER BY name LIMIT $3`
	rows, err := s.db.QueryContext(ctx, q, r.ID, lastName, limit)
	if err != nil {
		return nil, fmt.Errorf("error finding tags with pagination: %w", err)
	}

	return scanFullTags(rows)
}

// TagsCountAfterName counts all tags of a given repository with name lexicographically after lastName. This is used
// exclusively for the GET /v2/<name>/tags/list API route, where pagination is done with a marker (lastName). Even if
// there is no tag with a name of lastName, the counted tags will always be those with a path lexicographically after
// lastName. This constraint exists to preserve the existing API behaviour (when doing a filesystem walk based
// pagination).
func (s *repositoryStore) TagsCountAfterName(ctx context.Context, r *models.Repository, lastName string) (int, error) {
	q := "SELECT COUNT(id) FROM tags WHERE repository_id = $1 AND name > $2"

	var count int
	if err := s.db.QueryRowContext(ctx, q, r.ID, lastName).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting tags lexicographically after name: %w", err)
	}

	return count, nil
}

// ManifestTags finds all tags of a given repository manifest.
func (s *repositoryStore) ManifestTags(ctx context.Context, r *models.Repository, m *models.Manifest) (models.Tags, error) {
	q := `SELECT id, name, repository_id, manifest_id, manifest_list_id, created_at, updated_at
		FROM tags WHERE repository_id = $1 AND manifest_id = $2`

	rows, err := s.db.QueryContext(ctx, q, r.ID, m.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding tags: %w", err)
	}

	return scanFullTags(rows)
}

// ManifestListTags finds all tags of a given repository manifest list.
func (s *repositoryStore) ManifestListTags(ctx context.Context, r *models.Repository, ml *models.ManifestList) (models.Tags, error) {
	q := `SELECT id, name, repository_id, manifest_id, manifest_list_id, created_at, updated_at
		FROM tags WHERE repository_id = $1 AND manifest_list_id = $2`

	rows, err := s.db.QueryContext(ctx, q, r.ID, ml.ID)
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

// CountAfterPath counts all repositories with path lexicographically after lastPath. This is used exclusively
// for the GET /v2/_catalog API route, where pagination is done with a marker (lastPath). Empty repositories (which do
// not have at least a manifest or a manifest list) are ignored. Also, even if there is no repository with a path
// of lastPath, the counted repositories will always be those with a path lexicographically after lastPath. These
// constraints exists to preserve the existing API behaviour (when doing a filesystem walk based pagination).
func (s *repositoryStore) CountAfterPath(ctx context.Context, path string) (int, error) {
	q := `SELECT COUNT(DISTINCT(r.id)) FROM repositories AS r
		LEFT JOIN repository_manifests AS rm ON r.id = rm.repository_id
	 	LEFT JOIN repository_manifest_lists AS rml ON r.id = rml.repository_id
		WHERE COALESCE(rm.repository_id, rml.repository_id) IS NOT NULL
		AND r.path > $1`

	var count int
	if err := s.db.QueryRowContext(ctx, q, path).Scan(&count); err != nil {
		return count, fmt.Errorf("error counting repositories lexicographically after path: %w", err)
	}

	return count, nil
}

// Manifests finds all manifests associated with a repository.
func (s *repositoryStore) Manifests(ctx context.Context, r *models.Repository) (models.Manifests, error) {
	q := `SELECT m.id, m.schema_version, m.media_type, m.digest_hex, m.payload, m.created_at, m.marked_at
		FROM manifests as m
		JOIN repository_manifests as rm ON rm.manifest_id = m.id
		JOIN repositories AS r ON r.id = rm.repository_id
		WHERE r.id = $1`

	rows, err := s.db.QueryContext(ctx, q, r.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding manifests: %w", err)
	}

	return scanFullManifests(rows)
}

// ManifestLists finds all manifest lists associated with a repository.
func (s *repositoryStore) ManifestLists(ctx context.Context, r *models.Repository) (models.ManifestLists, error) {
	q := `SELECT ml.id, ml.schema_version, ml.media_type, ml.digest_hex, ml.payload, ml.created_at, ml.marked_at
		FROM manifest_lists as ml
		JOIN repository_manifest_lists as rml ON rml.manifest_list_id = ml.id
		JOIN repositories AS r ON r.id = rml.repository_id
		WHERE r.id = $1`

	rows, err := s.db.QueryContext(ctx, q, r.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding manifest lists: %w", err)
	}

	return scanFullManifestLists(rows)
}

// FindManifestByDigest finds a manifest by digest within a repository.
func (s *repositoryStore) FindManifestByDigest(ctx context.Context, r *models.Repository, d digest.Digest) (*models.Manifest, error) {
	q := `SELECT m.id, m.schema_version, m.media_type, m.digest_hex, m.payload, m.created_at, m.marked_at
		FROM manifests as m
		JOIN repository_manifests as rm ON rm.manifest_id = m.id
		JOIN repositories AS r ON r.id = rm.repository_id
		WHERE r.id = $1 AND m.digest_hex = decode($2, 'hex')`

	row := s.db.QueryRowContext(ctx, q, r.ID, d.Hex())
	return scanFullManifest(row)
}

// FindManifestListByDigest finds a manifest list by digest within a repository.
func (s *repositoryStore) FindManifestListByDigest(ctx context.Context, r *models.Repository, d digest.Digest) (*models.ManifestList, error) {
	q := `SELECT ml.id, ml.schema_version, ml.media_type, ml.digest_hex, ml.payload, ml.created_at, ml.marked_at
		FROM manifest_lists as ml
		JOIN repository_manifest_lists as rml ON rml.manifest_list_id = ml.id
		JOIN repositories AS r ON r.id = rml.repository_id
		WHERE r.id = $1 AND ml.digest_hex = decode($2, 'hex')`

	row := s.db.QueryRowContext(ctx, q, r.ID, d.Hex())
	return scanFullManifestList(row)
}

// Blobs finds all blobs associated with the repository.
func (s *repositoryStore) Blobs(ctx context.Context, r *models.Repository) (models.Blobs, error) {
	q := `SELECT b.id, b.media_type, b.digest_hex, b.size, b.created_at, b.marked_at
		FROM blobs as b
		JOIN repository_blobs as rb ON rb.blob_id = b.id
		JOIN repositories AS r ON r.id = rb.repository_id
		WHERE r.id = $1`

	rows, err := s.db.QueryContext(ctx, q, r.ID)
	if err != nil {
		return nil, fmt.Errorf("error finding blobs: %w", err)
	}

	return scanFullBlobs(rows)
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

// FindTagByName finds a tag by name within a repository.
func (s *repositoryStore) FindTagByName(ctx context.Context, r *models.Repository, name string) (*models.Tag, error) {
	q := `SELECT id, name, repository_id, manifest_id, manifest_list_id, created_at, updated_at
		FROM tags WHERE repository_id = $1 AND name = $2`
	row := s.db.QueryRowContext(ctx, q, r.ID, name)

	return scanFullTag(row)
}

// CreateOrFind attempts to create a repository. If the repository already exists (same path) that record is loaded from
// the database into r. This is similar to a FindByPath followed by a Create, but without being prone to race conditions
// on write operations between the corresponding read (FindByPath) and write (Create) operations. Separate Find* and
// Create method calls should be preferred to this when race conditions are not a concern.
func (s *repositoryStore) CreateOrFind(ctx context.Context, r *models.Repository) error {
	q := `INSERT INTO repositories (name, path, parent_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (path) DO NOTHING
		RETURNING id, created_at`

	row := s.db.QueryRowContext(ctx, q, r.Name, r.Path, r.ParentID)
	if err := row.Scan(&r.ID, &r.CreatedAt); err != nil {
		if err != sql.ErrNoRows {
			return fmt.Errorf("creating repository: %w", err)
		}
		// if the result set has no rows, then the repository already exists
		tmp, err := s.FindByPath(ctx, r.Path)
		if err != nil {
			return err
		}
		*r = *tmp
	}

	return nil
}

func splitRepositoryPath(path string) []string {
	return strings.Split(filepath.Clean(path), "/")
}

// repositoryParentPaths parses a repository path (e.g. `"a/b/c"`) and returns its parents path(s) (e.g.
// `["a", "a/b", "a/b/c"]`) starting from the root repository.
func repositoryParentPaths(path string) []string {
	segments := splitRepositoryPath(path)
	names := segments[:len(segments)-1]

	paths := make([]string, 0, len(names))
	for i := 0; i < len(names); i++ {
		paths = append(paths, strings.Join(names[:i+1], "/"))
	}

	return paths
}

// repositoryName parses a repository path (e.g. `"a/b/c"`) and returns its name (e.g. `"c"`).
func repositoryName(path string) string {
	segments := splitRepositoryPath(path)
	return segments[len(segments)-1]
}

// createOrFindParentByPath creates parent repositories for a given path, if any (e.g. `a` and `b` for path `"a/b/c"`),
// preserving their hierarchical relationship. Returns the immediate parent repository, if any (e.g. `b`). No error is
// raised if a repository already exists.
func (s *repositoryStore) createOrFindParentByPath(ctx context.Context, path string) (*models.Repository, error) {
	parentsPath := repositoryParentPaths(path)
	if len(parentsPath) == 0 {
		return nil, nil
	}

	var currParentID int64
	var r *models.Repository

	for _, parentPath := range parentsPath {
		r = &models.Repository{
			Name: repositoryName(parentPath),
			Path: parentPath,
			ParentID: sql.NullInt64{
				Int64: currParentID,
				Valid: currParentID > 0,
			},
		}
		err := s.CreateOrFind(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("finding parent repository: %w", err)
		}

		// track ID to continue linking the chain of repositories
		currParentID = r.ID
	}

	return r, nil
}

// CreateByPath creates the repositories for a given path (e.g. `"a/b/c"`), preserving their hierarchical relationship.
// Returns the leaf repository (e.g. `c`). No error is raised if a parent repository already exists, only if the leaf
// repository does.
func (s *repositoryStore) CreateByPath(ctx context.Context, path string) (*models.Repository, error) {
	p, err := s.createOrFindParentByPath(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("creating or finding parent repository: %w", err)
	}

	r := &models.Repository{Name: repositoryName(path), Path: path}
	if p != nil {
		r.ParentID = sql.NullInt64{Int64: p.ID, Valid: true}
	}
	if err := s.Create(ctx, r); err != nil {
		return nil, err
	}

	return r, nil
}

// CreateOrFindByPath is the fully idempotent version of CreateByPath, where no error is returned if the leaf repository
// already exists.
func (s *repositoryStore) CreateOrFindByPath(ctx context.Context, path string) (*models.Repository, error) {
	p, err := s.createOrFindParentByPath(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("creating or finding parent repository: %w", err)
	}

	r := &models.Repository{Name: repositoryName(path), Path: path}
	if p != nil {
		r.ParentID = sql.NullInt64{Int64: p.ID, Valid: true}
	}
	if err := s.CreateOrFind(ctx, r); err != nil {
		return nil, err
	}

	return r, nil
}

// Update updates an existing repository.
func (s *repositoryStore) Update(ctx context.Context, r *models.Repository) error {
	q := `UPDATE repositories SET (name, path, parent_id, updated_at) = ($1, $2, $3, now())
		WHERE id = $4 RETURNING updated_at`

	row := s.db.QueryRowContext(ctx, q, r.Name, r.Path, r.ParentID, r.ID)
	if err := row.Scan(&r.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("repository not found")
		}
		return fmt.Errorf("error updating repository: %w", err)
	}

	return nil
}

// AssociateManifest associates a manifest and a repository. It does nothing if already associated.
func (s *repositoryStore) AssociateManifest(ctx context.Context, r *models.Repository, m *models.Manifest) error {
	q := `INSERT INTO repository_manifests (repository_id, manifest_id) VALUES ($1, $2)
		ON CONFLICT (repository_id, manifest_id) DO NOTHING`

	if _, err := s.db.ExecContext(ctx, q, r.ID, m.ID); err != nil {
		return fmt.Errorf("error associating manifest: %w", err)
	}

	return nil
}

// AssociateManifestList associates a manifest list and a repository. It does nothing if already associated.
func (s *repositoryStore) AssociateManifestList(ctx context.Context, r *models.Repository, ml *models.ManifestList) error {
	q := `INSERT INTO repository_manifest_lists (repository_id, manifest_list_id) VALUES ($1, $2)
		ON CONFLICT (repository_id, manifest_list_id) DO NOTHING`

	if _, err := s.db.ExecContext(ctx, q, r.ID, ml.ID); err != nil {
		return fmt.Errorf("error associating manifest list: %w", err)
	}

	return nil
}

// DissociateManifest dissociates a manifest and a repository. It does nothing if not associated.
func (s *repositoryStore) DissociateManifest(ctx context.Context, r *models.Repository, m *models.Manifest) error {
	q := "DELETE FROM repository_manifests WHERE repository_id = $1 AND manifest_id = $2"

	res, err := s.db.ExecContext(ctx, q, r.ID, m.ID)
	if err != nil {
		return fmt.Errorf("error dissociating manifest: %w", err)
	}

	if _, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("error dissociating manifest: %w", err)
	}

	return nil
}

// DissociateManifestList dissociates a manifest list and a repository. It does nothing if not associated.
func (s *repositoryStore) DissociateManifestList(ctx context.Context, r *models.Repository, ml *models.ManifestList) error {
	q := "DELETE FROM repository_manifest_lists WHERE repository_id = $1 AND manifest_list_id = $2"

	res, err := s.db.ExecContext(ctx, q, r.ID, ml.ID)
	if err != nil {
		return fmt.Errorf("error dissociating manifest list: %w", err)
	}

	if _, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("error dissociating manifest list: %w", err)
	}

	return nil
}

// UntagManifest deletes all tags of a manifest in a repository.
func (s *repositoryStore) UntagManifest(ctx context.Context, r *models.Repository, m *models.Manifest) error {
	q := "DELETE FROM tags WHERE repository_id = $1 AND manifest_id = $2"

	_, err := s.db.ExecContext(ctx, q, r.ID, m.ID)
	if err != nil {
		return fmt.Errorf("error untagging manifest: %w", err)
	}

	return nil
}

// UntagManifest deletes all tags of a manifest list in a repository.
func (s *repositoryStore) UntagManifestList(ctx context.Context, r *models.Repository, ml *models.ManifestList) error {
	q := "DELETE FROM tags WHERE repository_id = $1 AND manifest_list_id = $2"

	_, err := s.db.ExecContext(ctx, q, r.ID, ml.ID)
	if err != nil {
		return fmt.Errorf("error untagging manifest list: %w", err)
	}

	return nil
}

// LinkBlob links a blob to a repository. It does nothing if already linked.
func (s *repositoryStore) LinkBlob(ctx context.Context, r *models.Repository, b *models.Blob) error {
	q := `INSERT INTO repository_blobs (repository_id, blob_id) VALUES ($1, $2)
		ON CONFLICT (repository_id, blob_id) DO NOTHING`

	if _, err := s.db.ExecContext(ctx, q, r.ID, b.ID); err != nil {
		return fmt.Errorf("error linking blob: %w", err)
	}

	return nil
}

// UnlinkBlob unlinks a blob from a repository. It does nothing if not linked.
func (s *repositoryStore) UnlinkBlob(ctx context.Context, r *models.Repository, b *models.Blob) error {
	q := "DELETE FROM repository_blobs WHERE repository_id = $1 AND blob_id = $2"

	res, err := s.db.ExecContext(ctx, q, r.ID, b.ID)
	if err != nil {
		return fmt.Errorf("error linking blob: %w", err)
	}

	if _, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("error linking blob: %w", err)
	}

	return nil
}

// Delete deletes a repository.
func (s *repositoryStore) Delete(ctx context.Context, id int64) error {
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
