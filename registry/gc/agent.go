package gc

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/cenkalti/backoff/v4"
	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/gc/internal"
	"github.com/docker/distribution/registry/gc/internal/metrics"
	"github.com/docker/distribution/registry/gc/worker"
	reginternal "github.com/docker/distribution/registry/internal"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/errortracking"
)

const (
	componentKey = "component"
	agentName    = "registry.gc.Agent"
)

var (
	defaultInitialInterval   = 5 * time.Second
	defaultMaxBackoff        = 24 * time.Hour
	backoffJitterFactor      = 0.33
	startJitterMaxSeconds    = 60
	queueSizeMonitorInterval = 10 * time.Minute
	queueSizeMonitorTimeout  = 100 * time.Millisecond

	// for testing purposes (mocks)
	backoffConstructor                   = newBackoff
	systemClock        reginternal.Clock = clock.New()
)

// Agent manages a online garbage collection worker.
type Agent struct {
	worker          worker.Worker
	logger          dcontext.Logger
	initialInterval time.Duration
	maxBackoff      time.Duration
	noIdleBackoff   bool
}

// AgentOption provides functional options for NewAgent.
type AgentOption func(*Agent)

// WithLogger sets the logger.
func WithLogger(l dcontext.Logger) AgentOption {
	return func(a *Agent) {
		a.logger = l
	}
}

// WithInitialInterval sets the initial interval between worker runs. Defaults to 5 seconds.
func WithInitialInterval(d time.Duration) AgentOption {
	return func(a *Agent) {
		a.initialInterval = d
	}
}

// WithMaxBackoff sets the maximum exponential back off duration used to sleep between worker runs when an error occurs.
// It is also applied when there are no tasks to be processed, unless WithoutIdleBackoff is provided. Please note that
// this is not the absolute maximum, as a randomized jitter factor of up to 33% is always added. Defaults to 24 hours.
func WithMaxBackoff(d time.Duration) AgentOption {
	return func(a *Agent) {
		a.maxBackoff = d
	}
}

// WithoutIdleBackoff disables exponential back offs between worker runs when there are no task to be processed.
func WithoutIdleBackoff() AgentOption {
	return func(a *Agent) {
		a.noIdleBackoff = true
	}
}

func (a *Agent) applyDefaults() {
	if a.logger == nil {
		defaultLogger := logrus.New()
		defaultLogger.SetOutput(ioutil.Discard)
		a.logger = defaultLogger
	}
	if a.initialInterval == 0 {
		a.initialInterval = defaultInitialInterval
	}
	if a.maxBackoff == 0 {
		a.maxBackoff = defaultMaxBackoff
	}
}

// NewAgent creates a new Agent.
func NewAgent(w worker.Worker, opts ...AgentOption) *Agent {
	a := &Agent{worker: w}
	a.applyDefaults()

	for _, opt := range opts {
		opt(a)
	}

	a.logger = a.logger.WithField(componentKey, agentName)

	return a
}

// Start starts the Agent. This is a blocking call that runs the worker in a loop. The loop can be stopped if the
// provided context is canceled. Each worker run is separate by an initial sleep interval (configured through
// WithInitialInterval) with an additional exponential back off up to a given limit (configured through WithMaxBackoff).
// The exponential back off is incremented after every failed run or when no task was found (unless
// WithoutIdleBackoff was provided). The sleep interval is reset to the initial value (removing the exponential back off
// delay) after every successful run, unless no task was found and WithoutIdleBackoff was not provided. The Agent starts
// with a randomized jitter of up to 60 seconds to ease concurrency in clustered environments.
func (a *Agent) Start(ctx context.Context) error {
	log := dcontext.GetLoggerWithField(ctx, "worker", a.worker.Name())
	b := backoffConstructor(a.initialInterval, a.maxBackoff)

	rand.Seed(systemClock.Now().UnixNano())
	/* #nosec G404 */
	jitter := time.Duration(rand.Intn(startJitterMaxSeconds)) * time.Second
	log.WithField("jitter_s", jitter.Seconds()).Info("starting online GC agent")
	systemClock.Sleep(jitter)

	quit := a.startQueueSizeMonitoring(ctx, log)
	defer quit.close()

	for {
		select {
		case <-ctx.Done():
			log.Warn("context cancelled, exiting")
			return ctx.Err()
		default:
			start := systemClock.Now()
			log.Info("running worker")

			report := metrics.WorkerRun(a.worker.Name())
			found, err := a.worker.Run(ctx)
			if err != nil {
				log.WithError(err).Error("failed run")
			} else if found || a.noIdleBackoff {
				b.Reset()
			}
			report(!found, err)
			log.WithField("duration_s", systemClock.Since(start).Seconds()).Info("run complete")

			sleep := b.NextBackOff()
			log.WithField("duration_s", sleep.Seconds()).Info("sleeping")
			metrics.WorkerSleep(a.worker.Name(), sleep)
			systemClock.Sleep(sleep)
		}
	}
}

// For testing purposes, so that we can close the channel there without causing a panic when attempting to do the same
// in the deferred close on Start.
type quitCh struct {
	c chan struct{}
	sync.Once
}

func (qc *quitCh) close() {
	qc.Do(func() { close(qc.c) })
}

func (a *Agent) startQueueSizeMonitoring(ctx context.Context, log dcontext.Logger) *quitCh {
	b := backoffConstructor(queueSizeMonitorInterval, a.maxBackoff)
	quit := &quitCh{c: make(chan struct{})}

	log.WithField("interval_s", queueSizeMonitorInterval.Seconds()).Info("starting online GC queue monitoring")

	go func() {
		time.Sleep(queueSizeMonitorInterval)
		for {
			select {
			case <-ctx.Done():
				log.WithError(ctx.Err()).Warn("context cancelled, stopping worker queue size monitoring")
				return
			case <-quit.c:
				log.Info("stopping worker queue size monitoring")
				return
			default:
				log.Info("measuring worker queue size")
				// apply tight timeout, this is a non-critical lookup
				ctx2, cancel := context.WithDeadline(ctx, systemClock.Now().Add(queueSizeMonitorTimeout))
				count, err := a.worker.QueueSize(ctx2)
				cancel()
				if err != nil {
					errortracking.Capture(
						fmt.Errorf("failed to measure worker queue size: %w", err),
						errortracking.WithContext(ctx),
						errortracking.WithField(componentKey, agentName),
					)
					log.WithError(err).Error("failed to measure worker queue size, backing off")
				} else {
					b.Reset()
					metrics.QueueSize(a.worker.QueueName(), count)
				}
				sleep := b.NextBackOff()
				log.WithField("duration_s", sleep.Seconds()).Debug("sleeping before next queue measurement")
				systemClock.Sleep(sleep)
				time.Sleep(queueSizeMonitorInterval)
			}
		}
	}()

	return quit
}

func newBackoff(initInterval, maxInterval time.Duration) internal.Backoff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initInterval
	b.MaxInterval = maxInterval
	b.RandomizationFactor = backoffJitterFactor
	b.MaxElapsedTime = 0
	b.Clock = systemClock
	b.Reset()

	return b
}
