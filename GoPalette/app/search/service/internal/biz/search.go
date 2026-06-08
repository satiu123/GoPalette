package biz

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	pb "github.com/satiu123/GoPalette/api/search/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/satiu123/GoPalette/pkg/events"
)

type PostSearch struct {
	ID           int64
	Title        string
	Summary      string
	Slug         string
	CategoryName string
	Tags         []string
	CreatedAt    time.Time
}

type SyncPost struct {
	ID           int64
	Title        string
	Summary      string
	Content      string
	Slug         string
	CategoryName string
	Tags         []string
	CreatedAt    time.Time
}

type RebuildTask struct {
	TaskID              string
	Status              string
	ResetFirst          bool
	IncludeNonPublished bool
	IndexedCount        int64
	Total               int64
	ErrorMessage        string
	StartedAt           time.Time
	FinishedAt          time.Time
}

type PostIndexEvent struct {
	Type         string
	PostID       int64
	Title        string
	Summary      string
	Content      string
	Slug         string
	CategoryName string
	Tags         []string
	CreatedAt    time.Time
}

const (
	RebuildStatusRunning   = "RUNNING"
	RebuildStatusSucceeded = "SUCCEEDED"
	RebuildStatusFailed    = "FAILED"
	rebuildPageSize        = 2000
)

type SearchRepo interface {
	SearchPosts(ctx context.Context, query string, offset, limit int64, category string) ([]*PostSearch, int64, error)
	SyncPost(ctx context.Context, p *SyncPost) error
	DeletePost(ctx context.Context, postID int64) error
	ResetIndex(ctx context.Context) error
	SyncPostsBatch(ctx context.Context, posts []*SyncPost) error
}

type PostSourceRepo interface {
	ListPostsByCursor(ctx context.Context, cursorID, pageSize int64, includeNonPublished bool) ([]*SyncPost, int64, int64, bool, error)
}

type PostIndexConsumer interface {
	Start(ctx context.Context, handler func(context.Context, *PostIndexEvent) error) error
}

type SearchUsecase struct {
	repo       SearchRepo
	postSource PostSourceRepo
	logger     *log.Helper

	mu           sync.RWMutex
	tasks        map[string]*RebuildTask
	latestTaskID string
	activeTaskID string
}

func NewSearchUsecase(repo SearchRepo, postSource PostSourceRepo, logger log.Logger) *SearchUsecase {
	return &SearchUsecase{
		repo:       repo,
		postSource: postSource,
		logger:     log.NewHelper(log.With(logger, "module", "usecase/search")),
		tasks:      make(map[string]*RebuildTask),
	}
}

func (uc *SearchUsecase) SearchPosts(ctx context.Context, query string, page, pageSize int64, category string) ([]*PostSearch, int64, int64, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, 0, 0, pb.ErrorInvalidArgument("%s", "搜索关键词不能为空")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	items, total, err := uc.repo.SearchPosts(ctx, query, offset, pageSize, category)
	if err != nil {
		return nil, 0, 0, err
	}
	totalPages := int64(math.Ceil(float64(total) / float64(pageSize)))
	if total == 0 {
		totalPages = 0
	}
	return items, total, totalPages, nil
}

func (uc *SearchUsecase) SyncPost(ctx context.Context, p *SyncPost) error {
	if p == nil || p.ID <= 0 {
		return pb.ErrorInvalidArgument("%s", "post 数据无效")
	}
	if strings.TrimSpace(p.Title) == "" {
		return pb.ErrorInvalidArgument("%s", "标题不能为空")
	}
	return uc.repo.SyncPost(ctx, p)
}

func (uc *SearchUsecase) DeleteIndex(ctx context.Context, postID int64) error {
	if postID <= 0 {
		return pb.ErrorInvalidArgument("%s", "post_id 无效")
	}
	return uc.repo.DeletePost(ctx, postID)
}

func (uc *SearchUsecase) ApplyPostIndexEvent(ctx context.Context, event *PostIndexEvent) error {
	if event == nil || event.PostID <= 0 {
		return pb.ErrorInvalidArgument("%s", "post event 数据无效")
	}
	switch event.Type {
	case events.PostDeleteEvent:
		return uc.DeleteIndex(ctx, event.PostID)
	case events.PostUpsertEvent, "":
		return uc.SyncPost(ctx, &SyncPost{
			ID:           event.PostID,
			Title:        event.Title,
			Summary:      event.Summary,
			Content:      event.Content,
			Slug:         event.Slug,
			CategoryName: event.CategoryName,
			Tags:         event.Tags,
			CreatedAt:    event.CreatedAt,
		})
	default:
		uc.logger.WithContext(ctx).Warnf("忽略未知文章索引事件: type=%s post_id=%d", event.Type, event.PostID)
		return nil
	}
}

