package data

import (
	userv1 "GoPalette/api/user/v1"
	"GoPalette/app/post/service/internal/conf"

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
var ProviderSet = wire.NewSet(NewData, NewPostRepo, NewCategoryRepo, NewTagRepo, NewUserClient)

// Data .
type Data struct {
	db         *gorm.DB
	rdb        *redis.Client
	userClient userv1.UserClient
}

func NewUserClient(d *Data) userv1.UserClient {
	return d.userClient
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
	if err := db.AutoMigrate(&Post{}, &Category{}, &Tag{}); err != nil {
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

	userConn, err := grpc.Dial(c.Clients.UserEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	d.userClient = userv1.NewUserClient(userConn)

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
		if err := userConn.Close(); err != nil {
			log.Errorf("failed to close user grpc connection: %v", err)
		}

	}, nil
}
