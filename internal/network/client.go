package network

import (
	"context"
	"sync"

	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

var (
	ClientKey  = "network.client"
	RequestKey = "network.request"
)

type Client struct {
	mu sync.RWMutex

	userID uuid.UUID
	wsConn *websocket.Conn
}

func (c *Client) GetUserID() uuid.UUID {
	return c.userID
}

func (c *Client) send(bytes []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.wsConn.WriteMessage(websocket.BinaryMessage, bytes)
}

func (c *Client) SendEvent(event *pbuf.ServerEvent) error {
	bytes, err := proto.Marshal(event)
	if err != nil {
		return err
	}

	return c.send(bytes)
}

type Context struct {
	context.Context
	Client  *Client
	Request *pbuf.ClientRequest
}

func (c *Context) Value(key any) any {
	switch key {
	case ClientKey:
		return c.Client
	case RequestKey:
		return c.Request
	default:
	}

	return c.Context.Value(key)
}

type ClientRequestHandler func(c *Context) (handled bool, err error)
