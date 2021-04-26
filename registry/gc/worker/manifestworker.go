package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/gc/internal/metrics"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

var (
	// for test purposes (mocking)
	manifestTaskStoreConstructor = datastore.NewGCManifestTaskStore
	manifestStoreConstructor     = datastore.NewManifestStore
)

var _ Worker = (*ManifestWorker)(nil)

// ManifestWorker is the online GC worker responsible for processing tasks related with manifests. It consumes tasks
// from the manifest review queue, identifies if the corresponding manifest is eligible for deletion, and if so,
// deletes it from the database.
type ManifestWorker struct {
	*baseWorker
}

// ManifestWorkerOption provides functional options for NewManifestWorker.
type ManifestWorkerOption func(*ManifestWorker)

// WithManifestLogger sets the logger.
func WithManifestLogger(l dcontext.Logger) ManifestWorkerOption {
	return func(w *ManifestWorker) {
		w.logger = l
	}
}

// WithManifestTxTimeout sets the database transaction timeout for each run. Defaults to 10 seconds.
func WithManifestTxTimeout(d time.Duration) ManifestWorkerOption {
	return func(w *ManifestWorker) {
		w.txTimeout = d
	}
}

// NewManifestWorker creates a new BlobWorker.
func NewManifestWorker(db datastore.Handler, opts ...ManifestWorkerOption) *ManifestWorker {
	w := &ManifestWorker{baseWorker: &baseWorker{db: db}}
	w.name = "registry.gc.worker.ManifestWorker"
	w.queueName = "gc_manifest_review_queue"
	w.applyDefaults()
	for _, opt := range opts {
		opt(w)
	}
	w.logger = w.logger.WithField(componentKey, w.name)

	return w
}

// Run implements Worker.
func (w *ManifestWorker) Run(ctx context.Context) (bool, error) {
	ctx = dcontext.WithLogger(ctx, w.logger)
	return w.run(ctx, w)
}

// QueueSize implements Worker.
func (w *ManifestWorker) QueueSize(ctx context.Context) (int, error) {
	return manifestTaskStoreConstructor(w.db).Count(ctx)
}

func (w *ManifestWorker) processTask(ctx context.Context) (bool, error) {
	log := dcontext.GetLogger(ctx)

	// don't let the database transaction run for longer than w.txTimeout
	ctx, cancel := context.WithDeadline(ctx, systemClock.Now().Add(w.txTimeout))
	defer cancel()

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("creating database transaction: %w", err)
	}
	defer w.rollbackOnExit(ctx, tx)

	mts := manifestTaskStoreConstructor(tx)
	t, err := mts.Next(ctx)
	if err != nil {
		return false, err
	}
	if t == nil {
		log.Info("no task available")
		return false, nil
	}

	log.WithFields(logrus.Fields{
		"review_after":  t.ReviewAfter.UTC(),
		"review_count":  t.ReviewCount,
		"repository_id": t.RepositoryID,
		"manifest_id":   t.ManifestID,
	}).Info("processing task")

	dangling, err := mts.IsDangling(ctx, t)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			// The transaction duration exceeded w.txTimeout and therefore the connection was closed, just return
			// because the task was unlocked on close and therefore we can't postpone the next review
		default:
			// we don't know how to react here, so just try to postpone the task review and return
			if innerErr := w.postponeTaskAndCommit(ctx, tx, t); innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}
		return true, err
	}

	if dangling {
		log.Info("the manifest is dangling")
		if err := w.deleteManifest(ctx, tx, t); err != nil {
			return true, err
		}
	} else {
		log.Info("the manifest is not dangling")
		// deleting the manifest cascades to the review queue, so we only delete the task directly if not dangling
		log.Info("deleting task")
		if err := mts.Delete(ctx, t); err != nil {
			return true, err
		}
	}

	if err := tx.Commit(); err != nil {
		return true, fmt.Errorf("committing database transaction: %w", err)
	}

	return true, nil
}

func (w *ManifestWorker) deleteManifest(ctx context.Context, tx datastore.Transactor, t *models.GCManifestTask) error {
	log := dcontext.GetLogger(ctx)

	var err error
	var found bool
	ms := manifestStoreConstructor(tx)

	report := metrics.ManifestDelete()
	found, err = ms.Delete(ctx, &models.Manifest{RepositoryID: t.RepositoryID, ID: t.ManifestID})
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			// The transaction duration exceeded w.txTimeout and therefore the connection was closed, just return
			// because the task was unlocked on close and therefore we can't postpone the next review
		default:
			if innerErr := w.postponeTaskAndCommit(ctx, tx, t); innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}
		report(err)
		return err
	}
	if !found {
		// this should never happen because deleting a manifest cascades to the review queue, nevertheless...
		log.Warn("manifest no longer exists on database")
	}

	report(nil)
	return nil
}

func (w *ManifestWorker) postponeTaskAndCommit(ctx context.Context, tx datastore.Transactor, t *models.GCManifestTask) error {
	d := exponentialBackoff(t.ReviewCount)
	dcontext.GetLogger(ctx).WithField("backoff_duration", d.String()).Info("postponing next review")

	if err := manifestTaskStoreConstructor(tx).Postpone(ctx, t, d); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing database transaction: %w", err)
	}

	metrics.ReviewPostpone(w.name)
	return nil
}
