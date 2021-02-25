package worker

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/docker/distribution/registry/datastore"
	storemock "github.com/docker/distribution/registry/datastore/mocks"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var (
	mtsMock *storemock.MockGCManifestTaskStore
	msMock  *storemock.MockManifestStore
)

func mockManifestStores(tb testing.TB, ctrl *gomock.Controller) {
	tb.Helper()

	mtsMock = storemock.NewMockGCManifestTaskStore(ctrl)
	msMock = storemock.NewMockManifestStore(ctrl)

	mtsBkp := manifestTaskStoreConstructor
	msBkp := manifestStoreConstructor

	manifestTaskStoreConstructor = func(db datastore.Queryer) datastore.GCManifestTaskStore { return mtsMock }
	manifestStoreConstructor = func(db datastore.Queryer) datastore.ManifestStore { return msMock }

	tb.Cleanup(func() {
		manifestTaskStoreConstructor = mtsBkp
		manifestStoreConstructor = msBkp
	})
}

func Test_NewManifestWorker(t *testing.T) {
	ctrl := gomock.NewController(t)

	dbMock := storemock.NewMockHandler(ctrl)
	w := NewManifestWorker(dbMock)

	require.NotNil(t, w.logger)
	require.Equal(t, defaultTxTimeout, w.txTimeout)
}

func Test_NewManifestWorker_WithLogger(t *testing.T) {
	ctrl := gomock.NewController(t)

	logger := logrus.New()
	dbMock := storemock.NewMockHandler(ctrl)
	w := NewManifestWorker(dbMock, WithManifestLogger(logger))

	got, ok := w.logger.(*logrus.Entry)
	require.True(t, ok)
	require.Equal(t, logger.WithField(componentKey, w.name), got)
}

func Test_NewManifestWorker_WithTxDeadline(t *testing.T) {
	ctrl := gomock.NewController(t)

	d := 5 * time.Minute
	dbMock := storemock.NewMockHandler(ctrl)
	w := NewManifestWorker(dbMock, WithManifestTxTimeout(d))

	require.Equal(t, d, w.txTimeout)
}

func fakeManifestTask() *models.GCManifestTask {
	return &models.GCManifestTask{
		RepositoryID: 1,
		ManifestID:   2,
		ReviewAfter:  time.Now().Add(-10 * time.Minute),
		ReviewCount:  0,
	}
}

func TestManifestWorker_processTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	ctx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()
	m := &models.Manifest{RepositoryID: mt.RepositoryID, ID: mt.ManifestID}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(ctx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(ctx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(ctx, mt).Return(true, nil).Times(1),
		msMock.EXPECT().Delete(ctx, m).Return(true, nil).Times(1),
		txMock.EXPECT().Commit().Return(nil).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrTxDone).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.NoError(t, err)
	require.True(t, found)
}

func TestManifestWorker_processTask_BeginTxError(t *testing.T) {
	ctrl := gomock.NewController(t)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	dbMock.EXPECT().BeginTx(dbCtx, nil).Return(nil, fakeErrorA).Times(1)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, fmt.Errorf("creating database transaction: %w", fakeErrorA).Error())
	require.False(t, found)
}

func TestManifestWorker_processTask_NextError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)

	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(nil, fakeErrorA).Times(1),
		txMock.EXPECT().Rollback().Return(nil).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, fakeErrorA.Error())
	require.False(t, found)
}

func TestManifestWorker_processTask_None(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)

	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(nil, nil).Times(1),
		txMock.EXPECT().Rollback().Return(nil).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.NoError(t, err)
	require.False(t, found)
}

func TestManifestWorker_processTask_IsDanglingError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(false, fakeErrorA).Times(1),
		mtsMock.EXPECT().Postpone(dbCtx, mt, isDuration{5 * time.Minute}).Return(nil).Times(1),
		txMock.EXPECT().Commit().Return(nil).Times(1),
		txMock.EXPECT().Rollback().Return(nil).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, fakeErrorA.Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_IsDanglingErrorAndPostponeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(false, fakeErrorA).Times(1),
		mtsMock.EXPECT().Postpone(dbCtx, mt, isDuration{5 * time.Minute}).Return(fakeErrorB).Times(1),
		txMock.EXPECT().Rollback().Return(nil).Times(1),
	)

	found, err := w.processTask(context.Background())
	expectedErr := multierror.Error{
		Errors: []error{
			fakeErrorA,
			fakeErrorB,
		},
	}
	require.EqualError(t, err, expectedErr.Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_IsDanglingDeadlineExceededError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(false, context.DeadlineExceeded).Times(1),
		txMock.EXPECT().Rollback().Return(nil).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, context.DeadlineExceeded.Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_StoreDeleteNotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()
	m := &models.Manifest{RepositoryID: mt.RepositoryID, ID: mt.ManifestID}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(true, nil).Times(1),
		msMock.EXPECT().Delete(dbCtx, m).Return(false, nil).Times(1),
		txMock.EXPECT().Commit().Return(nil).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrTxDone).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.NoError(t, err)
	require.True(t, found)
}

