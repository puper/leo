package localfile

import (
	"errors"
	"github.com/puper/leo/components/storage"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestFileErrorsAndDirectoryInfo(t *testing.T) {
	c, err := New(&Config{RootDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.GetFileData("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing error=%v", err)
	}
	if err = c.CreateDir("dir"); err != nil {
		t.Fatal(err)
	}
	info, err := c.GetFileInfo("dir")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir {
		t.Error("directory reported as file")
	}
}
func TestConcurrentCreateDoesNotOverwrite(t *testing.T) {
	c, _ := New(&Config{RootDir: t.TempDir()})
	name := strings.Repeat("nested/", 20) + "same"
	var successes atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if c.CreateFile(name, []byte("data"), storage.WithAutoCreateDir) == nil {
				successes.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if successes.Load() != 1 {
		t.Errorf("successful creators=%d", successes.Load())
	}
	if err := c.CreateFile(name, []byte("replacement"), storage.WithForceReplace); err != nil {
		t.Fatal(err)
	}
	b, err := c.GetFileData(name)
	if err != nil || string(b) != "replacement" {
		t.Fatalf("replacement=%q error=%v", b, err)
	}
}
func TestStorageCannotFollowSymlinkOutsideRoot(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(base, "outside")
	root := filepath.Join(base, "root")
	for _, p := range []string{outside, root} {
		if err := os.Mkdir(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(outside, "secret")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	c, _ := New(&Config{RootDir: root})
	checks := []struct {
		name string
		run  func() error
	}{
		{"read", func() error { _, e := c.GetFileData("link/secret"); return e }},
		{"stat", func() error { _, e := c.GetFileInfo("link/secret"); return e }},
		{"list", func() error { _, e := c.ListDir("link"); return e }},
		{"write", func() error { return c.CreateFile("link/secret", []byte("changed"), storage.WithForceReplace) }},
		{"mkdir", func() error { return c.CreateDir("link/new") }},
		{"remove", func() error { return c.Remove("link/secret") }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.run(); err == nil {
				t.Error("root escape accepted")
			}
		})
	}
	b, err := os.ReadFile(target)
	if err != nil || string(b) != "original" {
		t.Errorf("outside file changed: %q %v", b, err)
	}
}
func TestCreateMissingRootAndInternalSymlink(t *testing.T) {
	root := filepath.Join(t.TempDir(), "new")
	c, _ := New(&Config{RootDir: root})
	if err := c.CreateFile("dir/file", []byte("ok"), storage.WithAutoCreateDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("dir", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	b, err := c.GetFileData("link/file")
	if err != nil || string(b) != "ok" {
		t.Fatalf("internal link=%q %v", b, err)
	}
	if _, err = c.GetFileData("../escape"); err == nil {
		t.Error("parent traversal accepted")
	}
}

func TestRemoveConfiguredRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "storage")
	c, err := New(&Config{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.CreateFile("dir/file", []byte("data"), storage.WithAutoCreateDir); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove("", storage.WithRemoveAll); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("root still present: %v", err)
	}
}
