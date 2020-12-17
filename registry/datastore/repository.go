package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/docker/distribution"
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
	Tags(ctx context.Context, r *models.Repository) (models.Tags, error)
	TagsPaginated(ctx context.Context, r *models.Repository, limit int, lastName string) (models.Tags, error)
	TagsCountAfterName(ctx context.Context, r *models.Repository, lastName string) (int, error)
	ManifestTags(ctx context.Context, r *models.Repository, m *models.Manifest) (models.Tags, error)
	FindManifestByDigest(ctx context.Context, r *models.Repository, d digest.Digest) (*models.Manifest, error)
	FindManifestByTagName(ctx context.Context, r *models.Repository, tagName string) (*models.Manifest, error)
	FindTagByName(ctx context.Context, r *models.Repository, name string) (*models.Tag, error)
	Blobs(ctx context.Context, r *models.Repository) (models.Blobs, error)
	FindBlob(ctx context.Context, r *models.Repository, d digest.Digest) (*models.Blob, error)
	ExistsBlob(ctx context.Context, r *models.Repository, d digest.Digest) (bool, error)
}

// RepositoryWriter is the interface that defines write operations for a repository store.
type RepositoryWriter interface {
	Create(ctx context.Context, r *models.Repository) error
	CreateByPath(ctx context.Context, path string) (*models.Repository, error)
	CreateOrFind(ctx context.Context, r *models.Repository) error
	CreateOrFindByPath(ctx context.Context, path string) (*models.Repository, error)
	Update(ctx context.Context, r *models.Repository) error
	UntagManifest(ctx context.Context, r *models.Repository, m *models.Manifest) error
	LinkBlob(ctx context.Context, r *models.Repository, d digest.Digest) error
	UnlinkBlob(ctx context.Context, r *models.Repository, d digest.Digest) (bool, error)
	DeleteTagByName(ctx context.Context, r *models.Repository, name string) (bool, error)
	DeleteManifest(ctx context.Context, r *models.Repository, d digest.Digest) (bool, error)
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

// RepositoryManifestService implements the validation.ManifestExister
// interface for repository-scoped manifests.
type RepositoryManifestService struct {
	RepositoryReader
	RepositoryPath string
}

// Exists returns true if the manifest is linked in the repository.
func (rms *RepositoryManifestService) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	r, err := rms.FindByPath(ctx, rms.RepositoryPath)
	if err != nil {
		return false, err
	}

	m, err := rms.FindManifestByDigest(ctx, r, dgst)
	if err != nil {
		return false, err
	}

	return m != nil, nil
}

// RepositoryBlobService implements the distribution.BlobStatter interface for
// repository-scoped blobs.
type RepositoryBlobService struct {
	RepositoryReader
	RepositoryPath string
}

// Stat returns the descriptor of the blob with the provided digest, returns
// distribution.ErrBlobUnknown if not found.
func (rbs *RepositoryBlobService) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	r, err := rbs.FindByPath(ctx, rbs.RepositoryPath)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	b, err := rbs.FindBlob(ctx, r, dgst)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	if b == nil {
		return distribution.Descriptor{}, distribution.ErrBlobUnknown
	}

	return distribution.Descriptor{Digest: b.Digest, Size: b.Size, MediaType: b.MediaType}, nil
}

func scanFullRepository(row *sql.Row) (*models.Repository, error) {
	r := new(models.Repository)

	if err := row.Scan(&r.ID, &r.Name, &r.Path, &r.ParentID, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("scanning repository: %w", err)
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
			return nil, fmt.Errorf("scanning repository: %w", err)
		}
		rr = append(rr, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning repositories: %w", err)
	}

	return rr, nil
}

// FindByID finds a repository by ID.
func (s *repositoryStore) FindByID(ctx context.Context, id int64) (*models.Repository, error) {
	q := `SELECT
			id,
			name,
			path,
			parent_id,
			created_at,
			updated_at
		FROM
			repositories
		WHERE
			id = $1`
	row := s.db.QueryRowContext(ctx, q, id)

	return scanFullRepository(row)
}

// FindByPath finds a repository by path.
func (s *repositoryStore) FindByPath(ctx context.Context, path string) (*models.Repository, error) {
	q := `SELECT
			id,
			name,
			path,
			parent_id,
			created_at,
			updated_at
		FROM
			repositories
		WHERE
			path = $1`
	row := s.db.QueryRowContext(ctx, q, path)

	return scanFullRepository(row)
}

