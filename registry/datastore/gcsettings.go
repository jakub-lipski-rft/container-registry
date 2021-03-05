package datastore

import (
	"context"
	"time"

	"github.com/docker/distribution/registry/datastore/metrics"
)

type GCSettingsStore interface {
	UpdateAllReviewAfterDefaults(ctx context.Context, d time.Duration) error
}

type gcSettingsStore struct {
	db Queryer
}

// NewGCSettingsStore builds a new gcSettingsStore.
func NewGCSettingsStore(db Queryer) GCSettingsStore {
	return &gcSettingsStore{db: db}
}

// UpdateAllReviewAfterDefaults updates all review after defaults, regardless of the event type.
func (s *gcSettingsStore) UpdateAllReviewAfterDefaults(ctx context.Context, d time.Duration) error {
	defer metrics.StatementDuration("gc_settings_update_all_review_after_defaults")()

	q := `UPDATE gc_review_after_defaults
		SET
			value = make_interval(secs => $1)
		WHERE
			value <> make_interval(secs => $1)`
	if _, err := s.db.ExecContext(ctx, q, d.Seconds()); err != nil {
		return err
	}

	return nil
}
