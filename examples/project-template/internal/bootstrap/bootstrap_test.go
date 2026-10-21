package bootstrap

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestBootstrapWithoutDatabase(t *testing.T) {
	// 启动真实组件会等待退出信号，使用子进程隔离容器和全局路由状态。
	if path := os.Getenv("TEMPLATE_BOOTSTRAP_TEST_CONFIG"); path != "" {
		if err := Bootstrap(path); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	v := viper.New()
	v.SetConfigFile("../../config/config.example.toml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if v.IsSet("db") {
		t.Fatal("default configuration must not enable database")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	v.Set("app.auth.jwtSecret", base64.URLEncoding.EncodeToString(secret))
	v.Set("app.env", "prod")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	v.Set("web.addr", address)
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := v.WriteConfigAs(path); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=^TestBootstrapWithoutDatabase$")
	command.Env = append(os.Environ(), "TEMPLATE_BOOTSTRAP_TEST_CONFIG="+path)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	})
	client := &http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	for {
		response, err := client.Get("http://" + address + "/readiness")
		if err == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("readiness status: %d", response.StatusCode)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("service did not become ready without a database")
		}
		time.Sleep(20 * time.Millisecond)
	}
	for _, path := range []string{"/api/example/create", "/api/example/detail"} {
		response, err := client.Post("http://"+address+path, "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Errorf("disabled database route %s returned %d", path, response.StatusCode)
		}
	}
}
