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
	"github.com/emibotz/chat-server/internal/room"
	"github.com/emibotz/chat-server/internal/store/pgsql"
	"github.com/emibotz/chat-server/internal/store/valkey"
	"github.com/emibotz/chat-server/internal/user"
	"github.com/emibotz/chat-server/internal/websocket"
	"github.com/emibotz/chat-server/pkg/logging"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

const (
	// 所有者权限
	OwnerRead  = 0400
	OwnerWrite = 0200
	OwnerExec  = 0100

	// 组权限
	GroupRead  = 0040
	GroupWrite = 0020
	GroupExec  = 0010

	// 其他用户权限
	OtherRead  = 0004
	OtherWrite = 0002
	OtherExec  = 0001
)

func main() {

	// 所有用户可读可写，不可执行
	dirPerm := OwnerRead | OwnerWrite | OwnerExec | GroupRead | GroupWrite | GroupExec | OtherRead | OtherWrite | OtherExec
	filePerm := OwnerRead | OwnerWrite | GroupRead | GroupWrite | OtherRead | OtherWrite

	// 创建日志文件
	today := time.Now().Format(time.DateOnly)
	logFileDir := "./logs"
	logFileName := fmt.Sprintf("%s/%s.log", logFileDir, today)
	latestLogFileName := fmt.Sprintf("%s/%s", logFileDir, "latest.log")

	if err := os.Mkdir(logFileDir, os.FileMode(dirPerm)); err != nil && !errors.Is(err, os.ErrExist) {
		panic(err)
	}

	debugLogFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.FileMode(filePerm))
	if err != nil {
		panic(err)
	}
	defer debugLogFile.Close()

	latestLogFile, err := os.OpenFile(latestLogFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.FileMode(filePerm))
	if err != nil {
		panic(err)
	}
	defer latestLogFile.Close()

	// 创建控制台日志处理器
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
	})

	// 创建日志文件处理器
	debugJSONHandler := slog.NewJSONHandler(debugLogFile, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	latestJSONHandler := slog.NewJSONHandler(latestLogFile, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	// 组合为多处理器
	multiHandler := logging.NewMultiHandler(consoleHandler, debugJSONHandler, latestJSONHandler)

	// 创建日志器，并设置为默认结构化日志器
	logger := slog.New(multiHandler)
	slog.SetDefault(logger)

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

	// 创建请求处理器
	e := echo.New()
	e.Logger = logger
	e.Use(echoMiddleware.RequestLogger())
	e.Use(echoMiddleware.Recover())

	wsHandler := websocket.NewServer()

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
	gameService.AddMiddlewareFactory(tickMiddleware.Broadcast(wsHandler))

	// 创建 HTTP 请求处理器
	slog.Info("Create HTTP handlers.")
	gameHandler := game.NewHandler(gameService)
	userHandler := user.NewHandler(userService)
	roomHandler := room.NewHandler(userService, roomService)

	// 创建 WebSocket 请求处理器
	slog.Info("Create WebSocket handlers.")
	wsHandler.AddHandler(gameHandler)
	wsHandler.AddHandler(userHandler)
	wsHandler.AddHandler(roomHandler)

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
		logging.Error("error occurred when shutting down server", err)
	}

	slog.Info("Server is down.")
}
