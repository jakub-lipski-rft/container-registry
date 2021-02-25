package gc

import (
	"context"
	"errors"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/cenkalti/backoff/v4"
	"github.com/docker/distribution/registry/gc/internal"
	"github.com/docker/distribution/registry/gc/internal/mocks"
	"github.com/docker/distribution/registry/gc/worker"
	wmocks "github.com/docker/distribution/registry/gc/worker/mocks"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	workerMock := wmocks.NewMockWorker(ctrl)

	tmp := logrus.New()
	tmp.SetOutput(ioutil.Discard)
	defaultLogger := tmp.WithField(componentKey, agentName)

	tmp = logrus.New()
	customLogger := tmp.WithField(componentKey, agentName)

	type args struct {
		w    worker.Worker
		opts []AgentOption
	}
	tests := []struct {
		name string
		args args
		want *Agent
	}{
		{
			name: "defaults",
			args: args{
				w: workerMock,
			},
			want: &Agent{
				worker:          workerMock,
				logger:          defaultLogger,
				initialInterval: defaultInitialInterval,
				maxBackoff:      defaultMaxBackoff,
				noIdleBackoff:   false,
			},
		},
		{
			name: "with logger",
			args: args{
				w:    workerMock,
				opts: []AgentOption{WithLogger(customLogger)},
			},
			want: &Agent{
				worker:          workerMock,
				logger:          customLogger,
				initialInterval: defaultInitialInterval,
				maxBackoff:      defaultMaxBackoff,
				noIdleBackoff:   false,
			},
		},
		{
			name: "with initial interval",
			args: args{
				w:    workerMock,
				opts: []AgentOption{WithInitialInterval(10 * time.Hour)},
			},
			want: &Agent{
				worker:          workerMock,
				logger:          defaultLogger,
				initialInterval: 10 * time.Hour,
				maxBackoff:      defaultMaxBackoff,
				noIdleBackoff:   false,
			},
		},
		{
			name: "with max back off",
			args: args{
				w:    workerMock,
				opts: []AgentOption{WithMaxBackoff(10 * time.Hour)},
			},
			want: &Agent{
				worker:          workerMock,
				logger:          defaultLogger,
				initialInterval: defaultInitialInterval,
				maxBackoff:      10 * time.Hour,
				noIdleBackoff:   false,
			},
		},
		{
			name: "without idle back off",
			args: args{
				w:    workerMock,
				opts: []AgentOption{WithoutIdleBackoff()},
			},
			want: &Agent{
				worker:          workerMock,
				logger:          defaultLogger,
				initialInterval: defaultInitialInterval,
				maxBackoff:      defaultMaxBackoff,
				noIdleBackoff:   true,
			},
		},
		{
			name: "with all options",
			args: args{
				w: workerMock,
				opts: []AgentOption{
					WithLogger(customLogger),
					WithoutIdleBackoff(),
					WithInitialInterval(1 * time.Hour),
					WithMaxBackoff(2 * time.Hour),
					WithoutIdleBackoff(),
				},
			},
			want: &Agent{
				worker:          workerMock,
				logger:          customLogger,
				initialInterval: 1 * time.Hour,
				maxBackoff:      2 * time.Hour,
				noIdleBackoff:   true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAgent(tt.args.w, tt.args.opts...)

			require.Equal(t, tt.want.worker, got.worker)
			require.Equal(t, tt.want.initialInterval, got.initialInterval)
			require.Equal(t, tt.want.maxBackoff, got.maxBackoff)
			require.Equal(t, tt.want.noIdleBackoff, got.noIdleBackoff)

			// we have to cast loggers and compare only their public fields
			wantLogger, ok := tt.want.logger.(*logrus.Entry)
			require.True(t, ok)
			gotLogger, ok := got.logger.(*logrus.Entry)
			require.True(t, ok)
			require.EqualValues(t, wantLogger.Logger.Level, gotLogger.Logger.Level)
			require.Equal(t, wantLogger.Logger.Formatter, gotLogger.Logger.Formatter)
			require.Equal(t, wantLogger.Logger.Out, gotLogger.Logger.Out)
		})
	}
}

func stubBackoff(tb testing.TB, m *mocks.MockBackoff) {
	tb.Helper()

	bkp := backoffConstructor
	backoffConstructor = func(initInterval, maxInterval time.Duration) internal.Backoff {
		return m
	}
	tb.Cleanup(func() { backoffConstructor = bkp })
}

func stubSystemClock(tb testing.TB, m clock.Clock) {
	tb.Helper()

	bkp := systemClock
	systemClock = m
	tb.Cleanup(func() { systemClock = bkp })
}

