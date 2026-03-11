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
	_ "github.com/satiu123/GoPalette/docs"
	"github.com/satiu123/GoPalette/internal/handler"
	"github.com/satiu123/GoPalette/internal/middleware"
	"github.com/satiu123/GoPalette/internal/pkg/config"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
)

type application struct {
	cfg config.ServerConfig
}

func (app *application) mount(userHandler *handler.UserHandler, articleHandler *handler.ArticleHandler, categoryHandler *handler.CategoryHandler, tagHandler *handler.TagHandler, commentHandler *handler.CommentHandler, searchHandler *handler.SearchHandler) *gin.Engine {
	// 根据环境设置 Gin 模式
	if app.cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.Use(middleware.CORSMiddleware()) // 全局 CORS

	public := r.Group("/api")
	{
		public.GET("/health", handler.HealthCheckHandler)
		public.POST("/login", middleware.RateLimitAuth(), userHandler.Login)
		public.POST("/register", middleware.RateLimitAuth(), userHandler.Register)
		public.POST("/refresh", userHandler.Refresh)
		public.GET("/articles", articleHandler.List)
		public.GET("/articles/:id", articleHandler.Get)
		public.GET("/categories", categoryHandler.List)
		public.GET("/tags", tagHandler.List)
		public.GET("/articles/:id/comments", commentHandler.List)
		public.GET("/search", searchHandler.Search)
		// Swagger UI
		public.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	}

	private := r.Group("/api")
	private.Use(middleware.JWTAuthMiddleware())
	{
		private.POST("/articles", articleHandler.Create)
		private.PUT("/articles/:id", articleHandler.Update)
		private.DELETE("/articles/:id", articleHandler.Delete)
		private.POST("/categories", categoryHandler.Create)
		private.DELETE("/categories/:id", categoryHandler.Delete)
		private.POST("/tags", tagHandler.Create)
		private.DELETE("/tags/:id", tagHandler.Delete)
		private.POST("/articles/:id/comments", commentHandler.Create)
		private.DELETE("/comments/:id", commentHandler.Delete)
	}

	return r
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
