package data

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/euskadi31/wire"
	postv1 "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/app/search/service/internal/conf"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"github.com/satiu123/GoPalette/app/search/service/internal/biz"
	"github.com/satiu123/GoPalette/pkg/events"
	grpcgo "google.golang.org/grpc"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewSearchRepo, NewPostSourceRepo, NewPostClient, NewPostIndexConsumer)

// Data .
type Data struct {
	meili      meilisearch.ServiceManager
	postClient postv1.PostClient
	indexName  string
	rdb        *redis.Client
	eventRDB   *redis.Client
}

func (d *Data) Meili() meilisearch.ServiceManager {
	return d.meili
}

func (d *Data) Redis() *redis.Client {
	return d.rdb
}

func (d *Data) EventRedis() *redis.Client {
	return d.eventRDB
}

func NewPostClient(reg *etcd.Registry, c *conf.Data) postv1.PostClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(c.Clients.PostEndpoint),
		grpc.WithTimeout(c.Clients.Timeout.AsDuration()),
		grpc.WithDiscovery(reg),
		grpc.WithMiddleware(tracing.Client()),
		grpc.WithOptions(
			grpcgo.WithDefaultCallOptions(
				grpcgo.MaxCallRecvMsgSize(32*1024*1024),
			),
		),
	)
	if err != nil {
		panic(err)
	}
	return postv1.NewPostClient(conn)
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, pc postv1.PostClient) (*Data, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "search-service/data"))
	if c == nil {
		return nil, nil, errors.New("缺少 data 配置")
	}
	if c.Meilisearch == nil {
		return nil, nil, errors.New("缺少 data.meilisearch 配置")
	}
	ms := meilisearch.New(c.Meilisearch.Endpoint, meilisearch.WithAPIKey(c.Meilisearch.ApiKey))

	if c.Meilisearch.IndexName == "" {
		c.Meilisearch.IndexName = "posts"
	}
	if c.Redis == nil {
		return nil, nil, errors.New("缺少 data.redis 配置")
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

	d := &Data{
		meili:      ms,
		postClient: pc,
		indexName:  c.Meilisearch.IndexName,
		rdb:        rdb,
		eventRDB:   eventRDB,
	}

	if err := initIndexSettings(d); err != nil {
		helper.Warnf("初始化索引设置失败: %v", err)
	}

	cleanup := func() {
		helper.Info("closing search data resources")
		_ = rdb.Close()
		_ = eventRDB.Close()
	}
	return d, cleanup, nil
}

type postIndexConsumer struct {
	data         *Data
	consumerName string
}

func NewPostIndexConsumer(data *Data) biz.PostIndexConsumer {
	name := strings.TrimSpace(os.Getenv("HOSTNAME"))
	if name == "" {
		name = "search-service"
	}
	return &postIndexConsumer{
		data:         data,
		consumerName: name,
	}
}

func (c *postIndexConsumer) Start(ctx context.Context, handler func(context.Context, *biz.PostIndexEvent) error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := c.data.eventRDB.XGroupCreateMkStream(ctx, events.PostIndexStream, events.SearchConsumerGroup, "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			if !sleepOrDone(ctx, time.Second) {
				return nil
			}
			continue
		}

		streams, err := c.data.eventRDB.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    events.SearchConsumerGroup,
			Consumer: c.consumerName,
			Streams:  []string{events.PostIndexStream, ">"},
			Count:    16,
			Block:    time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			if !sleepOrDone(ctx, time.Second) {
				return nil
			}
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				event := &biz.PostIndexEvent{
					Type:         events.String(message.Values, events.FieldEventType),
					PostID:       events.Int64(message.Values, events.FieldPostID),
					Title:        events.String(message.Values, events.FieldTitle),
					Summary:      events.String(message.Values, events.FieldSummary),
					Content:      events.String(message.Values, events.FieldContent),
					Slug:         events.String(message.Values, events.FieldSlug),
					CategoryName: events.String(message.Values, events.FieldCategoryName),
					Tags:         events.StringSlice(message.Values, events.FieldTagsJSON),
					CreatedAt:    events.UnixTime(message.Values, events.FieldCreatedAtUnixSec),
				}
				if event.Type == "" {
					event.Type = events.PostUpsertEvent
				}
				if event.PostID > 0 {
					if err := handler(ctx, event); err != nil {
						continue
					}
				}
				_ = c.data.eventRDB.XAck(ctx, events.PostIndexStream, events.SearchConsumerGroup, message.ID).Err()
			}
		}
	}
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
