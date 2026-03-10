package main

import (
	"log"
	"log/slog"
)

func main() {
	// 设置日志记录器
	logger := slog.New(slog.NewTextHandler(log.Writer(), nil))
	slog.SetDefault(logger)

	cfg := config{
		Addr: ":8080",
		Env:  "development",
	}

	app := &application{
		cfg: cfg,
	}

	// 挂载路由
	router := app.mount()

	// 启动服务
	log.Fatal(app.run(router))
}
