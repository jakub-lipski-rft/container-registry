package migrations

import (
	migrate "github.com/rubenv/sql-migrate"
)

var allMigrations []*migrate.Migration

type Migration struct {
	ID   string
	Up   []string
	Down []string

	DisableTransactionUp   bool
	DisableTransactionDown bool
}

func All() []*Migration {
	mm := make([]*Migration, 0, len(allMigrations))
	for _, m := range allMigrations {
		mm = append(mm, &Migration{
			ID:                     m.Id,
			Up:                     m.Up,
			Down:                   m.Down,
			DisableTransactionUp:   m.DisableTransactionUp,
			DisableTransactionDown: m.DisableTransactionDown,
		})
	}

	return mm
}
