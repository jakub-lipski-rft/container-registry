//go:generate mockgen -package mocks -destination mocks/worker.go . Worker

package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/datastore"
	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/errortracking"
)

const (
	componentKey     = "component"
	defaultTxTimeout = 10 * time.Second
)

// Worker represents an online GC worker, which is responsible for processing review tasks, determining eligibility
// for deletion and deleting artifacts from the backend if eligible.
type Worker interface {
	// Name returns the worker name for observability purposes.
	Name() string
	// Run executes the worker once, processing the next available GC task. A bool is returned to indicate whether
	// there was a task available or not, regardless if processing it succeeded or not.
	Run(context.Context) (bool, error)
}

// for test purposes (mocking)
var timeNow = time.Now

type baseWorker struct {
	name      string
	db        datastore.Handler
	logger    dcontext.Logger
	txTimeout time.Duration
}

// Name implements Worker.
func (w *baseWorker) Name() string {
	return w.name
}

func (w *baseWorker) applyDefaults() {
	if w.logger == nil {
		defaultLogger := logrus.New()
		defaultLogger.SetOutput(ioutil.Discard)
		w.logger = defaultLogger
	}
	if w.txTimeout == 0 {
		w.txTimeout = defaultTxTimeout
	}
}

type processor interface {
	processTask(context.Context) (bool, error)
}

func (w *baseWorker) run(ctx context.Context, p processor) (bool, error) {
	ctx = injectCorrelationID(ctx, w.logger)

	found, err := p.processTask(ctx)
	if err != nil {
		err = fmt.Errorf("processing task: %w", err)
		w.logAndReportErr(ctx, err)
	}

	return found, err
}

func (w *baseWorker) logAndReportErr(ctx context.Context, err error) {
	errortracking.Capture(
		err,
		errortracking.WithContext(ctx),
		errortracking.WithField(componentKey, w.name),
	)
	dcontext.GetLogger(ctx).WithError(err).Error(err.Error())
}

func (w *baseWorker) rollbackOnExit(ctx context.Context, tx datastore.Transactor) {
	rollback := func() {
		// if err is sql.ErrTxDone then the transaction was already committed or rolled back, so it's safe to ignore
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			w.logAndReportErr(ctx, fmt.Errorf("rolling back database transaction: %w", err))
		}
	}
	// in case of panic we want to rollback the transaction straight away, notify Sentry, and then re-panic
	if err := recover(); err != nil {
		rollback()
		sentry.CurrentHub().Recover(err)
		sentry.Flush(5 * time.Second)
		panic(err)
	}
	rollback()
}

func injectCorrelationID(ctx context.Context, logger dcontext.Logger) context.Context {
	id := correlation.SafeRandomID()
	ctx = correlation.ContextWithCorrelation(ctx, id)

	log := logger.WithField("correlation_id", id)
	ctx = dcontext.WithLogger(ctx, log)

	return ctx
}

func exponentialBackoff(i int) time.Duration {
	base := 5 * time.Minute
	max := 24 * time.Hour

	// this should never happen, but just in case...
	if i < 0 {
		return base
	}
	// avoid int64 overflow
	if i > 30 {
		return max
	}

	backoff := base * time.Duration(1<<uint(i))
	if backoff > max {
		backoff = max
	}

	return backoff
}
