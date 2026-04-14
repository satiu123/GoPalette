package data

import (
	"github.com/satiu123/GoPalette/user-service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/extra/redisotel"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"gorm.io/gorm"

	"gorm.io/driver/mysql"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewUserRepo, NewAuthSessionRepo)

// Data .
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(logger)
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		log.Errorf("无法连接数据库: %v", err)
		return nil, nil, err
	}

	// 自动迁移数据库结构
	if err := db.AutoMigrate(&User{}); err != nil {
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
		db:  db,
		rdb: rdb,
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
