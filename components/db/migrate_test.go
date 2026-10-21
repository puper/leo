package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"strings"
	"testing"
)

// 夹具在 database/sql 行迭代层注入故障，保留真实 GORM 查询及 Migrate 控制流。
type migrationConnector struct {
	ids              []driver.Value
	queryErr, rowErr error
	inserts          int
}

func (c *migrationConnector) Connect(context.Context) (driver.Conn, error) {
	return &migrationConn{c}, nil
}
func (c *migrationConnector) Driver() driver.Driver { return migrationDriver{} }

type migrationDriver struct{}

func (migrationDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type migrationConn struct{ c *migrationConnector }

func (*migrationConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (*migrationConn) Close() error              { return nil }
func (*migrationConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }
func (c *migrationConn) ExecContext(_ context.Context, q string, _ []driver.NamedValue) (driver.Result, error) {
	if !strings.HasPrefix(q, "INSERT") {
		return nil, fmt.Errorf("unexpected exec: %s", q)
	}
	c.c.inserts++
	return migrationResult{}, nil
}
func (c *migrationConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case q == "SELECT DATABASE()":
		return &migrationRows{values: []driver.Value{"test"}}, nil
	case strings.Contains(q, "information_schema.tables"):
		return &migrationRows{values: []driver.Value{int64(1)}}, nil
	case strings.HasPrefix(q, "SELECT id "):
		if c.c.queryErr != nil {
			return nil, c.c.queryErr
		}
		return &migrationRows{values: c.c.ids, err: c.c.rowErr}, nil
	case strings.Contains(q, "count(*)"):
		var count int64
		for _, id := range c.c.ids {
			if len(args) > 0 && args[0].Value == id {
				count = 1
			}
		}
		return &migrationRows{values: []driver.Value{count}}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", q)
	}
}

type migrationRows struct {
	values []driver.Value
	err    error
	index  int
}

func (*migrationRows) Columns() []string { return []string{"id"} }
func (*migrationRows) Close() error      { return nil }
func (r *migrationRows) Next(dest []driver.Value) error {
	if r.index < len(r.values) {
		dest[0] = r.values[r.index]
		r.index++
		return nil
	}
	if r.err != nil {
		return r.err
	}
	return io.EOF
}
func TestMigrateValidatesCompleteHistory(t *testing.T) {
	injected := errors.New("history interrupted")
	for _, tc := range []struct {
		name                   string
		ids                    []driver.Value
		queryErr, rowErr, want error
		scanErr                bool
		runs                   int
	}{
		{name: "empty", runs: 2}, {name: "known", ids: []driver.Value{"known"}, runs: 1},
		{name: "unknown", ids: []driver.Value{"foreign"}, want: ErrUnknownPastMigration},
		{name: "query_error", queryErr: injected, want: injected},
		{name: "first_row_error", rowErr: injected, want: injected},
		{name: "middle_row_error", ids: []driver.Value{"known"}, rowErr: injected, want: injected},
		{name: "scan_error", ids: []driver.Value{nil}, scanErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := &migrationConnector{ids: tc.ids, queryErr: tc.queryErr, rowErr: tc.rowErr}
			sqldb := sql.OpenDB(fixture)
			defer sqldb.Close()
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqldb, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
			if err != nil {
				t.Fatal(err)
			}
			runs := 0
			apply := func(*gorm.DB) error { runs++; return nil }
			err = NewMigrate(db, &Options{ValidateUnknownMigrations: true}, []*Migration{{ID: "known", Migrate: apply}, {ID: "pending", Migrate: apply}}).Migrate()
			if tc.scanErr {
				if err == nil {
					t.Fatal("scan failure accepted")
				}
			} else if !errors.Is(err, tc.want) {
				t.Errorf("error=%v want=%v", err, tc.want)
			}
			if runs != tc.runs || fixture.inserts != tc.runs {
				t.Errorf("callbacks=%d inserts=%d want=%d", runs, fixture.inserts, tc.runs)
			}
		})
	}
}

type migrationResult struct{}

func (migrationResult) LastInsertId() (int64, error) { return 0, nil }
func (migrationResult) RowsAffected() (int64, error) { return 1, nil }
