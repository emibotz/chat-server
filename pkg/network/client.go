package network

import (
	"context"
	"sync"

	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/key"
	"github.com/google/uuid"
)

type Client interface {
	GetUserID() uuid.UUID

	SendEvent(event *pbuf.ServerEvent) error
}

type ClientRequestContext struct {
	context.Context

	mu sync.RWMutex

	Server  Server
	Client  Client
	Request *pbuf.ClientRequest

	values map[any]any
}

func NewClientRequestContext(parent context.Context, server Server, client Client, request *pbuf.ClientRequest) *ClientRequestContext {
	return &ClientRequestContext{
		Context: parent,

		mu: sync.RWMutex{},

		Server:  server,
		Client:  client,
		Request: request,

		values: make(map[any]any),
	}
}

func (c *ClientRequestContext) Value(k any) any {

	switch k {
	case key.ContextServer:
		return c.Server
	case key.ContextClient:
		return c.Client
	case key.ContextRequest:
		return c.Request
	}

	c.mu.RLock()
	if v, ok := c.values[k]; ok {
		c.mu.RUnlock()
		return v
	}
	c.mu.RUnlock()

	return c.Context.Value(k)
}

func (c *ClientRequestContext) Set(k any, v any) {
	c.mu.Lock()
	c.values[k] = v
	c.mu.Unlock()
}

type ClientCloseContext struct {
	context.Context

	mu sync.RWMutex

	Server Server
	Client Client

	values map[any]any
}

func NewClientCloseContext(parent context.Context, server Server, client Client) *ClientCloseContext {
	return &ClientCloseContext{
		Context: parent,

		mu: sync.RWMutex{},

		Server: server,
		Client: client,

		values: make(map[any]any),
	}
}

func (c *ClientCloseContext) Value(k any) any {

	switch k {
	case key.ContextServer:
		return c.Server
	case key.ContextClient:
		return c.Client
	}

	c.mu.RLock()
	if v, ok := c.values[k]; ok {
		c.mu.RUnlock()
		return v
	}
	c.mu.RUnlock()

	return c.Context.Value(k)
}

func (c *ClientCloseContext) Set(k any, v any) {
	c.mu.Lock()
	c.values[k] = v
	c.mu.Unlock()
}

type ClientHandler interface {
	HandleRequest(c *ClientRequestContext) (handled bool, err error)
	HandleClose(c *ClientCloseContext)
}
