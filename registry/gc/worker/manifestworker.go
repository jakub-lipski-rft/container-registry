package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

var (
	manifestTaskStoreConstructor = datastore.NewGCManifestTaskStore
	manifestStoreConstructor     = datastore.NewManifestStore
)

type ManifestWorker struct {
	*baseWorker
}

type ManifestWorkerOption func(*ManifestWorker)

func WithManifestLogger(l dcontext.Logger) ManifestWorkerOption {
	return func(w *ManifestWorker) {
		w.logger = l
	}
}

func WithManifestTxDeadline(d time.Duration) ManifestWorkerOption {
	return func(w *ManifestWorker) {
		w.dbTxDeadline = d
	}
}

func NewManifestWorker(db datastore.Handler, opts ...ManifestWorkerOption) *ManifestWorker {
	w := &ManifestWorker{baseWorker: &baseWorker{db: db}}
	w.name = "registry.gc.worker.ManifestWorker"
	w.applyDefaults()
	for _, opt := range opts {
		opt(w)
	}
	w.logger = w.logger.WithField(componentKey, w.name)

	return w
}

func (w *ManifestWorker) Run(ctx context.Context) error {
	ctx = dcontext.WithLogger(ctx, w.logger)
	return w.run(ctx, w)
}

func (w *ManifestWorker) processTask(ctx context.Context) error {
	log := dcontext.GetLogger(ctx)

	// don't let the database transaction run for longer than w.dbTxDeadline
	ctx, cancel := context.WithDeadline(ctx, timeNow().Add(w.dbTxDeadline))
	defer cancel()

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("creating database transaction: %w", err)
	}
	defer w.rollbackOnExit(ctx, tx)

	mts := manifestTaskStoreConstructor(tx)
	t, err := mts.Next(ctx)
	if err != nil {
		return err
	}
	if t == nil {
		log.Info("no task available")
		return nil
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
			// The transaction duration exceeded w.dbTxDeadline and therefore the connection was closed, just return
			// because the task was unlocked on close and therefore we can't postpone the next review
		default:
			// we don't know how to react here, so just try to postpone the task review and return
			if innerErr := w.postponeTaskAndCommit(ctx, tx, t); innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}
		return err
	}

	if dangling {
		log.Info("the manifest is dangling")
		if err := w.deleteManifest(ctx, tx, t); err != nil {
			return err
		}
	} else {
		log.Info("the manifest is not dangling")
		// deleting the manifest cascades to the review queue, so we only delete the task directly if not dangling
		log.Info("deleting task")
		if err := mts.Delete(ctx, t); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing database transaction: %w", err)
	}

	return nil
}

func (w *ManifestWorker) deleteManifest(ctx context.Context, tx datastore.Transactor, t *models.GCManifestTask) error {
	log := dcontext.GetLogger(ctx)

	ms := manifestStoreConstructor(tx)
	found, err := ms.Delete(ctx, &models.Manifest{RepositoryID: t.RepositoryID, ID: t.ManifestID})
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			// The transaction duration exceeded w.dbTxDeadline and therefore the connection was closed, just return
			// because the task was unlocked on close and therefore we can't postpone the next review
		default:
			if innerErr := w.postponeTaskAndCommit(ctx, tx, t); innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}
		return err
	}
	if !found {
		// this should never happen because deleting a manifest cascades to the review queue, nevertheless...
		log.Warn("manifest no longer exists on database")
	}

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
	return nil
}
