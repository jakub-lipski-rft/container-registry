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

const defaultStorageTimeout = 5 * time.Second

var (
	// for test purposes (mocking)
	blobTaskStoreConstructor = datastore.NewGCBlobTaskStore
	blobStoreConstructor     = datastore.NewBlobStore
)

var _ Worker = (*BlobWorker)(nil)

// BlobWorker is the online GC worker responsible for processing tasks related with blobs. It consumes tasks from the
// blob review queue, identifies if the corresponding blob is eligible for deletion, and if so, deletes it from storage
// and database backends, in this order.
type BlobWorker struct {
	*baseWorker
	vacuum         *storage.Vacuum
	storageTimeout time.Duration
}

// BlobWorkerOption provides functional options for NewBlobWorker.
type BlobWorkerOption func(*BlobWorker)

// WithBlobLogger sets the logger.
func WithBlobLogger(l dcontext.Logger) BlobWorkerOption {
	return func(w *BlobWorker) {
		w.logger = l
	}
}

// WithBlobTxTimeout sets the database transaction timeout for each run. Defaults to 10 seconds.
func WithBlobTxTimeout(d time.Duration) BlobWorkerOption {
	return func(w *BlobWorker) {
		w.txTimeout = d
	}
}

// WithBlobStorageTimeout sets the timeout for storage operations. This is currently used to limit the duration of
// requests to delete dangling blobs on the storage backend. Defaults to 5 seconds.
func WithBlobStorageTimeout(d time.Duration) BlobWorkerOption {
	return func(w *BlobWorker) {
		w.storageTimeout = d
	}
}

func (w *BlobWorker) applyDefaults() {
	w.baseWorker.applyDefaults()
	if w.storageTimeout == 0 {
		w.storageTimeout = defaultStorageTimeout
	}
}

// NewBlobWorker creates a new BlobWorker.
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

// Run implements Worker.
func (w *BlobWorker) Run(ctx context.Context) (bool, error) {
	return w.run(ctx, w)
}

func (w *BlobWorker) processTask(ctx context.Context) (bool, error) {
	log := dcontext.GetLogger(ctx)

	// don't let the database transaction run for longer than w.txTimeout
	ctx, cancel := context.WithDeadline(ctx, timeNow().Add(w.txTimeout))
	defer cancel()

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("creating database transaction: %w", err)
	}
	defer w.rollbackOnExit(ctx, tx)

	bts := blobTaskStoreConstructor(tx)
	t, err := bts.Next(ctx)
	if err != nil {
		return false, err
	}
	if t == nil {
		log.Info("no task available")
		return false, nil
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
		log.Info("the blob is dangling")
		if err := w.deleteBlob(ctx, tx, t); err != nil {
			return true, err
		}
	} else {
		log.Info("the blob is not dangling")
	}

	log.Info("deleting task")
	if err := bts.Delete(ctx, t); err != nil {
		return true, err
	}
	if err := tx.Commit(); err != nil {
		return true, fmt.Errorf("committing database transaction: %w", err)
	}

	return true, nil
}

func (w *BlobWorker) deleteBlob(ctx context.Context, tx datastore.Transactor, t *models.GCBlobTask) error {
	log := dcontext.GetLogger(ctx)

	// delete blob from storage
	ctx2, cancel := context.WithDeadline(ctx, timeNow().Add(w.storageTimeout))
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
			// the transaction duration exceeded w.txTimeout and therefore the connection was closed, just return
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
