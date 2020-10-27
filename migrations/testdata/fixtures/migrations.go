// +build integration

package migrationfixtures

import (
	"github.com/docker/distribution/migrations"
)

var allMigrations []*migrations.Migration

func All() []*migrations.Migration {
	return allMigrations
}
