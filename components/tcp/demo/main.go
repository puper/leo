package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puper/leo/pkg/reconnect"
)

type SimpleTCPClient struct {
	addr string
	conn net.Conn
	mu   sync.RWMutex
}

func NewSimpleTCPClient(addr string) *SimpleTCPClient {
	return &SimpleTCPClient{addr: addr}
}

func (c *SimpleTCPClient) Connect(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	return nil
}

func (c *SimpleTCPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *SimpleTCPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil {
		return false
	}
	return c.conn.SetReadDeadline(time.Now().Add(time.Millisecond)) == nil
}

func (c *SimpleTCPClient) GetConn() net.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

func (c *SimpleTCPClient) Send(data []byte) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}

	_, err := conn.Write(data)
	return err
}

func (c *SimpleTCPClient) Receive() ([]byte, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lenBuf)
	if length > 1024*1024 {
		return nil, fmt.Errorf("message too large: %d", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	return data, nil
}

type TCPConnector struct {
	addr   string
	client *SimpleTCPClient
	config *reconnect.DefaultReconnectConfig
}

func NewTCPConnector(addr string) *TCPConnector {
	return &TCPConnector{
		addr:   addr,
		client: NewSimpleTCPClient(addr),
		config: &reconnect.DefaultReconnectConfig{
			MaxRetries:      -1,
			InitialInterval: 200 * time.Millisecond,
			MaxInterval:     1 * time.Second,
		},
	}
}

func (c *TCPConnector) Connect(ctx context.Context) error {
	return c.client.Connect(c.addr)
}

func (c *TCPConnector) Disconnect() error {
	return c.client.Disconnect()
}

func (c *TCPConnector) IsConnected() bool {
	return c.client.IsConnected()
}

func (c *TCPConnector) GetClient() interface{} {
	return c.client
}

func (c *TCPConnector) Config() reconnect.ReconnectConfig {
	return c.config
}

type MyHandler struct {
	connectedCount int32
}

func (h *MyHandler) OnConnected() {
	c := atomic.AddInt32(&h.connectedCount, 1)
	fmt.Printf("[Handler] Connected (count=%d)\n", c)
}

func (h *MyHandler) OnDisconnected(err error) {
	fmt.Printf("[Handler] Disconnected: %v\n", err)
}

func (h *MyHandler) OnReconnecting(attempt int, delay time.Duration) {
	fmt.Printf("[Handler] Reconnecting: attempt=%d, delay=%v\n", attempt, delay)
}

func (h *MyHandler) OnError(err error) {
	fmt.Printf("[Handler] Error: %v\n", err)
}

func main() {
	fmt.Println("=== TCP Reconnect Demo ===")
	fmt.Println()

	listener, err := net.Listen("tcp", "localhost:9998")
	if err != nil {
		fmt.Println("Server listen error:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server started on :9998")

	var dropNext atomic.Bool
	connectionCount := 0
	var connMu sync.Mutex

	// 用于模拟服务端主动断开连接
	var activeConn net.Conn

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			connMu.Lock()
			connectionCount++
			id := connectionCount
			shouldDrop := dropNext.Load()
			connMu.Unlock()

			fmt.Printf("[Server] Accept #%d (drop=%v)\n", id, shouldDrop)

			if shouldDrop {
				conn.Close()
				dropNext.Store(false)
				continue
			}

			// 保存最新活跃连接
			connMu.Lock()
			if activeConn != nil {
				// 关闭旧连接
				activeConn.Close()
			}
			activeConn = conn
			connMu.Unlock()

			go func(c net.Conn, cid int) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					c.SetReadDeadline(time.Now().Add(30 * time.Second))
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					c.Write(buf[:n])
				}
			}(conn, id)
		}
	}()

	time.Sleep(time.Second)

	connector := NewTCPConnector("localhost:9998")
	handler := &MyHandler{}
	comp := reconnect.New(connector, handler, nil)

	if err := comp.Start(); err != nil {
		fmt.Println("Start error:", err)
		return
	}
	fmt.Println("Component started")
	fmt.Println()

	// 第一阶段：正常操作
	fmt.Println("=== Phase 1: Normal operation ===")
	for i := 0; i < 3; i++ {
		client := comp.GetClient()
		err := client.Do(func(raw interface{}) error {
			tcpClient := raw.(*SimpleTCPClient)
			data := []byte(fmt.Sprintf("Msg-%d", i))
			if err := tcpClient.Send(data); err != nil {
				return err
			}
			resp, err := tcpClient.Receive()
			if err != nil {
				return err
			}
			fmt.Printf("[Client] Sent %s, Received %s\n", data, resp)
			return nil
		})
		if err != nil {
			fmt.Printf("[Client] Op %d failed: %v\n", i, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 第二阶段：服务端主动断开连接
	fmt.Println("\n=== Phase 2: Server actively closes connection ===")
	connMu.Lock()
	if activeConn != nil {
		fmt.Println("[Server] Closing active connection...")
		activeConn.Close()
	}
	connMu.Unlock()
	time.Sleep(100 * time.Millisecond)

	// 这次操作会失败并触发重连
	client := comp.GetClient()
	err = client.Do(func(raw interface{}) error {
		tcpClient := raw.(*SimpleTCPClient)
		data := []byte("Msg-after-drop")
		fmt.Printf("[Client] Attempting to send %s\n", data)
		if err := tcpClient.Send(data); err != nil {
			return fmt.Errorf("send failed: %w", err)
		}
		resp, err := tcpClient.Receive()
		if err != nil {
			return fmt.Errorf("receive failed: %w", err)
		}
		fmt.Printf("[Client] Sent %s, Received %s\n", data, resp)
		return nil
	})
	if err != nil {
		fmt.Printf("[Client] Operation after drop failed: %v\n", err)
	}

	// 第三阶段：继续操作
	fmt.Println("\n=== Phase 3: Continue after reconnect ===")
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 3; i++ {
		client := comp.GetClient()
		err := client.Do(func(raw interface{}) error {
			tcpClient := raw.(*SimpleTCPClient)
			data := []byte(fmt.Sprintf("Msg-%d", i))
			if err := tcpClient.Send(data); err != nil {
				return err
			}
			resp, err := tcpClient.Receive()
			if err != nil {
				return err
			}
			fmt.Printf("[Client] Sent %s, Received %s\n", data, resp)
			return nil
		})
		if err != nil {
			fmt.Printf("[Client] Op %d failed: %v\n", i, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	comp.Close()
	fmt.Println("\n=== Demo completed ===")
}
