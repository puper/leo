package web

import (
	"context"
	"errors"
	"github.com/kataras/iris/v12"
	"github.com/puper/leo/v2/components/iris/web/config"
	"net/http"
	"testing"
	"time"
)

func TestCloseTimeoutCancelsActiveRequest(t *testing.T) {
	addr := testAddress(t)
	entered, finished := make(chan struct{}), make(chan struct{})
	value, err := Builder(&config.Config{Addr: addr, ShutdownTimeout: 20 * time.Millisecond}, func(w *Web) error {
		w.app.Get("/blocked", func(ctx iris.Context) {
			close(entered)
			<-ctx.Request().Context().Done()
			close(finished)
		})
		return nil
	})()
	if err != nil {
		t.Fatal(err)
	}
	w := value.(*Web)
	defer w.app.Hosts[0].Server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	defer client.CloseIdleConnections()
	done := make(chan struct{})
	go func() {
		defer close(done)
		r, _ := client.Get("http://" + addr + "/blocked")
		if r != nil {
			r.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("request not entered")
	}
	if err := w.Close(); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close error = %v", err)
	}
	select {
	case <-finished:
	case <-time.After(100 * time.Millisecond):
		t.Error("Close timed out but left active request connection and context alive")
	}
	w.app.Hosts[0].Server.Close()
	<-done
}
