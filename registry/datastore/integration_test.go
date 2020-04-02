// +build integration

package datastore_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/testutil"
)

type testSuite struct {
	db       *datastore.DB
	basePath string
	ctx      context.Context
}

var suite *testSuite

func (s *testSuite) setup() error {
	db, err := testutil.NewDB()
	if err != nil {
		return err
	}
	if err := db.MigrateUp(); err != nil {
		return err
	}
	basePath, err := os.Getwd()
	if err != nil {
		return err
	}

	s.db = db
	s.basePath = basePath
	s.ctx = context.Background()

	return nil
}

func (s *testSuite) teardown() error {
	if err := testutil.TruncateAllTables(s.db); err != nil {
		return err
	}
	if err := s.db.Close(); err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	suite = &testSuite{}

	if err := suite.setup(); err != nil {
		panic(fmt.Errorf("setup error: %w", err))
	}
	code := m.Run()
	if err := suite.teardown(); err != nil {
		panic(fmt.Errorf("teardown error: %w", err))
	}

	os.Exit(code)
}
