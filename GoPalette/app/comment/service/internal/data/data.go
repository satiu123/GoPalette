package data

import (
	"errors"

	"github.com/satiu123/GoPalette/app/comment/service/internal/conf"
	dbpool "github.com/satiu123/GoPalette/pkg/db"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"

	"github.com/euskadi31/wire"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewCommentRepo, NewRateLimitRepo, NewCommentEventPublisher)

// Data .
type Data struct {
	db       *gorm.DB
	rdb      *redis.Client
	eventRDB *redis.Client
}

func (d *Data) DB() *gorm.DB {
	return d.db
}

func (d *Data) Redis() *redis.Client {
	return d.rdb
}

func (d *Data) EventRedis() *redis.Client {
	return d.eventRDB
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "comment-service/data"))
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
	eventRedisConf := c.EventRedis
	if eventRedisConf == nil {
		eventRedisConf = &conf.Data_Redis{
			Addr:         c.Redis.Addr,
			Password:     c.Redis.Password,
			Db:           0,
			DialTimeout:  c.Redis.DialTimeout,
			ReadTimeout:  c.Redis.ReadTimeout,
			WriteTimeout: c.Redis.WriteTimeout,
		}
	}
	eventRDB := redis.NewClient(&redis.Options{
		Addr:         eventRedisConf.Addr,
		Password:     eventRedisConf.Password,
		DB:           int(eventRedisConf.Db),
		DialTimeout:  eventRedisConf.DialTimeout.AsDuration(),
		ReadTimeout:  eventRedisConf.ReadTimeout.AsDuration(),
		WriteTimeout: eventRedisConf.WriteTimeout.AsDuration(),
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})

	if err := redisotel.InstrumentTracing(rdb, redisotel.WithDialFilter(true), redisotel.WithCommandFilter(
		func(cmd redis.Cmder) bool {
			// 过滤掉不需要追踪的命令
			if cmd.Name() == "PING" {
				return false
			}
			return true
		})); err != nil {
		helper.Errorf("无法启用 Redis tracing: %v", err)
		return nil, nil, err
	}

	helper.Info("数据库和Redis连接成功")

	d := &Data{db: db, rdb: rdb, eventRDB: eventRDB}

	cleanup := func() {
		helper.Info("message", "close the data resource")
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = rdb.Close()
		_ = eventRDB.Close()
	}
	return d, cleanup, nil
}
