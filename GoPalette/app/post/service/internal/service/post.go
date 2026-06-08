package service

import (
	"context"
	"strings"

	"github.com/satiu123/GoPalette/pkg/auth"

	pb "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/app/post/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PostService struct {
	pb.UnimplementedPostServer

	uc                   *biz.PostUsecase
	postEvents           biz.PostEventPublisher
	commentCountConsumer biz.CommentCountConsumer
	logger               *log.Helper
}

func NewPostService(uc *biz.PostUsecase, postEvents biz.PostEventPublisher, commentCountConsumer biz.CommentCountConsumer, logger log.Logger) *PostService {
	s := &PostService{
		uc:                   uc,
		postEvents:           postEvents,
		commentCountConsumer: commentCountConsumer,
		logger:               log.NewHelper(log.With(logger, "module", "service/post")),
	}
	s.startCommentCountConsumer()
	return s
}

func (s *PostService) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostReply, error) {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return nil, pb.ErrorUnauthenticated("%s", "未认证的用户")
	}

	p := &biz.Post{
		Title:    req.Title,
		Summary:  req.Summary,
		Content:  req.Content,
		CoverURL: req.CoverUrl,
		Slug:     req.Slug,
		Status:   int32(req.Status),

		AuthorID:   claims.UserID,
		CategoryID: req.CategoryId,
		Tags:       req.Tags,
	}
	createdPost, err := s.uc.CreatePost(ctx, p)
	if err != nil {
		return nil, err
	}
	s.enqueueSearchIndexUpdate("create", createdPost)
	return &pb.CreatePostReply{
		Post: s.toPBDetail(createdPost),
	}, nil
}
func (s *PostService) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.UpdatePostReply, error) {
	p := &biz.Post{
		ID:       req.Id,
		Title:    req.Title,
		Summary:  req.Summary,
		Content:  req.Content,
		CoverURL: req.CoverUrl,
		Slug:     req.Slug,
		Status:   int32(req.Status),

		CategoryID: req.CategoryId,
		Tags:       req.Tags,
	}
	updatedPost, err := s.uc.UpdatePost(ctx, p, req.UpdateMask.GetPaths())
	if err != nil {
		return nil, err
	}
	s.enqueueSearchIndexUpdate("update", updatedPost)
	return &pb.UpdatePostReply{
		Post: s.toPBDetail(updatedPost),
	}, nil
}
func (s *PostService) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostReply, error) {
	err := s.uc.DeletePost(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	s.enqueueSearchIndexDelete("delete", req.Id)
	return &pb.DeletePostReply{Success: true}, nil
}
func (s *PostService) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.GetPostReply, error) {
	var id int64
	var slug string
	switch x := req.Query.(type) {
	case *pb.GetPostRequest_Id:
		id = x.Id
	case *pb.GetPostRequest_Slug:
		slug = x.Slug
	default:
		return nil, pb.ErrorInvalidArgument("%s", "必须提供文章ID或Slug")
	}
	post, err := s.uc.GetPost(ctx, id, slug)
	if err != nil {
		return nil, err
	}
	pbDetail := s.toPBDetail(post)

	return &pb.GetPostReply{
		Post: pbDetail,
	}, nil
}
func (s *PostService) ListPosts(ctx context.Context, req *pb.ListPostsRequest) (*pb.ListPostsReply, error) {
	res, total, err := s.uc.ListPosts(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.PostInfo, len(res))
	for i, p := range res {
		posts[i] = s.toPBInfo(p)
	}

	return &pb.ListPostsReply{
		Posts: posts,
		Total: total,
	}, nil

}

func (s *PostService) ListAuthorPosts(ctx context.Context, req *pb.ListAuthorPostsRequest) (*pb.ListAuthorPostsReply, error) {
	res, total, err := s.uc.ListAuthorPosts(ctx, req.AuthorId, req.Page, req.PageSize, req.IncludeNonPublished)
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.PostInfo, len(res))
	for i, p := range res {
		posts[i] = s.toPBInfo(p)
	}

	return &pb.ListAuthorPostsReply{Posts: posts, Total: total}, nil
}

