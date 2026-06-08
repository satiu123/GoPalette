package data

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/satiu123/GoPalette/app/post/service/internal/biz"
	"github.com/satiu123/GoPalette/pkg/events"
)

type postEventPublisher struct {
	data *Data
}

func NewPostEventPublisher(data *Data) biz.PostEventPublisher {
	return &postEventPublisher{data: data}
}

func (p *postEventPublisher) PublishPostUpsert(ctx context.Context, post *biz.Post) error {
	if post == nil || post.ID <= 0 {
		return nil
	}
	tags, _ := json.Marshal(post.Tags)
	values := map[string]any{
		events.FieldEventType:        events.PostUpsertEvent,
		events.FieldPostID:           strconv.FormatInt(post.ID, 10),
		events.FieldTitle:            post.Title,
		events.FieldSummary:          post.Summary,
		events.FieldContent:          post.Content,
		events.FieldSlug:             post.Slug,
		events.FieldCategoryName:     post.CategoryName,
		events.FieldTagsJSON:         string(tags),
		events.FieldCreatedAtUnixSec: strconv.FormatInt(post.CreatedAt.Unix(), 10),
	}
	return p.data.eventRDB.XAdd(ctx, &redis.XAddArgs{
		Stream: events.PostIndexStream,
		Values: values,
	}).Err()
}

func (p *postEventPublisher) PublishPostDelete(ctx context.Context, postID int64) error {
	if postID <= 0 {
		return nil
	}
	return p.data.eventRDB.XAdd(ctx, &redis.XAddArgs{
		Stream: events.PostIndexStream,
		Values: map[string]any{
			events.FieldEventType: events.PostDeleteEvent,
			events.FieldPostID:    strconv.FormatInt(postID, 10),
		},
	}).Err()
}

type commentCountConsumer struct {
	data         *Data
	consumerName string
}

func NewCommentCountConsumer(data *Data) biz.CommentCountConsumer {
	name := strings.TrimSpace(os.Getenv("HOSTNAME"))
	if name == "" {
		name = "post-service"
	}
	return &commentCountConsumer{
		data:         data,
		consumerName: name,
	}
}

func (c *commentCountConsumer) Start(ctx context.Context, handler func(context.Context, int64, int64) error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := c.data.eventRDB.XGroupCreateMkStream(ctx, events.CommentCountStream, events.PostConsumerGroup, "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			if !sleepOrDone(ctx, time.Second) {
				return nil
			}
			continue
		}

		streams, err := c.data.eventRDB.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    events.PostConsumerGroup,
			Consumer: c.consumerName,
			Streams:  []string{events.CommentCountStream, ">"},
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
				postID := events.Int64(message.Values, events.FieldPostID)
				delta := events.Int64(message.Values, events.FieldDelta)
				if postID > 0 && delta != 0 {
					if err := handler(ctx, postID, delta); err != nil {
						continue
					}
				}
				_ = c.data.eventRDB.XAck(ctx, events.CommentCountStream, events.PostConsumerGroup, message.ID).Err()
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
