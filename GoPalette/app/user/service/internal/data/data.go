package data

import (
	"errors"

	"github.com/euskadi31/wire"
	"github.com/satiu123/GoPalette/app/user/service/internal/conf"
	dbpool "github.com/satiu123/GoPalette/pkg/db"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
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

func (d *Data) DB() *gorm.DB {
	return d.db
}

func (d *Data) Redis() *redis.Client {
	return d.rdb
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "user-service/data"))
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

	// 自动迁移数据库结构
	if err := db.AutoMigrate(&User{}); err != nil {
		helper.Errorf("自动迁移数据库结构失败: %v", err)
		return nil, nil, err
	}
	if err := migrateUserIndexes(db); err != nil {
		helper.Errorf("迁移用户索引失败: %v", err)
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

	d := &Data{
		db:  db,
		rdb: rdb,
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

func migrateUserIndexes(db *gorm.DB) error {
	unique, err := hasUniqueUserNameIndex(db)
	if err != nil {
		return err
	}
	if unique {
		if err := db.Migrator().DropIndex(&User{}, "idx_users_username"); err != nil {
			return err
		}
	}

	if !db.Migrator().HasIndex(&User{}, "idx_users_username") {
		if err := db.Migrator().CreateIndex(&User{}, "Username"); err != nil {
			return err
		}
	}

	return nil
}

func hasUniqueUserNameIndex(db *gorm.DB) (bool, error) {
	var count int64
	err := db.Raw(
		`SELECT COUNT(*)
		   FROM information_schema.statistics
		  WHERE table_schema = DATABASE()
		    AND table_name = ?
		    AND index_name = ?
		    AND non_unique = 0`,
		"users",
		"idx_users_username",
	).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
