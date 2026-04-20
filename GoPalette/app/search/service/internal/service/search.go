package service

import (
	"context"

	pb "github.com/satiu123/GoPalette/api/search/v1"

	"github.com/satiu123/GoPalette/app/search/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SearchService struct {
	pb.UnimplementedSearchServer

	uc  *biz.SearchUsecase
	log *log.Helper
}

func NewSearchService(uc *biz.SearchUsecase, logger log.Logger) *SearchService {
	return &SearchService{uc: uc, log: log.NewHelper(log.With(logger, "module", "service/search"))}
}

func (s *SearchService) SearchPosts(ctx context.Context, req *pb.SearchPostsRequest) (*pb.SearchPostsReply, error) {
	items, total, totalPages, err := s.uc.SearchPosts(ctx, req.Query, req.Page, req.PageSize, req.Category)
	if err != nil {
		return nil, err
	}
	results := make([]*pb.PostSearchInfo, 0, len(items))
	for _, p := range items {
		results = append(results, &pb.PostSearchInfo{
			Id:           p.ID,
			Title:        p.Title,
			Summary:      p.Summary,
			Slug:         p.Slug,
			CategoryName: p.CategoryName,
			Tags:         p.Tags,
			CreatedAt:    timestamppb.New(p.CreatedAt),
		})
	}
	return &pb.SearchPostsReply{Results: results, Total: total, TotalPages: totalPages}, nil
}

func (s *SearchService) SyncPost(ctx context.Context, req *pb.SyncPostRequest) (*pb.SyncPostReply, error) {
	syncPost := &biz.SyncPost{
		ID:           req.Id,
		Title:        req.Title,
		Summary:      req.Summary,
		Content:      req.Content,
		Slug:         req.Slug,
		CategoryName: req.CategoryName,
		Tags:         req.Tags,
	}
	if req.CreatedAt != nil {
		syncPost.CreatedAt = req.CreatedAt.AsTime()
	}
	if err := s.uc.SyncPost(ctx, syncPost); err != nil {
		return nil, err
	}
	return &pb.SyncPostReply{Success: true}, nil
}

func (s *SearchService) DeleteIndex(ctx context.Context, req *pb.DeleteIndexRequest) (*pb.DeleteIndexReply, error) {
	if err := s.uc.DeleteIndex(ctx, req.PostId); err != nil {
		return nil, err
	}
	return &pb.DeleteIndexReply{Success: true}, nil
}

func (s *SearchService) RebuildIndex(ctx context.Context, req *pb.RebuildIndexRequest) (*pb.RebuildIndexReply, error) {
	task, err := s.uc.StartRebuildIndex(req.ResetFirst, req.IncludeNonPublished)
	if err != nil {
		return nil, err
	}
	return &pb.RebuildIndexReply{
		Accepted: true,
		Task:     toPBRebuildTask(task),
	}, nil
}

func (s *SearchService) GetRebuildStatus(ctx context.Context, req *pb.GetRebuildStatusRequest) (*pb.GetRebuildStatusReply, error) {
	task, err := s.uc.GetRebuildStatus(req.TaskId)
	if err != nil {
		return nil, err
	}
	return &pb.GetRebuildStatusReply{Task: toPBRebuildTask(task)}, nil
}

func toPBRebuildTask(task *biz.RebuildTask) *pb.RebuildTaskInfo {
	if task == nil {
		return nil
	}
	out := &pb.RebuildTaskInfo{
		TaskId:              task.TaskID,
		Status:              task.Status,
		ResetFirst:          task.ResetFirst,
		IncludeNonPublished: task.IncludeNonPublished,
		IndexedCount:        task.IndexedCount,
		Total:               task.Total,
		ErrorMessage:        task.ErrorMessage,
	}
	if !task.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(task.StartedAt)
	}
	if !task.FinishedAt.IsZero() {
		out.FinishedAt = timestamppb.New(task.FinishedAt)
	}
	return out
}