func (s *PostService) GetAuthorPostStats(ctx context.Context, req *pb.GetAuthorPostStatsRequest) (*pb.GetAuthorPostStatsReply, error) {
	stats, err := s.uc.GetAuthorPostStats(ctx, req.AuthorId, req.IncludeNonPublished)
	if err != nil {
		return nil, err
	}

	return &pb.GetAuthorPostStatsReply{
		Posts:     stats.Posts,
		Published: stats.Published,
		Drafts:    stats.Drafts,
		Archived:  stats.Archived,
		Views:     stats.Views,
		Likes:     stats.Likes,
		Comments:  stats.Comments,
		Private:   stats.Private,
		Offline:   stats.Offline,
	}, nil
}

func (s *PostService) ListTopAuthorPosts(ctx context.Context, req *pb.ListTopAuthorPostsRequest) (*pb.ListTopAuthorPostsReply, error) {
	res, err := s.uc.ListTopAuthorPosts(ctx, req.AuthorId, req.Limit, req.IncludeNonPublished)
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.PostInfo, len(res))
	for i, p := range res {
		posts[i] = s.toPBInfo(p)
	}

	return &pb.ListTopAuthorPostsReply{Posts: posts}, nil
}

func (s *PostService) ListPostsForIndex(ctx context.Context, req *pb.ListPostsForIndexRequest) (*pb.ListPostsForIndexReply, error) {
	publishedOnly := !req.IncludeNonPublished
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 1000
	}

	var (
		res          []*biz.Post
		total        int64
		nextCursorID int64
		hasMore      bool
		err          error
	)
	if req.CursorId > 0 || req.Page == 0 {
		res, total, nextCursorID, hasMore, err = s.uc.ListPostsForIndexByCursor(ctx, req.CursorId, pageSize, publishedOnly)
	} else {
		res, total, err = s.uc.ListPostsForIndex(ctx, req.Page, pageSize, publishedOnly)
		hasMore = int64(len(res)) == pageSize && int64(req.Page)*pageSize < total
		if len(res) > 0 {
			nextCursorID = res[len(res)-1].ID
		}
	}
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.PostIndexInfo, 0, len(res))
	for _, p := range res {
		content := truncateRunes(strings.ToValidUTF8(p.Content, ""), 500)
		posts = append(posts, &pb.PostIndexInfo{
			Id:           p.ID,
			Title:        strings.ToValidUTF8(p.Title, ""),
			Summary:      strings.ToValidUTF8(p.Summary, ""),
			Content:      content,
			Slug:         strings.ToValidUTF8(p.Slug, ""),
			CategoryName: strings.ToValidUTF8(p.CategoryName, ""),
			Tags:         sanitizeStrings(p.Tags),
			CreatedAt:    timestamppb.New(p.CreatedAt),
			Status:       pb.PostStatus(p.Status),
		})
	}

	return &pb.ListPostsForIndexReply{
		Posts:        posts,
		Total:        total,
		NextCursorId: nextCursorID,
		HasMore:      hasMore,
	}, nil
}

func (s *PostService) IncrCommentCount(ctx context.Context, req *pb.IncrCommentCountRequest) (*pb.IncrCommentCountReply, error) {
	if err := s.uc.IncrCommentCount(ctx, req.Id, req.Delta); err != nil {
		return nil, err
	}
	return &pb.IncrCommentCountReply{Success: true}, nil
}

func (s *PostService) RecordPostView(ctx context.Context, req *pb.RecordPostViewRequest) (*pb.RecordPostViewReply, error) {
	counted, viewCount, err := s.uc.RecordPostView(ctx, req.Id, req.ViewerKey)
	if err != nil {
		return nil, err
	}
	return &pb.RecordPostViewReply{
		Counted:   counted,
		ViewCount: viewCount,
	}, nil
}