// FindAll finds all repositories.
func (s *repositoryStore) FindAll(ctx context.Context) (models.Repositories, error) {
	q := `SELECT
			id,
			name,
			path,
			parent_id,
			created_at,
			updated_at
		FROM
			repositories`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("finding repositories: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindAllPaginated finds up to limit repositories with path lexicographically after lastPath. This is used exclusively
// for the GET /v2/_catalog API route, where pagination is done with a marker (lastPath). Empty repositories (which do
// not have at least a manifest) are ignored. Also, even if there is no repository with a path of lastPath, the returned
// repositories will always be those with a path lexicographically after lastPath. Finally, repositories are
// lexicographically sorted. These constraints exists to preserve the existing API behavior (when doing a filesystem
// walk based pagination).
func (s *repositoryStore) FindAllPaginated(ctx context.Context, limit int, lastPath string) (models.Repositories, error) {
	q := `SELECT
			r.id,
			r.name,
			r.path,
			r.parent_id,
			r.created_at,
			r.updated_at
		FROM
			repositories AS r
		WHERE
			EXISTS (
				SELECT
				FROM
					manifests AS m
				WHERE
					m.repository_id = r.id)
			AND r.path > $1
		ORDER BY
			r.path
		LIMIT $2`
	rows, err := s.db.QueryContext(ctx, q, lastPath, limit)
	if err != nil {
		return nil, fmt.Errorf("finding repositories with pagination: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindDescendantsOf finds all descendants of a given repository.
func (s *repositoryStore) FindDescendantsOf(ctx context.Context, id int64) (models.Repositories, error) {
	q := `WITH RECURSIVE descendants AS (
			SELECT
				id,
				name,
				path,
				parent_id,
				created_at,
				updated_at
			FROM
				repositories
			WHERE
				id = $1
			UNION ALL
			SELECT
				r.id,
				r.name,
				r.path,
				r.parent_id,
				r.created_at,
				r.updated_at
			FROM
				repositories AS r
				JOIN descendants ON descendants.id = r.parent_id
		)
		SELECT
			*
		FROM
			descendants
		WHERE
			descendants.id != $1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("finding descendants of repository: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindAncestorsOf finds all ancestors of a given repository.
func (s *repositoryStore) FindAncestorsOf(ctx context.Context, id int64) (models.Repositories, error) {
	q := `WITH RECURSIVE ancestors AS (
			SELECT
				id,
				name,
				path,
				parent_id,
				created_at,
				updated_at
			FROM
				repositories
			WHERE
				id = $1
			UNION ALL
			SELECT
				r.id,
				r.name,
				r.path,
				r.parent_id,
				r.created_at,
				r.updated_at
			FROM
				repositories AS r
				JOIN ancestors ON ancestors.parent_id = r.id
		)
		SELECT
			*
		FROM
			ancestors
		WHERE
			ancestors.id != $1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("finding ancestors of repository: %w", err)
	}

	return scanFullRepositories(rows)
}

// FindSiblingsOf finds all siblings of a given repository.
func (s *repositoryStore) FindSiblingsOf(ctx context.Context, id int64) (models.Repositories, error) {
	q := `SELECT
			siblings.id,
			siblings.name,
			siblings.path,
			siblings.parent_id,
			siblings.created_at,
			siblings.updated_at
		FROM
			repositories AS siblings
			LEFT JOIN repositories AS anchor ON siblings.parent_id = anchor.parent_id
		WHERE
			anchor.id = $1
			AND siblings.id != $1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("finding siblings of repository: %w", err)
	}

	return scanFullRepositories(rows)
}

// Tags finds all tags of a given repository.
func (s *repositoryStore) Tags(ctx context.Context, r *models.Repository) (models.Tags, error) {
	q := `SELECT
			id,
			name,
			repository_id,
			manifest_id,
			created_at,
			updated_at
		FROM
			tags
		WHERE
			repository_id = $1`

	rows, err := s.db.QueryContext(ctx, q, r.ID)
	if err != nil {
		return nil, fmt.Errorf("finding tags: %w", err)
	}

	return scanFullTags(rows)
}

// TagsPaginated finds up to limit tags of a given repository with name lexicographically after lastName. This is used
// exclusively for the GET /v2/<name>/tags/list API route, where pagination is done with a marker (lastName). Even if
// there is no tag with a name of lastName, the returned tags will always be those with a path lexicographically after
// lastName. Finally, tags are lexicographically sorted. These constraints exists to preserve the existing API behavior
// (when doing a filesystem walk based pagination).
func (s *repositoryStore) TagsPaginated(ctx context.Context, r *models.Repository, limit int, lastName string) (models.Tags, error) {
	q := `SELECT
			id,
			name,
			repository_id,
			manifest_id,
			created_at,
			updated_at
		FROM
			tags
		WHERE
			repository_id = $1
			AND name > $2
		ORDER BY
			name
		LIMIT $3`
	rows, err := s.db.QueryContext(ctx, q, r.ID, lastName, limit)
	if err != nil {
		return nil, fmt.Errorf("finding tags with pagination: %w", err)
	}

	return scanFullTags(rows)
}

// TagsCountAfterName counts all tags of a given repository with name lexicographically after lastName. This is used
// exclusively for the GET /v2/<name>/tags/list API route, where pagination is done with a marker (lastName). Even if
// there is no tag with a name of lastName, the counted tags will always be those with a path lexicographically after
// lastName. This constraint exists to preserve the existing API behavior (when doing a filesystem walk based
// pagination).
func (s *repositoryStore) TagsCountAfterName(ctx context.Context, r *models.Repository, lastName string) (int, error) {
	q := `SELECT
			COUNT(id)
		FROM
			tags
		WHERE
			repository_id = $1
			AND name > $2`

	var count int
	if err := s.db.QueryRowContext(ctx, q, r.ID, lastName).Scan(&count); err != nil {
		return count, fmt.Errorf("counting tags lexicographically after name: %w", err)
	}

	return count, nil
}

// ManifestTags finds all tags of a given repository manifest.
func (s *repositoryStore) ManifestTags(ctx context.Context, r *models.Repository, m *models.Manifest) (models.Tags, error) {
	q := `SELECT
			id,
			name,
			repository_id,
			manifest_id,
			created_at,
			updated_at
		FROM
			tags
		WHERE
			repository_id = $1
			AND manifest_id = $2`

	rows, err := s.db.QueryContext(ctx, q, r.ID, m.ID)
	if err != nil {
		return nil, fmt.Errorf("finding tags: %w", err)
	}

	return scanFullTags(rows)
}

// Count counts all repositories.
func (s *repositoryStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM repositories"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("counting repositories: %w", err)
	}

	return count, nil
}

// CountAfterPath counts all repositories with path lexicographically after lastPath. This is used exclusively
// for the GET /v2/_catalog API route, where pagination is done with a marker (lastPath). Empty repositories (which do
// not have at least a manifest) are ignored. Also, even if there is no repository with a path of lastPath, the counted
// repositories will always be those with a path lexicographically after lastPath. These constraints exists to preserve
// the existing API behavior (when doing a filesystem walk based pagination).
func (s *repositoryStore) CountAfterPath(ctx context.Context, path string) (int, error) {
	q := `SELECT
			COUNT(*)
		FROM
			repositories AS r
		WHERE
			EXISTS (
				SELECT
				FROM
					manifests AS m
				WHERE
					m.repository_id = r.id)
			AND r.path > $1`

	var count int
	if err := s.db.QueryRowContext(ctx, q, path).Scan(&count); err != nil {
		return count, fmt.Errorf("counting repositories lexicographically after path: %w", err)
	}

	return count, nil
}

// Manifests finds all manifests associated with a repository.
func (s *repositoryStore) Manifests(ctx context.Context, r *models.Repository) (models.Manifests, error) {
	q := `SELECT
			m.id,
			m.repository_id,
			m.schema_version,
			mt.media_type,
			encode(m.digest, 'hex') as digest,
			m.payload,
			mtc.media_type as configuration_media_type,
			encode(m.configuration_blob_digest, 'hex') as configuration_blob_digest,
			m.configuration_payload,
			m.created_at
		FROM
			manifests AS m
			JOIN media_types AS mt ON mt.id = m.media_type_id
			LEFT JOIN media_types AS mtc ON mtc.id = m.configuration_media_type_id
		WHERE
			m.repository_id = $1
		ORDER BY m.id`

	rows, err := s.db.QueryContext(ctx, q, r.ID)
	if err != nil {
		return nil, fmt.Errorf("finding manifests: %w", err)
	}

	return scanFullManifests(rows)
}

// FindManifestByDigest finds a manifest by digest within a repository.
func (s *repositoryStore) FindManifestByDigest(ctx context.Context, r *models.Repository, d digest.Digest) (*models.Manifest, error) {
	q := `SELECT
			m.id,
			m.repository_id,
			m.schema_version,
			mt.media_type,
			encode(m.digest, 'hex') as digest,
			m.payload,
			mtc.media_type as configuration_media_type,
			encode(m.configuration_blob_digest, 'hex') as configuration_blob_digest,
			m.configuration_payload,
			m.created_at
		FROM
			manifests AS m
			JOIN media_types AS mt ON mt.id = m.media_type_id
			LEFT JOIN media_types AS mtc ON mtc.id = m.configuration_media_type_id
		WHERE
			m.repository_id = $1
			AND m.digest = decode($2, 'hex')`

	dgst, err := NewDigest(d)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, q, r.ID, dgst)

	return scanFullManifest(row)
}

// FindManifestByTagName finds a manifest by tag name within a repository.
func (s *repositoryStore) FindManifestByTagName(ctx context.Context, r *models.Repository, tagName string) (*models.Manifest, error) {
	q := `SELECT
			m.id,
			m.repository_id,
			m.schema_version,
			mt.media_type,
			encode(m.digest, 'hex') as digest,
			m.payload,
			mtc.media_type as configuration_media_type,
			encode(m.configuration_blob_digest, 'hex') as configuration_blob_digest,
			m.configuration_payload,
			m.created_at
		FROM
			manifests AS m
			JOIN media_types AS mt ON mt.id = m.media_type_id
			LEFT JOIN media_types AS mtc ON mtc.id = m.configuration_media_type_id
			JOIN tags AS t ON t.repository_id = m.repository_id
				AND t.manifest_id = m.id
		WHERE
			m.repository_id = $1
			AND t.name = $2`

	row := s.db.QueryRowContext(ctx, q, r.ID, tagName)

	return scanFullManifest(row)
}

// Blobs finds all blobs associated with the repository.
func (s *repositoryStore) Blobs(ctx context.Context, r *models.Repository) (models.Blobs, error) {
	q := `SELECT
			mt.media_type,
			encode(b.digest, 'hex') as digest,
			b.size,
			b.created_at
		FROM
			blobs AS b
			JOIN repository_blobs AS rb ON rb.blob_digest = b.digest
			JOIN repositories AS r ON r.id = rb.repository_id
			JOIN media_types AS mt ON mt.id = b.media_type_id
		WHERE
			r.id = $1`

	rows, err := s.db.QueryContext(ctx, q, r.ID)
	if err != nil {
		return nil, fmt.Errorf("finding blobs: %w", err)
	}

	return scanFullBlobs(rows)
}

// FindBlobByDigest finds a blob by digest within a repository.
func (s *repositoryStore) FindBlob(ctx context.Context, r *models.Repository, d digest.Digest) (*models.Blob, error) {
	q := `SELECT
			mt.media_type,
			encode(b.digest, 'hex') as digest,
			b.size,
			b.created_at
		FROM
			blobs AS b
			JOIN media_types AS mt ON mt.id = b.media_type_id
			JOIN repository_blobs AS rb ON rb.blob_digest = b.digest
		WHERE
			rb.repository_id = $1
			AND b.digest = decode($2, 'hex')`

	dgst, err := NewDigest(d)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, q, r.ID, dgst)

	return scanFullBlob(row)
}

// ExistsBlobByDigest finds if a blob with a given digest exists within a repository.
func (s *repositoryStore) ExistsBlob(ctx context.Context, r *models.Repository, d digest.Digest) (bool, error) {
	q := `SELECT
			EXISTS (
				SELECT
					1
				FROM
					repository_blobs
				WHERE
					repository_id = $1
					AND blob_digest = decode($2, 'hex'))`

	dgst, err := NewDigest(d)
	if err != nil {
		return false, err
	}

	var exists bool
	row := s.db.QueryRowContext(ctx, q, r.ID, dgst)
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("scanning blob: %w", err)
	}

	return exists, nil
}

// Create saves a new repository.
func (s *repositoryStore) Create(ctx context.Context, r *models.Repository) error {
	q := `INSERT INTO repositories (name, path, parent_id)
			VALUES ($1, $2, $3)
		RETURNING
			id, created_at`

	row := s.db.QueryRowContext(ctx, q, r.Name, r.Path, r.ParentID)
	if err := row.Scan(&r.ID, &r.CreatedAt); err != nil {
		return fmt.Errorf("creating repository: %w", err)
	}

	return nil
}

// FindTagByName finds a tag by name within a repository.
func (s *repositoryStore) FindTagByName(ctx context.Context, r *models.Repository, name string) (*models.Tag, error) {
	q := `SELECT
			id,
			name,
			repository_id,
			manifest_id,
			created_at,
			updated_at
		FROM
			tags
		WHERE
			repository_id = $1
			AND name = $2`
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
		ON CONFLICT (path)
			DO NOTHING
		RETURNING
			id, created_at`

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
	q := `UPDATE
			repositories
		SET
			(name, path, parent_id, updated_at) = ($1, $2, $3, now())
		WHERE
			id = $4
		RETURNING
			updated_at`

	row := s.db.QueryRowContext(ctx, q, r.Name, r.Path, r.ParentID, r.ID)
	if err := row.Scan(&r.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("repository not found")
		}
		return fmt.Errorf("updating repository: %w", err)
	}

	return nil
}

// UntagManifest deletes all tags of a manifest in a repository.
func (s *repositoryStore) UntagManifest(ctx context.Context, r *models.Repository, m *models.Manifest) error {
	q := "DELETE FROM tags WHERE repository_id = $1 AND manifest_id = $2"

	_, err := s.db.ExecContext(ctx, q, r.ID, m.ID)
	if err != nil {
		return fmt.Errorf("untagging manifest: %w", err)
	}

	return nil
}

// LinkBlob links a blob to a repository. It does nothing if already linked.
func (s *repositoryStore) LinkBlob(ctx context.Context, r *models.Repository, d digest.Digest) error {
	q := `INSERT INTO repository_blobs (repository_id, blob_digest)
			VALUES ($1, decode($2, 'hex'))
		ON CONFLICT (repository_id, blob_digest)
			DO NOTHING`

	dgst, err := NewDigest(d)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, q, r.ID, dgst); err != nil {
		return fmt.Errorf("linking blob: %w", err)
	}

	return nil
}

// UnlinkBlob unlinks a blob from a repository. It does nothing if not linked. A boolean is returned to denote whether
// the link was deleted or not. This avoids the need for a separate preceding `SELECT` to find if it exists.
func (s *repositoryStore) UnlinkBlob(ctx context.Context, r *models.Repository, d digest.Digest) (bool, error) {
	q := "DELETE FROM repository_blobs WHERE repository_id = $1 AND blob_digest = decode($2, 'hex')"

	dgst, err := NewDigest(d)
	if err != nil {
		return false, err
	}
	res, err := s.db.ExecContext(ctx, q, r.ID, dgst)
	if err != nil {
		return false, fmt.Errorf("linking blob: %w", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("linking blob: %w", err)
	}

	return count == 1, nil
}

// DeleteTagByName deletes a tag by name within a repository. A boolean is returned to denote whether the tag was
// deleted or not. This avoids the need for a separate preceding `SELECT` to find if it exists.
func (s *repositoryStore) DeleteTagByName(ctx context.Context, r *models.Repository, name string) (bool, error) {
	q := "DELETE FROM tags WHERE repository_id = $1 AND name = $2"

	res, err := s.db.ExecContext(ctx, q, r.ID, name)
	if err != nil {
		return false, fmt.Errorf("deleting tag: %w", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("deleting tag: %w", err)
	}

	return count == 1, nil
}

// DeleteManifest deletes a manifest from a repository. A boolean is returned to denote whether the manifest was deleted
// or not. This avoids the need for a separate preceding `SELECT` to find if it exists.
func (s *repositoryStore) DeleteManifest(ctx context.Context, r *models.Repository, d digest.Digest) (bool, error) {
	q := "DELETE FROM manifests WHERE repository_id = $1 AND digest = decode($2, 'hex')"

	dgst, err := NewDigest(d)
	if err != nil {
		return false, err
	}

	res, err := s.db.ExecContext(ctx, q, r.ID, dgst)
	if err != nil {
		return false, fmt.Errorf("deleting manifest: %w", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("deleting manifest: %w", err)
	}

	return count == 1, nil
}

// Delete deletes a repository.
func (s *repositoryStore) Delete(ctx context.Context, id int64) error {
	q := "DELETE FROM repositories WHERE id = $1"

	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("deleting repository: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting repository: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("repository not found")
	}

	return nil
}
