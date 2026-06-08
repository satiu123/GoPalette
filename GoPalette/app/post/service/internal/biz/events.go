package biz

import "context"

type PostEventPublisher interface {
	PublishPostUpsert(ctx context.Context, p *Post) error
	PublishPostDelete(ctx context.Context, postID int64) error
}

type CommentCountConsumer interface {
	Start(ctx context.Context, handler func(context.Context, int64, int64) error) error
}
