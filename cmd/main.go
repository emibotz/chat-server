package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emibotz/chat-server/internal/game"
	tickMiddleware "github.com/emibotz/chat-server/internal/game/middleware"
	"github.com/emibotz/chat-server/internal/middleware"
	"github.com/emibotz/chat-server/internal/network"
	"github.com/emibotz/chat-server/internal/room"
	"github.com/emibotz/chat-server/internal/store/pgsql"
	"github.com/emibotz/chat-server/internal/store/valkey"
	"github.com/emibotz/chat-server/internal/user"
	"github.com/emibotz/chat-server/pkg/logger"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

func main() {

	// 创建请求处理器
	e := echo.New()
	e.Use(echoMiddleware.RequestLogger())
	e.Use(echoMiddleware.Recover())

	wsHandler := network.NewServer()

	// 设置结构化日志
	slog.SetDefault(e.Logger)

	// 创建根上下文
	slog.Info("Create and notify context.")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// 读取环境变量
	slog.Info("Load environment variables.")
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	// 加载认证配置
	slog.Info("Load auth configs.")
	user.Config.Load()
	user.Pepper = os.Getenv("AUTH_PEPPER")

	// 创建 Redis 数据库仓库
	// redisAddr := os.Getenv("REDIS_ADDR")

	// redisDB, err := redis.New(ctx, redisAddr)
	// if err != nil {
	// 	panic(err)
	// }

	// sessions := redisDB.Sessions()

	// 创建 Valkey 数据库仓库
	slog.Info("Create valkey store.")
	valkeyAddr := os.Getenv("VALKEY_ADDR")

	valkeyDB, err := valkey.New(ctx, valkeyAddr)
	if err != nil {
		panic(err)
	}

	sessions := valkeyDB.Sessions()

	// 创建 Postgresql 数据库仓库
	slog.Info("Create postgresql store.")
	connString := os.Getenv("CONN_STRING")

	pgsqlDB, err := pgsql.New(ctx, connString)
	if err != nil {
		panic(err)
	}

	users := pgsqlDB.Users()

	// 创建服务
	slog.Info("Create and configuring services.")
	gameService := game.NewService()
	userService := user.NewService(sessions, users)
	roomService := room.NewService(userService, gameService)

	// 配置游戏服务使用中间件
	gameService.AddMiddlewareFactory(game.AlwaysUse(tickMiddleware.Broadcast(wsHandler.Broadcaster())))

	// 创建 HTTP 请求处理器
	slog.Info("Create HTTP handlers.")
	gameHandler := game.NewHandler(gameService)
	userHandler := user.NewHandler(userService)
	roomHandler := room.NewHandler(userService, roomService)

	// 创建 WebSocket 请求处理器
	slog.Info("Create WebSocket handlers.")
	wsHandler.HandleFunc(gameHandler.HandleWS)
	wsHandler.HandleFunc(userHandler.HandleWS)
	wsHandler.HandleFunc(roomHandler.HandleWS)

	// 创建路由
	slog.Info("Create routes.")
	apiRoute := e.Group("/api")
	{
		userRoute := apiRoute.Group("/user")
		{
			userRoute.POST("/register", userHandler.Register)
			userRoute.POST("/login", userHandler.Login)
		}
	}

	e.GET("/ws", wsHandler.Handle, middleware.Auth(userService))

	// 读取服务器监听地址
	serverAddr := os.Getenv("ADDR")

	// 创建 HTTP 服务器
	slog.Info("Create HTTP Server.")
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      e,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器，优雅退出
	slog.Info("Server is on.",
		slog.String("listening", serverAddr),
	)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				stop()
				return
			}

			panic(err)
		}
	}()

	// 手动退出流程
	<-ctx.Done()

	fmt.Println()
	slog.Info("Received termination signal, shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("error occurred when shutting down server", err)
	}

	slog.Info("Server is down.")
}
