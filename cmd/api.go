package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal"
)

type application struct {
	cfg config
}

type config struct {
	Addr string
	Env  string
}

func (app *application) mount() *gin.Engine {
	// 根据环境设置 Gin 模式
	if app.cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()

	// 健康检查接口
	router.GET("/health", internal.HealthCheckHandler)

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", internal.PingHandler)
		v1.GET("/hello", internal.HelloHandler)
	}

	return router
}

func (app *application) run(router *gin.Engine) error {
	srv := &http.Server{
		Addr:         app.cfg.Addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  time.Minute,
	}

	// 在后台启动服务
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("服务启动在 %s 环境: %s", app.cfg.Addr, app.cfg.Env)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// 监听系统信号（Ctrl+C 或 kill）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-quit:
		log.Printf("收到信号 %s，正在关闭...", sig)
	}

	// 最多等待 5 秒完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("服务已安全关闭")
	return nil
}
