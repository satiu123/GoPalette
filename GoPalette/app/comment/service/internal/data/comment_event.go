package data

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/satiu123/GoPalette/app/comment/service/internal/biz"
	"github.com/satiu123/GoPalette/pkg/events"
)

type commentEventPublisher struct {
	data *Data
}

func NewCommentEventPublisher(data *Data) biz.CommentEventPublisher {
	return &commentEventPublisher{data: data}
}

func (p *commentEventPublisher) PublishCommentCountChanged(ctx context.Context, postID, commentID, delta int64) error {
	if postID <= 0 || delta == 0 {
		return nil
	}
	return p.data.eventRDB.XAdd(ctx, &redis.XAddArgs{
		Stream: events.CommentCountStream,
		Values: map[string]any{
			events.FieldEventType: events.CommentCountEvent,
			events.FieldPostID:    strconv.FormatInt(postID, 10),
			events.FieldCommentID: strconv.FormatInt(commentID, 10),
			events.FieldDelta:     strconv.FormatInt(delta, 10),
		},
	}).Err()
}