func (s *PostService) TogglePostLike(ctx context.Context, req *pb.TogglePostLikeRequest) (*pb.TogglePostLikeReply, error) {
	liked, likeCount, err := s.uc.TogglePostLike(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.TogglePostLikeReply{Liked: liked, LikeCount: likeCount}, nil
}

func (s *PostService) GetPostLikeState(ctx context.Context, req *pb.GetPostLikeStateRequest) (*pb.GetPostLikeStateReply, error) {
	liked, likeCount, err := s.uc.GetPostLikeState(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.GetPostLikeStateReply{Liked: liked, LikeCount: likeCount}, nil
}

func (s *PostService) ListUserLikedPosts(ctx context.Context, req *pb.ListUserLikedPostsRequest) (*pb.ListUserLikedPostsReply, error) {
	res, total, err := s.uc.ListUserLikedPosts(ctx, req.UserId, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.PostInfo, len(res))
	for i, p := range res {
		posts[i] = s.toPBInfo(p)
	}
	return &pb.ListUserLikedPostsReply{Posts: posts, Total: total}, nil
}

func (s *PostService) toPBInfo(p *biz.Post) *pb.PostInfo {
	return &pb.PostInfo{
		Id:           p.ID,
		Title:        strings.ToValidUTF8(p.Title, ""),
		Summary:      strings.ToValidUTF8(p.Summary, ""),
		Slug:         strings.ToValidUTF8(p.Slug, ""),
		Status:       pb.PostStatus(p.Status),
		ViewCount:    p.ViewCount,
		LikeCount:    p.LikeCount,
		CommentCount: p.CommentCount,
		Author: &pb.AuthorInfo{
			Id: p.AuthorID,
			// 此处后续通过rpc调用用户服务获取作者名称，目前先使用占位符
		},
		Category: &pb.CategoryInfo{
			Id:   p.CategoryID,
			Name: strings.ToValidUTF8(p.CategoryName, ""),
		},
		Tags:           sanitizeStrings(p.Tags),
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
		CoverUrl:       strings.ToValidUTF8(p.CoverURL, ""),
		ReadingMinutes: estimateReadingMinutes(p.Content, p.Summary),
	}
}

func (s *PostService) toPBDetail(p *biz.Post) *pb.PostDetail {
	return &pb.PostDetail{
		Info:            s.toPBInfo(p),
		Content:         strings.ToValidUTF8(p.Content, ""),
		OriginalContent: strings.ToValidUTF8(p.OriginalContent, ""),
	}
}

func (s *PostService) enqueueSearchIndexUpdate(action string, p *biz.Post) {
	if p == nil {
		return
	}
	if pb.PostStatus(p.Status) != pb.PostStatus_PUBLISHED {
		if action == "create" {
			return
		}
		s.enqueueSearchIndexDelete(action, p.ID)
		return
	}

	if err := s.postEvents.PublishPostUpsert(context.Background(), p); err != nil {
		s.logger.Warnf("发布搜索索引事件失败(%s): post_id=%d err=%v", action, p.ID, err)
	}
}

func (s *PostService) enqueueSearchIndexDelete(action string, postID int64) {
	if postID <= 0 {
		return
	}

	if err := s.postEvents.PublishPostDelete(context.Background(), postID); err != nil {
		s.logger.Warnf("发布搜索索引删除事件失败(%s): post_id=%d err=%v", action, postID, err)
	}
}

func (s *PostService) startCommentCountConsumer() {
	if s.commentCountConsumer == nil {
		return
	}
	go func() {
		if err := s.commentCountConsumer.Start(context.Background(), func(ctx context.Context, postID, delta int64) error {
			return s.uc.IncrCommentCount(ctx, postID, delta)
		}); err != nil {
			s.logger.Warnf("评论数事件消费者退出: %v", err)
		}
	}()
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max])
}

func sanitizeStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, strings.ToValidUTF8(item, ""))
	}
	return out
}
