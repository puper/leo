package db

import (
	"database/sql"
	"encoding/json"
	"github.com/puper/leo/components/db/config"
	"os"
	"testing"
	"time"
)

func TestPartialInitializationClosesOpenedPools(t *testing.T) {
	dsn := os.Getenv("LEO_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("requires isolated MySQL fixture via LEO_TEST_MYSQL_DSN")
	}
	control, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	if err = control.Ping(); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		t.Helper()
		var n int
		if err := control.QueryRow("SELECT COUNT(*) FROM information_schema.processlist WHERE db = DATABASE()").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := count()
	cfg := &config.Config{}
	if err := json.Unmarshal([]byte(`{"servers":{"test":{"slave":["invalid-dsn"],"maxIdleConns":2,"maxOpenConns":2}}}`), cfg); err != nil {
		t.Fatal(err)
	}
	server := cfg.Servers["test"]
	server.Master = dsn
	cfg.Servers["test"] = server
	result, err := New(cfg)
	if result != nil || err == nil {
		t.Fatalf("expected failed partial initialization: result=%v error=%v", result, err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		after := count()
		if after <= before {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("leaked connections: before=%d after=%d", before, after)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
