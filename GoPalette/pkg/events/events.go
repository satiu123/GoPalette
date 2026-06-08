package events

import (
	"encoding/json"
	"strconv"
	"time"
)

const (
	PostIndexStream       = "gopalette.events.post_index"
	CommentCountStream    = "gopalette.events.comment_count"
	SearchConsumerGroup   = "search-indexers"
	PostConsumerGroup     = "post-comment-counters"
	PostUpsertEvent       = "post.upsert"
	PostDeleteEvent       = "post.delete"
	CommentCountEvent     = "comment.count.changed"
	FieldEventType        = "event_type"
	FieldPostID           = "post_id"
	FieldCommentID        = "comment_id"
	FieldDelta            = "delta"
	FieldTitle            = "title"
	FieldSummary          = "summary"
	FieldContent          = "content"
	FieldSlug             = "slug"
	FieldCategoryName     = "category_name"
	FieldTagsJSON         = "tags_json"
	FieldCreatedAtUnixSec = "created_at_unix_sec"
)

func String(values map[string]any, key string) string {
	v, ok := values[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return strconv.FormatInt(Int64(values, key), 10)
	}
}

func Int64(values map[string]any, key string) int64 {
	v, ok := values[key]
	if !ok || v == nil {
		return 0
	}
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	case []byte:
		n, _ := strconv.ParseInt(string(x), 10, 64)
		return n
	default:
		return 0
	}
}

func StringSlice(values map[string]any, key string) []string {
	raw := String(values, key)
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func UnixTime(values map[string]any, key string) time.Time {
	sec := Int64(values, key)
	if sec <= 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}
