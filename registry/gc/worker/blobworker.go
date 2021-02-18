package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/driver"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

const defaultStorageDeadline = 5 * time.Second

var (
	blobTaskStoreConstructor = datastore.NewGCBlobTaskStore
	blobStoreConstructor     = datastore.NewBlobStore
)

type BlobWorker struct {
	*baseWorker
	vacuum          *storage.Vacuum
	storageDeadline time.Duration
}

type BlobWorkerOption func(*BlobWorker)

func WithBlobLogger(l dcontext.Logger) BlobWorkerOption {
	return func(w *BlobWorker) {
		w.logger = l
	}
}

func WithBlobTxDeadline(d time.Duration) BlobWorkerOption {
	return func(w *BlobWorker) {
		w.dbTxDeadline = d
	}
}

func WithBlobStorageDeadline(d time.Duration) BlobWorkerOption {
	return func(w *BlobWorker) {
		w.storageDeadline = d
	}
}

func (w *BlobWorker) applyDefaults() {
	w.baseWorker.applyDefaults()
	if w.storageDeadline == 0 {
		w.storageDeadline = defaultStorageDeadline
	}
}

func NewBlobWorker(db datastore.Handler, storageDeleter driver.StorageDeleter, opts ...BlobWorkerOption) *BlobWorker {
	w := &BlobWorker{
		baseWorker: &baseWorker{db: db},
		vacuum:     storage.NewVacuum(storageDeleter),
	}
	w.name = "registry.gc.worker.BlobWorker"
	w.applyDefaults()
	for _, opt := range opts {
		opt(w)
	}
	w.logger = w.logger.WithField(componentKey, w.name)

	return w
}

func (w *BlobWorker) Run(ctx context.Context) error {
	return w.run(ctx, w)
}

func (w *BlobWorker) processTask(ctx context.Context) error {
	log := dcontext.GetLogger(ctx)

	// don't let the database transaction run for longer than w.dbTxDeadline
	ctx, cancel := context.WithDeadline(ctx, timeNow().Add(w.dbTxDeadline))
	defer cancel()

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("creating database transaction: %w", err)
	}
	defer w.rollbackOnExit(ctx, tx)

	bts := blobTaskStoreConstructor(tx)
	t, err := bts.Next(ctx)
	if err != nil {
		return err
	}
	if t == nil {
		log.Info("no task available")
		return nil
	}
	log.WithFields(logrus.Fields{
		"review_after": t.ReviewAfter.UTC(),
		"review_count": t.ReviewCount,
		"digest":       t.Digest,
	}).Info("processing task")

	dangling, err := bts.IsDangling(ctx, t)
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
		log.Info("the blob is dangling")
		if err := w.deleteBlob(ctx, tx, t); err != nil {
			return err
		}
	} else {
		log.Info("the blob is not dangling")
	}

	log.Info("deleting task")
	if err := bts.Delete(ctx, t); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing database transaction: %w", err)
	}

	return nil
}

func (w *BlobWorker) deleteBlob(ctx context.Context, tx datastore.Transactor, t *models.GCBlobTask) error {
	log := dcontext.GetLogger(ctx)

	// delete blob from storage
	ctx2, cancel := context.WithDeadline(ctx, timeNow().Add(w.storageDeadline))
	defer cancel()

	if err := w.vacuum.RemoveBlob(ctx2, t.Digest); err != nil {
		switch {
		case errors.As(err, &driver.PathNotFoundError{}):
			// this is unexpected, but it's not a show stopper for GC
			log.Warn("blob no longer exists on storage")
		default:
			err = fmt.Errorf("deleting blob from storage: %w", err)
			// we don't know how to react here, so just try to postpone the task review and return
			if innerErr := w.postponeTaskAndCommit(ctx, tx, t); innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
			return err
		}
	}

	// delete blob from database
	bs := blobStoreConstructor(tx)
	if err := bs.Delete(ctx, t.Digest); err != nil {
		switch {
		case err == datastore.ErrNotFound:
			// this is unexpected, but it's not a show stopper for GC
			log.Warn("blob no longer exists on database")
			return nil
		case errors.Is(err, context.DeadlineExceeded):
			// the transaction duration exceeded w.dbTxDeadline and therefore the connection was closed, just return
		default:
			// we don't know how to react here, so just try to postpone the task review and return
			if innerErr := w.postponeTaskAndCommit(ctx, tx, t); innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}
		return err
	}

	return nil
}

func (w *BlobWorker) postponeTaskAndCommit(ctx context.Context, tx datastore.Transactor, t *models.GCBlobTask) error {
	d := exponentialBackoff(t.ReviewCount)
	dcontext.GetLogger(ctx).WithField("backoff_duration", d.String()).Info("postponing next review")

	if err := blobTaskStoreConstructor(tx).Postpone(ctx, t, d); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing database transaction: %w", err)
	}
	return nil
}
