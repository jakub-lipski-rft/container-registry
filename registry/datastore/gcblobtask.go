package datastore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/docker/distribution/registry/datastore/models"
)

type gcBlobTaskStore struct {
	db Queryer
}

// NewGCBlobTaskStore builds a new gcBlobTaskStore.
func NewGCBlobTaskStore(db Queryer) *gcBlobTaskStore {
	return &gcBlobTaskStore{db: db}
}

func scanFullGCBlobTasks(rows *sql.Rows) ([]*models.GCBlobTask, error) {
	rr := make([]*models.GCBlobTask, 0)
	defer rows.Close()

	for rows.Next() {
		var dgst Digest
		r := new(models.GCBlobTask)

		err := rows.Scan(&r.ReviewAfter, &r.ReviewCount, &dgst)
		if err != nil {
			return nil, fmt.Errorf("scanning GC blob task: %w", err)
		}

		d, err := dgst.Parse()
		if err != nil {
			return nil, err
		}
		r.Digest = d

		rr = append(rr, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning GC blob tasks: %w", err)
	}

	return rr, nil
}

// FindAll finds all GC blob tasks.
func (s *gcBlobTaskStore) FindAll(ctx context.Context) ([]*models.GCBlobTask, error) {
	q := `SELECT
			review_after,
			review_count,
			encode(digest, 'hex') as digest
		FROM
			gc_blob_review_queue`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("finding GC blob tasks: %w", err)
	}

	return scanFullGCBlobTasks(rows)
}

// Count counts all GC blob tasks.
func (s *gcBlobTaskStore) Count(ctx context.Context) (int, error) {
	q := "SELECT COUNT(*) FROM gc_blob_review_queue"
	var count int

	if err := s.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return count, fmt.Errorf("counting GC blob tasks: %w", err)
	}

	return count, nil
}
