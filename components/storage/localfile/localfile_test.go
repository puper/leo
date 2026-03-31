package localfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/puper/leo/components/storage"
)

func TestCreateFileMkdirAllErrorReturned(t *testing.T) {
	tmpDir := t.TempDir()

	restrictedDir := filepath.Join(tmpDir, "restricted")
	if err := os.MkdirAll(restrictedDir, 0444); err != nil {
		t.Skipf("Cannot create restricted directory for test: %v", err)
	}

	cfg := &Config{
		RootDir: restrictedDir,
	}
	comp, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = comp.CreateFile("subdir/file.txt", []byte("data"), func(o *storage.Options) *storage.Options {
		o.AutoCreateDir = true
		return o
	})

	if err == nil {
		t.Error("BUG FIXED: MkdirAll should fail but succeeded")
	} else if os.IsPermission(err) {
		t.Logf("FIX VERIFIED: MkdirAll error correctly propagated: %v", err)
	} else {
		t.Logf("CreateFile returned error: %v", err)
	}
}
