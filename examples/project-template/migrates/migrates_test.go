package migrates

import (
	"testing"

	"github.com/puper/leo/components/db"
)

func TestMigrationDiscovery(t *testing.T) {
	loaded, err := db.LoadMigrates(Fs)
	if err != nil {
		t.Fatal(err)
	}
	migrations := loaded["server"]
	if len(migrations) != 1 || migrations[0].ID != "001_create_example" || migrations[0].Migrate == nil {
		t.Fatalf("example migration not discoverable: %v", migrations)
	}
}
