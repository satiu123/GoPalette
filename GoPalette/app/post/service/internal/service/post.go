package service

import (
	"context"

	"github.com/satiu123/GoPalette/pkg/auth"

	pb "github.com/satiu123/GoPalette/api/post/v1"
	searchpb "github.com/satiu123/GoPalette/api/search/v1"
	userpb "github.com/satiu123/GoPalette/api/user/v1"

	"github.com/satiu123/GoPalette/app/post/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PostService struct {
	pb.UnimplementedPostServer

	uc      *biz.PostUsecase
	userc   userpb.UserClient
	searchc searchpb.SearchClient
	logger  *log.Helper
}

func NewPostService(uc *biz.PostUsecase, userc userpb.UserClient, searchc searchpb.SearchClient, logger log.Logger) *PostService {
	return &PostService{
		uc:      uc,
		userc:   userc,
		searchc: searchc,
		logger:  log.NewHelper(log.With(logger, "module", "service/post")),
	}
}

func (s *PostService) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostReply, error) {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return nil, pb.ErrorUnauthenticated("%s", "未认证的用户")
	}

	p := &biz.Post{
		Title:   req.Title,
		Summary: req.Summary,
		Content: req.Content,
		Slug:    req.Slug,
		Status:  int32(req.Status),

		AuthorID:   claims.UserID,
		CategoryID: req.CategoryId,
		Tags:       req.Tags,
	}
	createdPost, err := s.uc.CreatePost(ctx, p)
	if err != nil {
		return nil, err
	}
	if pb.PostStatus(createdPost.Status) == pb.PostStatus_PUBLISHED {
		if _, syncErr := s.searchc.SyncPost(ctx, s.toSearchSyncReq(createdPost)); syncErr != nil {
			s.logger.WithContext(ctx).Warnf("同步搜索索引失败(create): %v", syncErr)
		}
	}
	return &pb.CreatePostReply{
		Post: s.toPBDetail(createdPost),
	}, nil
}
func (s *PostService) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.UpdatePostReply, error) {
	p := &biz.Post{
		ID:      req.Id,
		Title:   req.Title,
		Summary: req.Summary,
		Content: req.Content,
		Slug:    req.Slug,
		Status:  int32(req.Status),

		CategoryID: req.CategoryId,
		Tags:       req.Tags,
	}
	updatedPost, err := s.uc.UpdatePost(ctx, p, req.UpdateMask.GetPaths())
	if err != nil {
		return nil, err
	}
	if pb.PostStatus(updatedPost.Status) == pb.PostStatus_PUBLISHED {
		if _, syncErr := s.searchc.SyncPost(ctx, s.toSearchSyncReq(updatedPost)); syncErr != nil {
			s.logger.WithContext(ctx).Warnf("同步搜索索引失败(update): %v", syncErr)
		}
	} else {
		if _, delErr := s.searchc.DeleteIndex(ctx, &searchpb.DeleteIndexRequest{PostId: updatedPost.ID}); delErr != nil {
			s.logger.WithContext(ctx).Warnf("同步删除搜索索引失败(update): %v", delErr)
		}
	}
	return &pb.UpdatePostReply{
		Post: s.toPBDetail(updatedPost),
	}, nil
}
func (s *PostService) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostReply, error) {
	err := s.uc.DeletePost(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if _, delErr := s.searchc.DeleteIndex(ctx, &searchpb.DeleteIndexRequest{PostId: req.Id}); delErr != nil {
		s.logger.WithContext(ctx).Warnf("同步删除搜索索引失败(delete): %v", delErr)
	}
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
	authorsResp, err := s.userc.BatchGetUsers(ctx, &userpb.BatchGetUsersRequest{Ids: []int64{post.AuthorID}})
	if err != nil {
		s.logger.WithContext(ctx).Warnf("批量获取用户信息失败: %v", err)
	} else if len(authorsResp.Users) > 0 {
		pbDetail.Info.Author.Name = authorsResp.Users[0].Username
		pbDetail.Info.Author.AvatarUrl = authorsResp.Users[0].AvatarUrl
	}

	return &pb.GetPostReply{
		Post: pbDetail,
	}, nil
}
func (s *PostService) ListPosts(ctx context.Context, req *pb.ListPostsRequest) (*pb.ListPostsReply, error) {
	res, total, err := s.uc.ListPosts(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	authorSet := make(map[int64]struct{})
	for _, p := range res {
		authorSet[p.AuthorID] = struct{}{}
	}
	authorIDs := make([]int64, 0, len(authorSet))
	for id := range authorSet {
		authorIDs = append(authorIDs, id)
	}
	authors := make(map[int64]*userpb.UserProfile)
	if len(authorIDs) > 0 {
		usersResp, userErr := s.userc.BatchGetUsers(ctx, &userpb.BatchGetUsersRequest{Ids: authorIDs})
		if userErr != nil {
			s.logger.WithContext(ctx).Warnf("批量获取用户信息失败: %v", userErr)
		} else {
			for _, u := range usersResp.Users {
				authors[u.Id] = u
			}
		}
	}

	posts := make([]*pb.PostInfo, len(res))
	for i, p := range res {
		posts[i] = s.toPBInfo(p)
		if u, ok := authors[p.AuthorID]; ok {
			posts[i].Author.Name = u.Username
			posts[i].Author.AvatarUrl = u.AvatarUrl
		}
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

	res, total, err := s.uc.ListPostsForIndex(ctx, req.Page, req.PageSize, publishedOnly)
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.PostIndexInfo, 0, len(res))
	for _, p := range res {
		content := p.Content
		if len(content) > 500 {
			content = content[:500]
		}
		posts = append(posts, &pb.PostIndexInfo{
			Id:           p.ID,
			Title:        p.Title,
			Summary:      p.Summary,
			Content:      content,
			Slug:         p.Slug,
			CategoryName: p.CategoryName,
			Tags:         p.Tags,
			CreatedAt:    timestamppb.New(p.CreatedAt),
			Status:       pb.PostStatus(p.Status),
		})
	}

	return &pb.ListPostsForIndexReply{Posts: posts, Total: total}, nil
}

func (s *PostService) IncrCommentCount(ctx context.Context, req *pb.IncrCommentCountRequest) (*pb.IncrCommentCountReply, error) {
	if err := s.uc.IncrCommentCount(ctx, req.Id, req.Delta); err != nil {
		return nil, err
	}
	return &pb.IncrCommentCountReply{Success: true}, nil
}

func (s *PostService) TogglePostLike(ctx context.Context, req *pb.TogglePostLikeRequest) (*pb.TogglePostLikeReply, error) {
	liked, likeCount, err := s.uc.TogglePostLike(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.TogglePostLikeReply{Liked: liked, LikeCount: likeCount}, nil
}

func (s *PostService) ListUserLikedPosts(ctx context.Context, req *pb.ListUserLikedPostsRequest) (*pb.ListUserLikedPostsReply, error) {
	res, total, err := s.uc.ListUserLikedPosts(ctx, req.UserId, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	authorSet := make(map[int64]struct{})
	for _, p := range res {
		authorSet[p.AuthorID] = struct{}{}
	}
	authorIDs := make([]int64, 0, len(authorSet))
	for id := range authorSet {
		authorIDs = append(authorIDs, id)
	}
	authors := make(map[int64]*userpb.UserProfile)
	if len(authorIDs) > 0 {
		usersResp, userErr := s.userc.BatchGetUsers(ctx, &userpb.BatchGetUsersRequest{Ids: authorIDs})
		if userErr != nil {
			s.logger.WithContext(ctx).Warnf("批量获取用户信息失败: %v", userErr)
		} else {
			for _, u := range usersResp.Users {
				authors[u.Id] = u
			}
		}
	}

	posts := make([]*pb.PostInfo, len(res))
	for i, p := range res {
		posts[i] = s.toPBInfo(p)
		if u, ok := authors[p.AuthorID]; ok {
			posts[i].Author.Name = u.Username
			posts[i].Author.AvatarUrl = u.AvatarUrl
		}
	}
	return &pb.ListUserLikedPostsReply{Posts: posts, Total: total}, nil
}

func (s *PostService) toPBInfo(p *biz.Post) *pb.PostInfo {
	return &pb.PostInfo{
		Id:           p.ID,
		Title:        p.Title,
		Summary:      p.Summary,
		Slug:         p.Slug,
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
			Name: p.CategoryName,
		},
		Tags:      p.Tags,
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func (s *PostService) toPBDetail(p *biz.Post) *pb.PostDetail {
	return &pb.PostDetail{
		Info:            s.toPBInfo(p),
		Content:         p.Content,
		OriginalContent: p.OriginalContent,
	}
}

func (s *PostService) toSearchSyncReq(p *biz.Post) *searchpb.SyncPostRequest {
	content := p.Content
	if len(content) > 500 {
		content = content[:500]
	}
	return &searchpb.SyncPostRequest{
		Id:           p.ID,
		Title:        p.Title,
		Summary:      p.Summary,
		Content:      content,
		Slug:         p.Slug,
		CategoryName: p.CategoryName,
		Tags:         p.Tags,
		CreatedAt:    timestamppb.New(p.CreatedAt),
	}
}
