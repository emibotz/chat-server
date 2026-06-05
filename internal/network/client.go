package network

import (
	"context"

	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	userID uuid.UUID
	wsConn *websocket.Conn
}

func (c *Client) Send(bytes []byte) error {
	return c.wsConn.WriteMessage(websocket.BinaryMessage, bytes)
}

type Context struct {
	context.Context
	Client  *Client
	Request *pbuf.ClientRequest
}

type ClientRequestHandler func(c *Context)
