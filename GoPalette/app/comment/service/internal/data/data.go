package data

import (
	"context"
	"errors"

	"github.com/satiu123/GoPalette/app/comment/service/internal/conf"
	dbpool "github.com/satiu123/GoPalette/pkg/db"

	postv1 "github.com/satiu123/GoPalette/api/post/v1"
	userv1 "github.com/satiu123/GoPalette/api/user/v1"

	"github.com/euskadi31/wire"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-redis/redis/extra/redisotel"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewCommentRepo, NewPostRepo, NewUserRepo, NewRateLimitRepo, NewUserClient, NewPostClient)

// Data .
type Data struct {
	db         *gorm.DB
	rdb        *redis.Client
	postClient postv1.PostClient
	userClient userv1.UserClient
}

func NewPostClient(reg *etcd.Registry, c *conf.Data) postv1.PostClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.PostEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
		grpc.WithMiddleware(tracing.Client()),
	)
	if err != nil {
		panic(err)
	}
	return postv1.NewPostClient(conn)
}

func NewUserClient(reg *etcd.Registry, c *conf.Data) userv1.UserClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.UserEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
		grpc.WithMiddleware(tracing.Client()),
	)
	if err != nil {
		panic(err)
	}
	return userv1.NewUserClient(conn)
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, pc postv1.PostClient, uc userv1.UserClient) (*Data, func(), error) {
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
	if sqlDB, err := db.DB(); err == nil {
		dbpool.ConfigurePool(sqlDB)
	} else {
		helper.Errorf("failed to get sqlDB: %v", err)
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

	d := &Data{
		db:         db,
		rdb:        rdb,
		postClient: pc,
		userClient: uc,
	}

	cleanup := func() {
		helper.Info("message", "close the data resource")
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = rdb.Close()
	}
	return d, cleanup, nil
}
