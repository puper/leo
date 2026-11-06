package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/puper/leo/v2/components/db/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
)

func TestBuilderClosesClientWhenConfigurerFails(t *testing.T) {
	sqldb := sql.OpenDB(&migrationConnector{})
	defer sqldb.Close()
	conn, err := gorm.Open(mysql.New(mysql.Config{Conn: sqldb, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("configuration failed")
	value, err := Builder(&config.Config{}, func(c *Db) error { c.wrappers["test"] = &Wrapper{master: conn}; return want })()
	if value != nil || !errors.Is(err, want) {
		t.Fatalf("value=%v error=%v", value, err)
	}
	if err := sqldb.PingContext(context.Background()); err == nil {
		t.Error("failed builder left pool open")
	}
}
