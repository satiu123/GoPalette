package data

import (
	"errors"

	"github.com/satiu123/GoPalette/comment-service/internal/conf"

	postv1 "github.com/satiu123/GoPalette/api/post/v1"
	userv1 "github.com/satiu123/GoPalette/api/user/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/extra/redisotel"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewCommentRepo, NewPostRepo, NewUserRepo, NewRateLimitRepo)

// Data .
type Data struct {
	db         *gorm.DB
	rdb        *redis.Client
	postClient postv1.PostClient
	userClient userv1.UserClient
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(logger)
	if c == nil {
		return nil, nil, errors.New("缺少 data 配置")
	}
	if c.Database == nil {
		return nil, nil, errors.New("缺少 data.database 配置")
	}
	if c.Redis == nil {
		return nil, nil, errors.New("缺少 data.redis 配置")
	}
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		helper.Errorf("无法连接数据库: %v", err)
		return nil, nil, err
	}
	if err := db.AutoMigrate(&Comment{}); err != nil {
		helper.Errorf("自动迁移评论表失败: %v", err)
		return nil, nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Db),
		DialTimeout:  c.Redis.DialTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
	})
	rdb.AddHook(redisotel.TracingHook{})

	helper.Info("数据库和Redis连接成功")

	postConn, err := grpc.NewClient(c.Clients.PostEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	helper.Infof("成功连接Post服务: %s", c.Clients.PostEndpoint)

	userConn, err := grpc.NewClient(c.Clients.UserEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = postConn.Close()
		return nil, nil, err
	}
	helper.Infof("成功连接User服务: %s", c.Clients.UserEndpoint)

	d := &Data{
		db:         db,
		rdb:        rdb,
		postClient: postv1.NewPostClient(postConn),
		userClient: userv1.NewUserClient(userConn),
	}

	cleanup := func() {
		helper.Info("message", "close the data resource")
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = rdb.Close()
		_ = postConn.Close()
		_ = userConn.Close()
	}
	return d, cleanup, nil
}
