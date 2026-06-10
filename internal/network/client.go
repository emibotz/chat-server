package network

import (
	"context"
	"sync"

	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/key"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
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
	event.Version = &APIVersion

	bytes, err := proto.Marshal(event)
	if err != nil {
		return err
	}

	return c.send(bytes)
}

type Context struct {
	context.Context

	mu sync.RWMutex

	Server  *Server
	Client  *Client
	Request *pbuf.ClientRequest

	values map[any]any
}

func (c *Context) Value(k any) any {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch k {
	case key.ContextServer:
		return c.Server
	case key.ContextClient:
		return c.Client
	case key.ContextRequest:
		return c.Request
	default:
		if v, ok := c.values[k]; ok {
			return v
		}
	}

	return c.Context.Value(k)
}

func (c *Context) Set(k any, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.values == nil {
		c.values = make(map[any]any)
	}
	c.values[k] = v
}

type ClientRequestHandler func(c *Context) (handled bool, err error)
