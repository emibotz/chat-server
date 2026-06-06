package network

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sync"

	"github.com/emibotz/chat-server/internal/user"
	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/errcode"
	"github.com/emibotz/chat-server/pkg/key"
	"github.com/emibotz/chat-server/pkg/response"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	mu sync.RWMutex

	userService *user.Service

	wsUpgrader *websocket.Upgrader

	handlers []ClientRequestHandler
	clients  []*Client
}

func NewServer(userService *user.Service) *Server {
	return &Server{
		userService: userService,

		wsUpgrader: &websocket.Upgrader{
			// [FIXME] 测试环境用，生产环境修改为更严谨的跨域检测
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},

		clients: nil,
	}
}

func (s *Server) HandleFunc(handler ClientRequestHandler) {
	s.handlers = append(s.handlers, handler)
}

func (s *Server) disconnectClient(client *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	i := slices.Index(s.clients, client)
	s.clients = slices.Delete(s.clients, i, i+1)

	return client.wsConn.Close()
}

func (s *Server) handleClient(ctx context.Context, client *Client) error {
	// 在函数退出时同步断开客户端连接
	defer s.disconnectClient(client)

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

		// [TODO] 验证 API 版本

		// 创建上下文
		ctx, done := context.WithCancel(ctx)

		c := Context{
			Context: ctx,
			Client:  client,
			Request: &request,
		}

		// 分发给客户端请求处理器
		for _, handle := range s.handlers {
			handled, err := handle(&c)

			// 处理错误
			if err != nil {
				event := errcode.NewEvent(errcode.InternalError)

				if err := client.SendEvent(event); err != nil {
					done()
					return err
				}
			}

			// 如果请求已被处理，停止处理
			if handled {
				done()
				break
			}
		}

		done()
	}
}

func (s *Server) Handle(c *echo.Context) error {
	// 从上下文中获取用户 ID
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

	s.mu.Lock()
	s.clients = append(s.clients, &client)
	s.mu.Unlock()

	// 处理客户端请求
	if err := s.handleClient(ctx, &client); err != nil {
		return response.InternalServerError(c, err)
	}

	return nil
}