func (uc *SearchUsecase) StartRebuildIndex(resetFirst bool, includeNonPublished bool) (*RebuildTask, error) {
	uc.mu.Lock()
	if uc.activeTaskID != "" {
		if active, ok := uc.tasks[uc.activeTaskID]; ok && active.Status == RebuildStatusRunning {
			uc.mu.Unlock()
			return nil, pb.ErrorRebuildInProgress("已有重建任务在执行中: %s", active.TaskID)
		}
	}

	taskID := fmt.Sprintf("rebuild-%d", time.Now().UnixNano())
	task := &RebuildTask{
		TaskID:              taskID,
		Status:              RebuildStatusRunning,
		ResetFirst:          resetFirst,
		IncludeNonPublished: includeNonPublished,
		StartedAt:           time.Now(),
	}
	uc.tasks[taskID] = task
	uc.latestTaskID = taskID
	uc.activeTaskID = taskID
	uc.mu.Unlock()

	go uc.runRebuildTask(taskID)
	return cloneRebuildTask(task), nil
}

func (uc *SearchUsecase) GetRebuildStatus(taskID string) (*RebuildTask, error) {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	if strings.TrimSpace(taskID) == "" {
		taskID = uc.latestTaskID
	}
	if taskID == "" {
		return nil, pb.ErrorInvalidArgument("%s", "暂无重建任务，请先调用 RebuildIndex")
	}
	task, ok := uc.tasks[taskID]
	if !ok {
		return nil, pb.ErrorInvalidArgument("task_id=%s 不存在", taskID)
	}
	return cloneRebuildTask(task), nil
}

func (uc *SearchUsecase) runRebuildTask(taskID string) {
	ctx := context.Background()
	task, ok := uc.getTask(taskID)
	if !ok {
		return
	}

	if task.ResetFirst {
		if err := uc.repo.ResetIndex(ctx); err != nil {
			uc.finishTaskFailed(taskID, err)
			return
		}
	}

	const pageSize int64 = rebuildPageSize
	var cursorID int64
	var indexed int64
	for {
		posts, total, nextCursorID, hasMore, err := uc.postSource.ListPostsByCursor(ctx, cursorID, pageSize, task.IncludeNonPublished)
		if err != nil {
			uc.finishTaskFailed(taskID, err)
			return
		}
		if total > 0 {
			uc.updateTaskTotal(taskID, total)
		}

		if len(posts) == 0 {
			break
		}
		if err := uc.repo.SyncPostsBatch(ctx, posts); err != nil {
			uc.finishTaskFailed(taskID, err)
			return
		}
		indexed += int64(len(posts))
		uc.updateTaskIndexed(taskID, indexed)
		if !hasMore || nextCursorID <= 0 {
			break
		}
		cursorID = nextCursorID
	}
	uc.finishTaskSucceeded(taskID, indexed)
}

func (uc *SearchUsecase) getTask(taskID string) (*RebuildTask, bool) {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	task, ok := uc.tasks[taskID]
	return task, ok
}

func (uc *SearchUsecase) updateTaskTotal(taskID string, total int64) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if task, ok := uc.tasks[taskID]; ok {
		task.Total = total
	}
}

func (uc *SearchUsecase) updateTaskIndexed(taskID string, indexed int64) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if task, ok := uc.tasks[taskID]; ok {
		task.IndexedCount = indexed
	}
}

func (uc *SearchUsecase) finishTaskSucceeded(taskID string, indexed int64) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	task, ok := uc.tasks[taskID]
	if !ok {
		return
	}
	task.Status = RebuildStatusSucceeded
	task.IndexedCount = indexed
	task.ErrorMessage = ""
	task.FinishedAt = time.Now()
	if uc.activeTaskID == taskID {
		uc.activeTaskID = ""
	}
}

func (uc *SearchUsecase) finishTaskFailed(taskID string, err error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	task, ok := uc.tasks[taskID]
	if !ok {
		return
	}
	task.Status = RebuildStatusFailed
	task.ErrorMessage = err.Error()
	task.FinishedAt = time.Now()
	if uc.activeTaskID == taskID {
		uc.activeTaskID = ""
	}
}

func cloneRebuildTask(task *RebuildTask) *RebuildTask {
	if task == nil {
		return nil
	}
	cp := *task
	return &cp
}
