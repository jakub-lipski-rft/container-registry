package datastore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
)

type gcManifestTaskStore struct {
	db Queryer
}

// NewGCManifestTaskStore builds a new gcManifestTaskStore.
func NewGCManifestTaskStore(db Queryer) *gcManifestTaskStore {
	return &gcManifestTaskStore{db: db}
}

func scanFullGCManifestTasks(rows *sql.Rows) ([]*models.GCManifestTask, error) {
	rr := make([]*models.GCManifestTask, 0)
	defer rows.Close()

	for rows.Next() {
		r := new(models.GCManifestTask)
		err := rows.Scan(&r.RepositoryID, &r.ManifestID, &r.ReviewAfter, &r.ReviewCount)
		if err != nil {
			return nil, fmt.Errorf("scanning GC manifest task: %w", err)
		}
		rr = append(rr, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning GC manifest tasks: %w", err)
	}

	return rr, nil
}

// FindAll finds all GC manifest tasks.
func (s *gcManifestTaskStore) FindAll(ctx context.Context) ([]*models.GCManifestTask, error) {
	q := `SELECT
			repository_id,
			manifest_id,
			review_after,
			review_count
		FROM
			gc_manifest_review_queue`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("finding GC manifest tasks: %w", err)
	}

	return scanFullGCManifestTasks(rows)
}

// Count counts all GC manifest tasks.
func (s *gcManifestTaskStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM gc_manifest_review_queue"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("counting GC manifest tasks: %w", err)
	}

	return count, nil
}
