package data

import (
	"context"
	"errors"
	stdlog "log"
	"os"
	"time"

	"github.com/euskadi31/wire"
	searchv1 "github.com/satiu123/GoPalette/api/search/v1"
	userv1 "github.com/satiu123/GoPalette/api/user/v1"
	dbpool "github.com/satiu123/GoPalette/pkg/db"

	"github.com/satiu123/GoPalette/app/post/service/internal/conf"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"
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

func (d *Data) DB() *gorm.DB {
	return d.db
}

func (d *Data) Redis() *redis.Client {
	return d.rdb
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

func NewSearchClient(reg *etcd.Registry, c *conf.Data) searchv1.SearchClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.SearchEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
		grpc.WithMiddleware(tracing.Client()),
	)
	if err != nil {
		panic(err)
	}
	return searchv1.NewSearchClient(conn)
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, uc userv1.UserClient, sc searchv1.SearchClient) (*Data, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "post-service/data"))
	if c == nil {
		return nil, nil, errors.New("缺少 data 配置")
	}
	if c.Database == nil {
		return nil, nil, errors.New("缺少 data.database 配置")
	}
	if c.Redis == nil {
		return nil, nil, errors.New("缺少 data.redis 配置")
	}

	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{
		Logger: gormlogger.New(
			stdlog.New(os.Stdout, "\r\n", stdlog.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  gormlogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		helper.Errorf("无法连接数据库: %v", err)
		return nil, nil, err
	}
	if sqlDB, err := db.DB(); err == nil {
		dbpool.ConfigurePool(sqlDB)
	} else {
		helper.Errorf("failed to get sqlDB: %v", err)
	}

	// 自动迁移数据库结构
	if err := db.AutoMigrate(&Post{}, &Category{}, &Tag{}, &PostLike{}); err != nil {
		helper.Errorf("自动迁移数据库结构失败: %v", err)
		return nil, nil, err
	}
	helper.Info("数据库连接成功并完成自动迁移")

	// 启用 GORM 的 OpenTelemetry 插件
	if err := db.Use(gormtracing.NewPlugin()); err != nil {
		helper.Errorf("failed to enable gorm tracing: %v", err)
		return nil, nil, err
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Db),
		DialTimeout:  c.Redis.DialTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})

	if err := redisotel.InstrumentTracing(rdb, redisotel.WithDialFilter(true)); err != nil {
		helper.Errorf("无法启用 Redis tracing: %v", err)
		return nil, nil, err
	}

	d := &Data{
		db:           db,
		rdb:          rdb,
		userClient:   uc,
		searchClient: sc,
	}

	return d, func() {
		helper.Info("message", "close the data resource")
		if sqlDB, err := db.DB(); err != nil {
			helper.Errorf("failed to get sqlDB: %v", err)
		} else {
			if err := sqlDB.Close(); err != nil {
				helper.Errorf("failed to close database: %v", err)
			}
		}
		if err := d.rdb.Close(); err != nil {
			helper.Errorf("failed to close redis: %v", err)
		}

	}, nil
}
