package websocket

import (
	"sync"

	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
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
