package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emibotz/chat-server/internal/store/pgsql"
	"github.com/emibotz/chat-server/internal/store/redis"
	"github.com/emibotz/chat-server/internal/user"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// [TODO]
func handleWebSocket(c *echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Close()
}

func main() {
	// 创建根上下文
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// 读取环境变量
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	// 加载认证配置
	user.Config.Load()

	// 创建 Redis 数据库仓库
	redisAddr := os.Getenv("REDIS_ADDR")

	redisDB, err := redis.New(ctx, redisAddr)
	if err != nil {
		panic(err)
	}

	sessions := redisDB.Sessions()

	// 创建 Postgresql 数据库仓库
	connString := os.Getenv("CONN_STRING")

	pgsqlDB, err := pgsql.New(ctx, connString)
	if err != nil {
		panic(err)
	}

	users := pgsqlDB.Users()

	// 创建用户服务
	userService := user.NewService(sessions, users)

	// 创建用户请求处理器
	userHandler := user.NewHandler(userService)

	// 读取服务器监听地址
	serverAddr := os.Getenv("ADDR")

	// 创建请求处理器
	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/ws", handleWebSocket)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      e,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 创建路由
	userRoute := e.Group("/user")
	{
		userRoute.POST("/register", userHandler.Register)
		userRoute.POST("/login", userHandler.Login)
	}

	// 启动服务器，优雅退出
	fmt.Printf("Server is on, listening %s.\n", serverAddr)
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

	fmt.Println("Received termination signal, shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("err occurred when shutting down server: %v\n", err)
	}

	fmt.Println("Server is down.")
}