func TestManifestWorker_processTask_StoreDeleteDeadlineExceededError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()
	m := &models.Manifest{RepositoryID: mt.RepositoryID, ID: mt.ManifestID}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(true, nil).Times(1),
		msMock.EXPECT().Delete(dbCtx, m).Return(false, context.DeadlineExceeded).Times(1),
		txMock.EXPECT().Rollback().Return(nil).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, context.DeadlineExceeded.Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_StoreDeleteUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()
	m := &models.Manifest{RepositoryID: mt.RepositoryID, ID: mt.ManifestID}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(true, nil).Times(1),
		msMock.EXPECT().Delete(dbCtx, m).Return(false, fakeErrorA).Times(1),
		mtsMock.EXPECT().Postpone(dbCtx, mt, isDuration{5 * time.Minute}).Return(nil).Times(1),
		txMock.EXPECT().Commit().Return(nil).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrTxDone).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, fakeErrorA.Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_StoreDeleteUnknownErrorAndPostponeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()
	m := &models.Manifest{RepositoryID: mt.RepositoryID, ID: mt.ManifestID}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(true, nil).Times(1),
		msMock.EXPECT().Delete(dbCtx, m).Return(false, fakeErrorA).Times(1),
		mtsMock.EXPECT().Postpone(dbCtx, mt, isDuration{5 * time.Minute}).Return(fakeErrorB).Times(1),
		txMock.EXPECT().Rollback().Return(nil).Times(1),
	)

	found, err := w.processTask(context.Background())
	expectedErr := multierror.Error{
		Errors: []error{
			fakeErrorA,
			fakeErrorB,
		},
	}
	require.EqualError(t, err, expectedErr.Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_StoreDeleteUnknownErrorAndCommitError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()
	m := &models.Manifest{RepositoryID: mt.RepositoryID, ID: mt.ManifestID}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(true, nil).Times(1),
		msMock.EXPECT().Delete(dbCtx, m).Return(false, fakeErrorA).Times(1),
		mtsMock.EXPECT().Postpone(dbCtx, mt, isDuration{5 * time.Minute}).Return(nil).Times(1),
		txMock.EXPECT().Commit().Return(fakeErrorB).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrTxDone).Times(1),
	)

	found, err := w.processTask(context.Background())
	expectedErr := multierror.Error{
		Errors: []error{
			fakeErrorA,
			fmt.Errorf("committing database transaction: %w", fakeErrorB),
		},
	}
	require.EqualError(t, err, expectedErr.Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_IsDanglingNo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(false, nil).Times(1),
		mtsMock.EXPECT().Delete(dbCtx, mt).Return(nil).Times(1),
		txMock.EXPECT().Commit().Return(nil).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrTxDone).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.NoError(t, err)
	require.True(t, found)
}

func TestManifestWorker_processTask_IsDanglingNo_DeleteTaskError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(false, nil).Times(1),
		mtsMock.EXPECT().Delete(dbCtx, mt).Return(fakeErrorA).Times(1),
		txMock.EXPECT().Rollback().Return(nil).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, fakeErrorA.Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_IsDanglingNo_CommitError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}
	mt := fakeManifestTask()

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(mt, nil).Times(1),
		mtsMock.EXPECT().IsDangling(dbCtx, mt).Return(false, nil).Times(1),
		mtsMock.EXPECT().Delete(dbCtx, mt).Return(nil).Times(1),
		txMock.EXPECT().Commit().Return(fakeErrorA).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrConnDone).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, fmt.Errorf("committing database transaction: %w", fakeErrorA).Error())
	require.True(t, found)
}

func TestManifestWorker_processTask_RollbackOnExitUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(nil, fakeErrorA).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrConnDone).Times(1),
	)

	found, err := w.processTask(context.Background())
	require.EqualError(t, err, fakeErrorA.Error())
	require.False(t, found)
}

func TestManifestWorker_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(txMock, nil).Times(1),
		mtsMock.EXPECT().Next(dbCtx).Return(nil, nil).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrTxDone).Times(1),
	)

	found, err := w.Run(context.Background())
	require.NoError(t, err)
	require.False(t, found)
}

func TestManifestWorker_Run_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockManifestStores(t, ctrl)
	stubClock(t, time.Now())

	dbMock := storemock.NewMockHandler(ctrl)
	txMock := storemock.NewMockTransactor(ctrl)
	w := NewManifestWorker(dbMock)

	dbCtx := isContextWithDeadline{timeNow().Add(defaultTxTimeout)}

	gomock.InOrder(
		dbMock.EXPECT().BeginTx(dbCtx, nil).Return(nil, fakeErrorA).Times(1),
		txMock.EXPECT().Rollback().Return(sql.ErrConnDone).Times(1),
	)

	found, err := w.Run(context.Background())
	require.EqualError(t, err, fmt.Errorf("processing task: creating database transaction: %w", fakeErrorA).Error())
	require.False(t, found)
}
