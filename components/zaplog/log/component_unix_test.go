//go:build darwin || linux

package log

import (
	"github.com/puper/leo/components/zaplog/log/config"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
)

func countOpenLog(t *testing.T, name string) int {
	t.Helper()
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	target := info.Sys().(*syscall.Stat_t)
	fds, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Skipf("descriptor inventory unavailable: %v", err)
	}
	count := 0
	for _, fd := range fds {
		n, err := strconv.Atoi(fd.Name())
		if err != nil {
			continue
		}
		var st syscall.Stat_t
		if syscall.Fstat(n, &st) == nil && st.Dev == target.Dev && st.Ino == target.Ino {
			count++
		}
	}
	return count
}
func TestCloseReleasesOwnedLogFile(t *testing.T) {
	name := filepath.Join(t.TempDir(), "test.log")
	l, err := New(&config.Config{Logs: map[string]*config.LogConfig{"default": {Output: name}}})
	if err != nil {
		t.Fatal(err)
	}
	l.Get().Info("file descriptor lifecycle")
	if countOpenLog(t, name) == 0 {
		t.Fatal("log file was not opened")
	}
	if err = l.Close(); err != nil {
		t.Fatal(err)
	}
	if n := countOpenLog(t, name); n != 0 {
		t.Errorf("Close left %d log descriptors open", n)
	}
}
