package tcp

import (
	"bufio"
	"context"
	"io"
	"net"
	"redis/lib/logger"
	"redis/lib/sync/atomic"
	"redis/lib/sync/wait"
	"sync"
	"time"
)

type EhcoClient struct {
	Conn    net.Conn
	Waiting wait.Wait
}

func (e *EhcoClient) Close() error {
	e.Waiting.WaitWithTimeout(10 * time.Second)
	_ = e.Conn.Close()
	return nil
}

type EchoHandler struct {
	activeConn sync.Map
	closing    atomic.Boolean
}

func MakeHandler() *EchoHandler {
	return &EchoHandler{}
}
func (handler *EchoHandler) Handle(ctx context.Context, conn net.Conn) {
	if handler.closing.Get() {
		_ = conn.Close()
	}
	client := &EhcoClient{
		Conn: conn,
	}
	handler.activeConn.Store(client, struct{}{})
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				logger.Info("Connection close")
				handler.activeConn.Delete(client)
			} else {
				logger.Warn(err)
			}
			return
		}
		client.Waiting.Add(1)
		b := []byte(msg)
		_, _ = conn.Write(b)
		client.Waiting.Done()
	}
}

func (handler EchoHandler) Close() error {
	logger.Info("handler shutting doen")
	handler.closing.Set(true)
	handler.activeConn.Range(func(key, value interface{}) bool {
		client := key.(*EhcoClient)
		_ = client.Conn.Close()
		return true
	})
	return nil
}
