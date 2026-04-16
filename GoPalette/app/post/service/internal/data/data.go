package data

import (
	"context"
	"errors"

	"github.com/euskadi31/wire"
	searchv1 "github.com/satiu123/GoPalette/api/search/v1"
	userv1 "github.com/satiu123/GoPalette/api/user/v1"

	"github.com/satiu123/GoPalette/app/post/service/internal/conf"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-redis/redis/extra/redisotel"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewPostRepo, NewCategoryRepo, NewTagRepo, NewUserClient, NewSearchClient)

// Data .
type Data struct {
	db           *gorm.DB
	rdb          *redis.Client
	userClient   userv1.UserClient
	searchClient searchv1.SearchClient
}

func NewUserClient(reg *etcd.Registry, c *conf.Data) userv1.UserClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.UserEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
	)
	if err != nil {
		panic(err)
	}
	return userv1.NewUserClient(conn)
}

func NewSearchClient(reg *etcd.Registry, c *conf.Data) searchv1.SearchClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.SearchEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
	)
	if err != nil {
		panic(err)
	}
	return searchv1.NewSearchClient(conn)
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, uc userv1.UserClient, sc searchv1.SearchClient) (*Data, func(), error) {
	log := log.NewHelper(logger)
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
		log.Errorf("无法连接数据库: %v", err)
		return nil, nil, err
	}

	// 自动迁移数据库结构
	if err := db.AutoMigrate(&Post{}, &Category{}, &Tag{}, &PostLike{}); err != nil {
		log.Errorf("自动迁移数据库结构失败: %v", err)
		return nil, nil, err
	}
	log.Info("数据库连接成功并完成自动迁移")

	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Db),
		DialTimeout:  c.Redis.DialTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
	})
	rdb.AddHook(redisotel.TracingHook{})
	d := &Data{
		db:           db,
		rdb:          rdb,
		userClient:   uc,
		searchClient: sc,
	}

	return d, func() {
		log.Info("message", "close the data resource")
		if sqlDB, err := db.DB(); err != nil {
			log.Errorf("failed to get sqlDB: %v", err)
		} else {
			if err := sqlDB.Close(); err != nil {
				log.Errorf("failed to close database: %v", err)
			}
		}
		if err := d.rdb.Close(); err != nil {
			log.Errorf("failed to close redis: %v", err)
		}

	}, nil
}
