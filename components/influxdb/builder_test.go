package influxdb

import (
	"errors"
	"github.com/puper/leo/components/influxdb/config"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBuilderReleasesOwnedHTTPConnections(t *testing.T) {
	for _, mode := range []string{"bucket_error", "configurer_error", "success"} {
		t.Run(mode, func(t *testing.T) {
			closed := make(chan struct{}, 4)
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if mode == "bucket_error" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"code":"invalid","message":"bad bucket"}`))
					return
				}
				_, _ = w.Write([]byte(`{"buckets":[{"id":"bucket-id","orgID":"org-id","name":"bucket"}]}`))
			}))
			srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
				if state == http.StateClosed {
					closed <- struct{}{}
				}
			}
			srv.Start()
			defer srv.Close()
			want := errors.New("configuration failed")
			value, err := Builder(&config.Config{ServerUrl: srv.URL, Bucket: "bucket", Org: "org-id"}, func(c *Component) error {
				if mode == "configurer_error" {
					return want
				}
				return nil
			})()
			if mode == "success" {
				if err != nil {
					t.Fatal(err)
				}
				if err = value.(*Component).Close(); err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("failed builder succeeded")
			}
			if mode == "configurer_error" && !errors.Is(err, want) {
				t.Fatal(err)
			}
			select {
			case <-closed:
			case <-time.After(time.Second):
				t.Error("owned HTTP transport kept idle connection open")
			}
		})
	}
}
