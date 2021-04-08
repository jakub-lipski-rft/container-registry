//go:generate mockgen -package mocks -destination mocks/gcsettings.go . GCSettingsStore

package datastore

import (
	"context"
	"time"

	"github.com/docker/distribution/registry/datastore/metrics"
)

type GCSettingsStore interface {
	UpdateAllReviewAfterDefaults(ctx context.Context, d time.Duration) (bool, error)
}

type gcSettingsStore struct {
	db Queryer
}

// NewGCSettingsStore builds a new gcSettingsStore.
func NewGCSettingsStore(db Queryer) GCSettingsStore {
	return &gcSettingsStore{db: db}
}

// UpdateAllReviewAfterDefaults updates all review after defaults, regardless of the event type. Returns a bool to
// signal if any rows were updated.
func (s *gcSettingsStore) UpdateAllReviewAfterDefaults(ctx context.Context, d time.Duration) (bool, error) {
	defer metrics.InstrumentQuery("gc_settings_update_all_review_after_defaults")()

	q := `UPDATE gc_review_after_defaults
		SET
			value = make_interval(secs => $1)
		WHERE
			value <> make_interval(secs => $1)`

	res, err := s.db.ExecContext(ctx, q, d.Seconds())
	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return n > 0, nil
}
