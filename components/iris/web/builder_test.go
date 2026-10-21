package web

import (
	"errors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/host"
	"github.com/puper/leo/components/iris/web/config"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func testAddress(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	if err := lis.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}
func assertAddressReleased(t *testing.T, addr string) {
	t.Helper()
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Errorf("listener leaked: %v", err)
		return
	}
	lis.Close()
}
func TestBuildFailureReleasesListener(t *testing.T) {
	addr := testAddress(t)
	want := errors.New("build failed")
	value, err := Builder(&config.Config{Addr: addr, StartCheckTimeout: time.Second}, func(w *Web) error { w.app.OnBuild = func() error { return want }; return nil })()
	if value != nil || !errors.Is(err, want) {
		t.Fatalf("value=%v error=%v", value, err)
	}
	assertAddressReleased(t, addr)
}
func TestHostStartTimeoutIsFailure(t *testing.T) {
	addr := testAddress(t)
	release := make(chan struct{})
	entered := make(chan struct{})
	type result struct {
		value any
		err   error
	}
	done := make(chan result, 1)
	go func() {
		v, e := Builder(&config.Config{Addr: addr, StartCheckTimeout: 20 * time.Millisecond, ShutdownTimeout: time.Second}, func(w *Web) error {
			w.app.ConfigureHost(func(*host.Supervisor) { close(entered); <-release })
			return nil
		})()
		done <- result{v, e}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("host configuration not reached")
	}
	select {
	case r := <-done:
		close(release)
		if r.err == nil {
			t.Error("startup timeout was returned as success")
			_ = r.value.(*Web).Close()
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("startup timeout did not return")
	}
	assertAddressReleased(t, addr)
}
func TestServerUsesConfiguredTimeoutsAndCloses(t *testing.T) {
	addr := testAddress(t)
	cfg := &config.Config{Addr: addr, ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: 3 * time.Second, ShutdownTimeout: time.Second}
	value, err := Builder(cfg, func(w *Web) error {
		w.app.Get("/probe", func(ctx iris.Context) {
			srv := w.app.Hosts[0].Server
			if srv.ReadTimeout != cfg.ReadTimeout || srv.WriteTimeout != cfg.WriteTimeout || srv.IdleTimeout != cfg.IdleTimeout {
				t.Error("server timeout configuration lost")
			}
			_, _ = ctx.WriteString("ok")
		})
		return nil
	})()
	if err != nil {
		t.Fatal(err)
	}
	w := value.(*Web)
	defer w.Close()
	cli := &http.Client{Timeout: time.Second}
	defer cli.CloseIdleConnections()
	resp, err := cli.Get("http://" + addr + "/probe")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || string(data) != "ok" {
		t.Fatalf("response=%q error=%v", data, err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	assertAddressReleased(t, addr)
}

func TestImmediateCloseReleasesListener(t *testing.T) {
	addr := testAddress(t)
	value, err := Builder(&config.Config{Addr: addr, ShutdownTimeout: time.Second})()
	if err != nil {
		t.Fatal(err)
	}
	if err := value.(*Web).Close(); err != nil {
		t.Fatal(err)
	}
	assertAddressReleased(t, addr)
}
