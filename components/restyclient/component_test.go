package restyclient

import (
	"github.com/go-resty/resty/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSetTimeoutPreservesClientConfiguration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "configured" || r.Header.Get("X-Hook") != "called" {
			t.Errorf("headers lost: %v", r.Header)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := New(&Config{Timeout: time.Second})
	c.SetBaseURL(srv.URL)
	c.SetHeader("X-Test", "configured")
	c.OnBeforeRequest(func(_ *resty.Client, r *resty.Request) error { r.SetHeader("X-Hook", "called"); return nil })
	updated := c.SetTimeout(2 * time.Second)
	resp, err := updated.R().Get("/probe")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode() != http.StatusNoContent {
		t.Fatal(resp.Status())
	}
	if updated.GetClient().Timeout != 2*time.Second {
		t.Error("timeout unchanged")
	}
}
