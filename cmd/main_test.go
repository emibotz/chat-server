package main_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/emibotz/chat-server/internal/game"
	"github.com/emibotz/chat-server/internal/middleware"
	"github.com/emibotz/chat-server/internal/network"
	"github.com/emibotz/chat-server/internal/room"
	"github.com/emibotz/chat-server/internal/store/mock"
	"github.com/emibotz/chat-server/internal/user"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

func TestMain(t *testing.T) {
	// 创建根上下文
	ctx, timeout := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer timeout()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// 创建仓库
	sessions := mock.SessionStore()
	users := mock.UserStore()

	// 创建服务
	userService := user.NewService(sessions, users)
	gameService := game.NewService()
	roomService := room.NewService(userService, gameService)

	// 创建 HTTP 请求处理器
	userHandler := user.NewHandler(userService)
	roomHandler := room.NewHandler(userService, roomService)

	// 创建 WebSocket 请求处理器
	wsHandler := network.NewServer()
	{
		wsHandler.HandleFunc(userHandler.HandleWS)
		wsHandler.HandleFunc(roomHandler.HandleWS)
	}

	// 创建请求处理器
	e := echo.New()
	e.Use(echoMiddleware.RequestLogger())
	e.Use(echoMiddleware.Recover())

	// 创建路由
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
	serverAddr := ":5678"

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      e,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
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
