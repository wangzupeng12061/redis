package handler

import (
	"context"
	"errors"
	"io"
	"net"
	"redis/database"
	databaseface "redis/interface/database"
	"redis/lib/logger"
	"redis/lib/sync/atomic"
	"redis/resp/connection"
	"redis/resp/parser"
	"redis/resp/reply"
	"strings"
	"sync"
)

var unknownErrReplyBytes = []byte("-ERR unknown\r\n")

type RespHandler struct {
	activeConn sync.Map
	db         databaseface.Database
	closing    atomic.Boolean
}

func MakeHandler() *RespHandler {
	var db databaseface.Database
	db = database.NewEchoDatabase()
	return &RespHandler{db: db}

}
func (r *RespHandler) closeClient(client *connection.Connection) {
	_ = client.Close()
	r.db.AfterClientClose(client)
	r.activeConn.Delete(client)
}
func (r *RespHandler) Handle(ctx context.Context, conn net.Conn) {
	if r.closing.Get() {
		_ = conn.Close()
	}
	client := connection.NewConn(conn)
	r.activeConn.Store(client, struct{}{})
	ch := parser.ParseStream(conn)
	for payload := range ch {
		if payload.Err != nil {
			if payload.Err == io.EOF || errors.Is(payload.Err, io.ErrUnexpectedEOF) ||
				strings.Contains(payload.Err.Error(), "use of closed network connection") {
				r.closeClient(client)
				logger.Info("connection closed" + client.RemoteAddr().String())
				return
			}
			errReply := reply.MakeErrReply(payload.Err.Error())
			err := client.Write(errReply.ToBytes())
			if err != nil {
				r.closeClient(client)
				logger.Info("connection closed" + client.RemoteAddr().String())
			}
			continue
		}
		if payload.Data == nil {
			continue
		}
		reply, ok := payload.Data.(*reply.MultiBulkReply)
		if !ok {
			logger.Error("require multi bulk reply")
			continue
		}
		result := r.db.Exec(client, reply.Args)
		if result != nil {
			_ = client.Write(result.ToBytes())
		} else {
			_ = client.Write(unknownErrReplyBytes)

		}
	}
}

func (r *RespHandler) Close() error {
	logger.Info("handle shutting down")
	r.closing.Set(true)
	r.activeConn.Range(func(key interface{}, value interface{}) bool {
		client := key.(*connection.Connection)
		_ = client.Close()
		return true

	})
	r.db.Close()
	return nil
}