func TestAgent_Start_Jitter(t *testing.T) {
	ctrl := gomock.NewController(t)
	workerMock := wmocks.NewMockWorker(ctrl)

	clockMock := mocks.NewMockClock(ctrl)
	stubSystemClock(t, clockMock)

	agent := NewAgent(workerMock,
		WithLogger(logrus.New()), // so that we can see the log output during test runs
	)

	// use fixed time for reproducible rand seeds (used to generate jitter durations)
	now := time.Time{}
	rand.Seed(now.UnixNano())
	expectedJitter := time.Duration(rand.Intn(startJitterMaxSeconds)) * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gomock.InOrder(
		workerMock.EXPECT().Name().Times(1),
		clockMock.EXPECT().Now().Return(now).Times(2), // backoff.NewExponentialBackOff calls Now() once
		clockMock.EXPECT().Sleep(expectedJitter).Do(func(_ time.Duration) {
			// cancel context here to avoid a subsequent worker run, which is not needed for the purpose of this test
			cancel()
		}).Times(1),
	)

	err := agent.Start(ctx)
	require.NotNil(t, err)
	require.EqualError(t, context.Canceled, err.Error())
}

func TestAgent_Start_NoTaskFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	workerMock := wmocks.NewMockWorker(ctrl)

	backoffMock := mocks.NewMockBackoff(ctrl)
	stubBackoff(t, backoffMock)

	clockMock := mocks.NewMockClock(ctrl)
	stubSystemClock(t, clockMock)

	agent := NewAgent(workerMock, WithLogger(logrus.New()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seedTime := time.Time{}
	startTime := seedTime.Add(1 * time.Millisecond)
	backOff := defaultInitialInterval

	gomock.InOrder(
		workerMock.EXPECT().Name().Times(1),
		clockMock.EXPECT().Now().Return(seedTime).Times(1),
		clockMock.EXPECT().Sleep(gomock.Any()).Times(1),
		clockMock.EXPECT().Now().Return(startTime).Times(1),
		workerMock.EXPECT().Run(ctx).Return(false, nil).Times(1),
		clockMock.EXPECT().Since(startTime).Return(100*time.Millisecond).Times(1),
		backoffMock.EXPECT().NextBackOff().Return(backOff).Times(1),
		clockMock.EXPECT().Sleep(backOff).Do(func(_ time.Duration) { cancel() }).Times(1),
	)

	err := agent.Start(ctx)
	require.NotNil(t, err)
	require.EqualError(t, context.Canceled, err.Error())
}

func TestAgent_Start_NoTaskFoundWithoutIdleBackoff(t *testing.T) {
	ctrl := gomock.NewController(t)
	workerMock := wmocks.NewMockWorker(ctrl)

	backoffMock := mocks.NewMockBackoff(ctrl)
	stubBackoff(t, backoffMock)

	clockMock := mocks.NewMockClock(ctrl)
	stubSystemClock(t, clockMock)

	agent := NewAgent(workerMock, WithLogger(logrus.New()), WithoutIdleBackoff())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seedTime := time.Time{}
	startTime := seedTime.Add(1 * time.Millisecond)
	backOff := defaultInitialInterval

	gomock.InOrder(
		workerMock.EXPECT().Name().Times(1),
		clockMock.EXPECT().Now().Return(seedTime).Times(1),
		clockMock.EXPECT().Sleep(gomock.Any()).Times(1),
		clockMock.EXPECT().Now().Return(startTime).Times(1),
		workerMock.EXPECT().Run(ctx).Return(false, nil).Times(1),
		backoffMock.EXPECT().Reset().Times(1), // ensure backoff reset
		clockMock.EXPECT().Since(startTime).Return(100*time.Millisecond).Times(1),
		backoffMock.EXPECT().NextBackOff().Return(backOff).Times(1),
		clockMock.EXPECT().Sleep(backOff).Do(func(_ time.Duration) { cancel() }).Times(1),
	)

	err := agent.Start(ctx)
	require.NotNil(t, err)
	require.EqualError(t, context.Canceled, err.Error())
}

func TestAgent_Start_RunFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	workerMock := wmocks.NewMockWorker(ctrl)

	backoffMock := mocks.NewMockBackoff(ctrl)
	stubBackoff(t, backoffMock)

	clockMock := mocks.NewMockClock(ctrl)
	stubSystemClock(t, clockMock)

	agent := NewAgent(workerMock, WithLogger(logrus.New()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seedTime := time.Time{}
	startTime := seedTime.Add(1 * time.Millisecond)
	backOff := defaultInitialInterval

	gomock.InOrder(
		workerMock.EXPECT().Name().Times(1),
		clockMock.EXPECT().Now().Return(seedTime).Times(1),
		clockMock.EXPECT().Sleep(gomock.Any()).Times(1),
		clockMock.EXPECT().Now().Return(startTime).Times(1),
		workerMock.EXPECT().Run(ctx).Return(true, nil).Times(1),
		backoffMock.EXPECT().Reset().Times(1), // ensure backoff reset
		clockMock.EXPECT().Since(startTime).Return(100*time.Millisecond).Times(1),
		backoffMock.EXPECT().NextBackOff().Return(backOff).Times(1),
		clockMock.EXPECT().Sleep(backOff).Do(func(_ time.Duration) { cancel() }).Times(1),
	)

	err := agent.Start(ctx)
	require.NotNil(t, err)
	require.EqualError(t, context.Canceled, err.Error())
}

func TestAgent_Start_RunError(t *testing.T) {
	ctrl := gomock.NewController(t)
	workerMock := wmocks.NewMockWorker(ctrl)

	backoffMock := mocks.NewMockBackoff(ctrl)
	stubBackoff(t, backoffMock)

	clockMock := mocks.NewMockClock(ctrl)
	stubSystemClock(t, clockMock)

	agent := NewAgent(workerMock, WithLogger(logrus.New()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seedTime := time.Time{}
	startTime := seedTime.Add(1 * time.Millisecond)
	backOff := defaultInitialInterval

	gomock.InOrder(
		// there is no backoff reset here
		workerMock.EXPECT().Name().Times(1),
		clockMock.EXPECT().Now().Return(seedTime).Times(1),
		clockMock.EXPECT().Sleep(gomock.Any()).Times(1),
		clockMock.EXPECT().Now().Return(startTime).Times(1),
		workerMock.EXPECT().Run(ctx).Return(false, errors.New("fake error")).Times(1),
		clockMock.EXPECT().Since(startTime).Return(100*time.Millisecond).Times(1),
		backoffMock.EXPECT().NextBackOff().Return(backOff).Times(1),
		clockMock.EXPECT().Sleep(backOff).Do(func(_ time.Duration) { cancel() }).Times(1),
	)

	err := agent.Start(ctx)
	require.NotNil(t, err)
	require.EqualError(t, context.Canceled, err.Error())
}

func TestAgent_Start_RunLoopSurvivesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	workerMock := wmocks.NewMockWorker(ctrl)

	backoffMock := mocks.NewMockBackoff(ctrl)
	stubBackoff(t, backoffMock)

	clockMock := mocks.NewMockClock(ctrl)
	stubSystemClock(t, clockMock)

	agent := NewAgent(workerMock, WithLogger(logrus.New()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seedTime := time.Time{}
	startTime := seedTime.Add(1 * time.Millisecond)
	backOff := defaultInitialInterval

	gomock.InOrder(
		// 1st loop iteration
		workerMock.EXPECT().Name().Times(1),
		clockMock.EXPECT().Now().Return(seedTime).Times(1),
		clockMock.EXPECT().Sleep(gomock.Any()).Times(1),
		clockMock.EXPECT().Now().Return(startTime).Times(1),
		workerMock.EXPECT().Run(ctx).Return(false, errors.New("fake error")).Times(1),
		clockMock.EXPECT().Since(startTime).Return(100*time.Millisecond).Times(1),
		backoffMock.EXPECT().NextBackOff().Return(backOff).Times(1),
		clockMock.EXPECT().Sleep(backOff).Times(1),
		// 2nd loop iteration
		clockMock.EXPECT().Now().Return(startTime).Times(1),
		workerMock.EXPECT().Run(ctx).Return(true, nil).Times(1),
		backoffMock.EXPECT().Reset().Times(1), // ensure backoff reset
		clockMock.EXPECT().Since(startTime).Return(100*time.Millisecond).Times(1),
		backoffMock.EXPECT().NextBackOff().Return(backOff).Times(1),
		clockMock.EXPECT().Sleep(backOff).Do(func(_ time.Duration) {
			// cancel context here to avoid a 3rd worker run
			cancel()
		}).Times(1),
	)

	err := agent.Start(ctx)
	require.NotNil(t, err)
	require.EqualError(t, context.Canceled, err.Error())
}

func Test_newBackoff(t *testing.T) {
	clockMock := clock.NewMock()
	clockMock.Set(time.Time{})
	stubSystemClock(t, clockMock)

	initInterval := 5 * time.Minute
	maxInterval := 24 * time.Hour

	want := &backoff.ExponentialBackOff{
		InitialInterval:     initInterval,
		RandomizationFactor: backoffJitterFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         maxInterval,
		MaxElapsedTime:      0,
		Stop:                backoff.Stop,
		Clock:               clockMock,
	}
	want.Reset()

	tmp := newBackoff(initInterval, maxInterval)
	got, ok := tmp.(*backoff.ExponentialBackOff)
	require.True(t, ok)
	require.NotNil(t, got)
	require.Equal(t, want, got)
}
