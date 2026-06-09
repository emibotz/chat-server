package network

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sync"

	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/key"
	"github.com/emibotz/chat-server/pkg/response"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"google.golang.org/protobuf/proto"
)

var APIVersion = "dev_0.0.1"

// WebSocket 服务器
type Server struct {
	mu sync.RWMutex

	wsUpgrader *websocket.Upgrader

	handlers []ClientRequestHandler

	clients         []*Client
	clientsByUserID map[uuid.UUID]*Client
}

// 初始化服务器
func NewServer() *Server {
	return &Server{
		mu: sync.RWMutex{},

		wsUpgrader: &websocket.Upgrader{
			// [FIXME] 测试环境用，生产环境修改为更严谨的跨域检测
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},

		handlers: nil,

		clients:         nil,
		clientsByUserID: make(map[uuid.UUID]*Client),
	}
}

// 添加处理器
func (s *Server) HandleFunc(handler ClientRequestHandler) {
	s.handlers = append(s.handlers, handler)
}

// 添加客户端的内部实现，把客户端添加到列表的同时建立键值链接
func (s *Server) addClient(client *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients = append(s.clients, client)
	s.clientsByUserID[client.userID] = client

	return nil
}

// 移除客户端的内部实现，从列表中删除的同时删除键值链接，并且关闭客户端连接
func (s *Server) removeClient(client *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	i := slices.Index(s.clients, client)
	s.clients = slices.Delete(s.clients, i, i+1)

	delete(s.clientsByUserID, client.userID)

	return client.wsConn.Close()
}

// 处理客户端连接
func (s *Server) handleClient(ctx context.Context, client *Client) error {
	// 在函数退出时同步断开客户端连接
	defer s.removeClient(client)

	for {
		// 读取信息
		messageType, bytes, err := client.wsConn.ReadMessage()
		if err != nil {
			return err
		}

		// 跳过非二进制信息
		if messageType != websocket.BinaryMessage {
			continue
		}

		// 解析客户端请求
		var request pbuf.ClientRequest
		if err := proto.Unmarshal(bytes, &request); err != nil {
			slog.Error(
				"unmarshal client message failed",
				slog.String("error", err.Error()),
			)

			continue
		}

		// 创建请求上下文
		ctx, done := context.WithCancel(ctx)

		c := Context{
			Context: ctx,
			Client:  client,
			Request: &request,
		}

		// 把请求上下文分发给客户端请求处理器
	handling:
		for _, handle := range s.handlers {

			// 处理请求上下文
			handled, err := handle(&c)

			// 如果处理器返回错误，直接将其返回，由上游函数处理
			if err != nil {
				done()
				return err
			}

			// 如果请求已被处理，中断处理器调用，开始解析下一个请求
			if handled {
				break handling
			}
		}

		// 请求处理完成，关闭上下文
		done()
	}
}

// 处理连接
func (s *Server) Handle(c *echo.Context) error {
	// 从上下文中获取用户 ID ，应该被认证中间件注入
	id, ok := c.Get(key.ContextUserID).(uuid.UUID)
	if !ok {
		return response.Unauthorized(c)
	}

	// 获取请求上下文
	ctx := c.Request().Context()

	// 建立 Websocket 连接
	conn, err := s.wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return response.InternalServerError(c, err)
	}
	defer conn.Close()

	// 创建客户端
	client := Client{
		userID: id,
		wsConn: conn,
	}

	// 将客户端添加到列表中并建立键值连接
	s.addClient(&client)

	// 处理客户端请求
	if err := s.handleClient(ctx, &client); err != nil {
		return response.InternalServerError(c, err)
	}

	// 如果客户端没有返回错误，返回空值？？？
	// 我看到 Echo 官方示例中这样做，所以也许
	// 这里连接已经关闭了，所以我也可以这样做？
	return nil
}
