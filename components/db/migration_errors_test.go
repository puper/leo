package db

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openMigrationFixture(t *testing.T, fixture *migrationConnector) *gorm.DB {
	t.Helper()
	conn := sql.OpenDB(fixture)
	t.Cleanup(func() { conn.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMigrationBeginFailureDoesNotRunCallbacks(t *testing.T) {
	want := errors.New("transaction unavailable")
	for _, operation := range []string{"migrate", "rollback_last", "rollback_to", "rollback_migration"} {
		t.Run(operation, func(t *testing.T) {
			fixture := &migrationConnector{beginErr: want, ids: []driver.Value{"first", "second"}}
			db := openMigrationFixture(t, fixture)
			calls := 0
			callback := func(*gorm.DB) error { calls++; return nil }
			first := &Migration{ID: "first", Migrate: callback, Rollback: callback}
			second := &Migration{ID: "second", Migrate: callback, Rollback: callback}
			m := NewMigrate(db, &Options{UseTransaction: true}, []*Migration{first, second})
			var err error
			switch operation {
			case "migrate":
				err = m.Migrate()
			case "rollback_last":
				err = m.RollbackLast()
			case "rollback_to":
				err = m.RollbackTo("first")
			case "rollback_migration":
				err = m.RollbackMigration(second)
			}
			if !errors.Is(err, want) {
				t.Errorf("error = %v, want %v", err, want)
			}
			if calls != 0 || fixture.inserts != 0 {
				t.Fatalf("failed transaction ran %d callbacks and inserted %d records", calls, fixture.inserts)
			}
		})
	}
}

func TestMigrationWithoutUpCallback(t *testing.T) {
	for _, applied := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending", true: "applied"}[applied], func(t *testing.T) {
			fixture := &migrationConnector{}
			if applied {
				fixture.ids = []driver.Value{"rollback_only"}
			}
			db := openMigrationFixture(t, fixture)
			defer func() {
				if value := recover(); value != nil {
					t.Fatalf("missing up callback panicked: %v", value)
				}
			}()
			err := NewMigrate(db, &Options{}, []*Migration{{ID: "rollback_only"}}).Migrate()
			if applied && err != nil {
				t.Fatalf("applied migration needs no up callback: %v", err)
			}
			if !applied && err == nil {
				t.Fatal("pending migration without an up callback succeeded")
			}
			if !applied && (!strings.Contains(err.Error(), "rollback_only") || !strings.Contains(err.Error(), "no migrate function")) {
				t.Fatalf("missing up callback returned an unrelated error: %v", err)
			}
			if fixture.inserts != 0 {
				t.Fatal("migration history changed without running an up callback")
			}
		})
	}
}
