package service

import (
	"context"

	pb "GoPalette/api/user/v1"
	"GoPalette/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserService struct {
	pb.UnimplementedUserServer

	uc  *biz.UserUsecase
	ac  *biz.AuthUsecase
	log *log.Helper
}

func NewUserService(uc *biz.UserUsecase, ac *biz.AuthUsecase, logger log.Logger) *UserService {
	return &UserService{
		uc:  uc,
		ac:  ac,
		log: log.NewHelper(log.With(logger, "module", "service/user")),
	}
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserReply, error) {
	u := &biz.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     int32(req.Role),
	}
	createdUser, err := s.uc.CreateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	return &pb.CreateUserReply{
		User: &pb.UserInfo{
			Id:        createdUser.ID,
			Username:  createdUser.Username,
			Email:     createdUser.Email,
			Role:      pb.Role(createdUser.Role),
			AvatarURL: createdUser.AvatarURL,
			Status:    pb.UserStatus(createdUser.Status),
			CreatedAt: timestamppb.New(createdUser.CreatedAt),
			UpdatedAt: timestamppb.New(createdUser.UpdatedAt),
		}}, nil
}

func (s *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterReply, error) {
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return nil, pb.ErrorInvalidArgument("用户名、邮箱和密码不能为空")
	}
	u := &biz.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     int32(pb.Role_ROLE_USER), // 注册用户默认角色为 USER
	}
	createdUser, err := s.uc.Register(ctx, u)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterReply{
		User: &pb.UserInfo{
			Id:        createdUser.ID,
			Username:  createdUser.Username,
			Email:     createdUser.Email,
			Role:      pb.Role(createdUser.Role),
			AvatarURL: createdUser.AvatarURL,
			Status:    pb.UserStatus(createdUser.Status),
			CreatedAt: timestamppb.New(createdUser.CreatedAt),
			UpdatedAt: timestamppb.New(createdUser.UpdatedAt),
		}}, nil
}

func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginReply, error) {
	if req.Email == "" || req.Password == "" {
		return nil, pb.ErrorInvalidArgument("邮箱和密码不能为空")
	}
	accessToken, refreshToken, err := s.ac.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	return &pb.LoginReply{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserReply, error) {
	u := &biz.User{
		ID:       req.Id,
		Username: req.User.Username,
		Email:    req.User.Email,
		Role:     int32(req.User.Role),
		Status:   int32(req.User.Status),
	}
	updatedUser, err := s.uc.UpdateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateUserReply{
		User: &pb.UserInfo{
			Id:        updatedUser.ID,
			Username:  updatedUser.Username,
			Email:     updatedUser.Email,
			Role:      pb.Role(updatedUser.Role),
			AvatarURL: updatedUser.AvatarURL,
			Status:    pb.UserStatus(updatedUser.Status),
			CreatedAt: timestamppb.New(updatedUser.CreatedAt),
			UpdatedAt: timestamppb.New(updatedUser.UpdatedAt),
		}}, nil
}
func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserReply, error) {
	if err := s.uc.DeleteUser(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteUserReply{}, nil
}
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserReply, error) {
	user, err := s.uc.GetUser(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.GetUserReply{
		User: &pb.UserInfo{
			Id:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      pb.Role(user.Role),
			AvatarURL: user.AvatarURL,
			Status:    pb.UserStatus(user.Status),
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		}}, nil
}
func (s *UserService) ListUser(ctx context.Context, req *pb.ListUserRequest) (*pb.ListUserReply, error) {
	users, total, err := s.uc.ListUser(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	pbUsers := make([]*pb.UserInfo, len(users))
	for _, user := range users {
		// 处理 UpdatedAt 字段，确保它不是零值
		var pbUpdatedAt *timestamppb.Timestamp
		if !user.UpdatedAt.IsZero() {
			pbUpdatedAt = timestamppb.New(user.UpdatedAt)
		}

		// 将 biz.User 转换为 pb.UserInfo
		pbUsers = append(pbUsers, &pb.UserInfo{
			Id:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      pb.Role(user.Role),
			AvatarURL: user.AvatarURL,
			Status:    pb.UserStatus(user.Status),
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: pbUpdatedAt,
		})
	}

	return &pb.ListUserReply{
		Users: pbUsers,
		Total: total,
	}, nil
}
