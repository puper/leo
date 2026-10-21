package nats

import (
	"bufio"
	"bytes"
	"errors"
	natslib "github.com/nats-io/nats.go"
	"github.com/puper/leo/v2/components/nats/config"
	"io"
	"log"
	"net"
	"strings"
	"testing"
	"time"
)

func testNATSServer(t *testing.T) (string, <-chan struct{}) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lis.Close() })
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		_, _ = io.WriteString(conn, "INFO {\"server_id\":\"test\",\"version\":\"2.10.0\",\"proto\":1,\"max_payload\":1048576}\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == "PING" {
				_, _ = io.WriteString(conn, "PONG\r\n")
			}
		}
	}()
	return "nats://" + lis.Addr().String(), done
}
func TestBuilderClosesConnectionWhenConfigurerFails(t *testing.T) {
	addr, done := testNATSServer(t)
	want := errors.New("configuration failed")
	var captured *natslib.Conn
	value, err := Builder(&config.Config{Url: addr}, func(c *natslib.Conn) error { captured = c; return want })()
	if value != nil || !errors.Is(err, want) {
		t.Fatalf("value=%v error=%v", value, err)
	}
	defer captured.Close()
	if !captured.IsClosed() {
		t.Error("failed builder left connection open")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("server connection remained open")
	}
}
func TestErrorHandlerDoesNotLogConnectionCredentials(t *testing.T) {
	addr, _ := testNATSServer(t)
	secret := "test-value-that-must-not-be-logged"
	value, err := Builder(&config.Config{Url: addr, Username: "fixture", Password: secret})()
	if err != nil {
		t.Fatal(err)
	}
	conn := value.(*natslib.Conn)
	defer conn.Close()
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	conn.Opts.AsyncErrorCB(conn, nil, errors.New("synthetic failure"))
	if strings.Contains(output.String(), secret) {
		t.Fatal("connection credentials were included in error log")
	}
	if !strings.Contains(output.String(), "synthetic failure") {
		t.Fatal("error detail was lost")
	}
}
