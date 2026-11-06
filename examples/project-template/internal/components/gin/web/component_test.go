package web

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"server/internal/components/gin/web/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type testContainer struct{}

func (testContainer) GetLogger(...string) *zap.SugaredLogger { return zap.NewNop().Sugar() }

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	component, err := New(testContainer{}, &config.Config{Addr: "127.0.0.1:0", CorsOrigins: []string{"https://example.com"}}, func(c *Component) error {
		c.GetGinEngine().GET("/test", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := component.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, tc := range []struct {
		origin string
		status int
	}{
		{"https://example.com", http.StatusOK},
		{"https://untrusted.example", http.StatusForbidden},
		{"", http.StatusOK},
	} {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Host = "api.example.com"
		req.Header.Set("Origin", tc.origin)
		res := httptest.NewRecorder()
		component.GetGinEngine().ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Errorf("origin %q: status %d", tc.origin, res.Code)
		}
		if tc.origin == "https://example.com" && res.Header().Get("Access-Control-Allow-Origin") != tc.origin {
			t.Error("missing allowed origin")
		}
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	for _, cfg := range []*config.Config{
		{CorsOrigins: []string{"*"}}, {CorsOrigins: []string{"https://example.com/path"}}, {ReadTimeout: -1},
	} {
		if _, err := New(testContainer{}, cfg); err == nil {
			t.Error("invalid config accepted")
		}
	}
}

func TestNewReturnsListenAndConfigureErrors(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if _, err := New(testContainer{}, &config.Config{Addr: listener.Addr().String()}); err == nil {
		t.Fatal("occupied port accepted")
	}
	sentinel := errors.New("route setup failed")
	if _, err := New(testContainer{}, &config.Config{Addr: "127.0.0.1:0"}, func(*Component) error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("configure error: %v", err)
	}
}

func TestCloseBeforeServeReleasesListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	component := &Component{Container: testContainer{}, srv: &http.Server{}, listener: listener}
	if err := component.Close(); err != nil {
		t.Fatal(err)
	}
	probe, err := net.Listen("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Close returned before releasing listener: %v", err)
	}
	probe.Close()
	if err := component.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
