package main

import (
	"log"
	"log/slog"

	"github.com/satiu123/GoPalette/internal/handler"
	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/pkg/config"
	"github.com/satiu123/GoPalette/internal/pkg/database"
	"github.com/satiu123/GoPalette/internal/repository/mysql"
	tokenredis "github.com/satiu123/GoPalette/internal/repository/redis"
	"github.com/satiu123/GoPalette/internal/service"
)

func main() {
	// 设置日志记录器
	logger := slog.New(slog.NewTextHandler(log.Writer(), nil))
	slog.SetDefault(logger)

	config.LoadConfig()

	app := &application{
		cfg: config.GlobalConfig.Server,
	}

	db := database.InitMySQL(&model.User{}, &model.Category{}, &model.Tag{}, &model.Article{}, &model.ArticleTag{}, &model.Comment{})
	rdb := database.InitRedis()

	userRepo := mysql.NewUserGormRepository(db, rdb)
	tokenRepo := tokenredis.NewTokenRedisRepository(rdb)
	userService := service.NewUserService(userRepo, tokenRepo)
	userHandler := handler.NewUserHandler(userService)

	articleRepo := mysql.NewArticleGormRepository(db)
	articleService := service.NewArticleService(articleRepo)
	articleHandler := handler.NewArticleHandler(articleService)

	categoryRepo := mysql.NewCategoryGormRepository(db)
	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := handler.NewCategoryHandler(categoryService)

	tagRepo := mysql.NewTagGormRepository(db)
	tagService := service.NewTagService(tagRepo)
	tagHandler := handler.NewTagHandler(tagService)

	commentRepo := mysql.NewCommentGormRepository(db)
	commentService := service.NewCommentService(commentRepo)
	commentHandler := handler.NewCommentHandler(commentService)

	searchHandler := handler.NewSearchHandler(articleService)

	// 挂载路由
	router := app.mount(userHandler, articleHandler, categoryHandler, tagHandler, commentHandler, searchHandler)

	// 启动服务
	log.Fatal(app.run(router))
}
