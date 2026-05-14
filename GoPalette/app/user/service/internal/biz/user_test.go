package biz

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/satiu123/GoPalette/pkg/auth"
	"github.com/satiu123/GoPalette/pkg/pagination"

	pb "github.com/satiu123/GoPalette/api/user/v1"

	"github.com/satiu123/GoPalette/app/user/service/internal/pkg/util"

	"github.com/go-kratos/kratos/v2/log"
	jwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// 1. 成功创建用户（正常流程）
// 2. 创建用户时邮箱已存在（返回邮箱冲突错误）
// 3. 创建用户时密码加密失败（模拟 HashPassword 返回错误，返回内部服务器错误）
// 4. 仓库层创建失败（模拟 repo.Create 返回错误，返回相同错误）
func TestUserUsecase_CreateUser(t *testing.T) {
	ctx := jwt.NewContext(context.Background(), &auth.AuthClaims{
		UserID: 1,
		Role:   int32(pb.Role_ADMIN),
	})

	tests := []struct {
		name         string
		input        *User
		setupMock    func(repo *MockUserRepo, input *User)
		wantErr      bool
		assertErr    func(t *testing.T, err error)
		assertResult func(t *testing.T, got *User, input *User)
	}{
		{
			name: "success",
			input: &User{
				Email:    "new-user@example.com",
				Password: "plain-password",
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				plainPassword := input.Password
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(nil, nil).Once()
				repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(u *User) bool {
					return u != nil &&
						u.Email == input.Email &&
						u.Password != plainPassword &&
						util.CheckPasswordHash(plainPassword, u.Password)
				})).Return(&User{ID: 1, Email: input.Email}, nil).Once()
			},
			assertResult: func(t *testing.T, got *User, input *User) {
				require.NotNil(t, got)
				assert.Equal(t, int64(1), got.ID)
				assert.Equal(t, input.Email, got.Email)
			},
		},
		{
			name: "email conflict",
			input: &User{
				Email:    "exist@example.com",
				Password: "plain-password",
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(&User{ID: 2, Email: input.Email}, nil).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsEmailConflict(err))
			},
		},
		{
			name: "hash password failed",
			input: &User{
				Email:    "hash-failed@example.com",
				Password: strings.Repeat("a", 73),
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(nil, nil).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsInternalServerError(err))
			},
		},
		{
			name: "repo create failed",
			input: &User{
				Email:    "repo-failed@example.com",
				Password: "plain-password",
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(nil, nil).Once()
				repo.EXPECT().Create(mock.Anything, mock.AnythingOfType("*biz.User")).Return(nil, errors.New("create failed")).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "create failed")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			uc := NewUserUsecase(repo, log.DefaultLogger)

			input := *tt.input
			if tt.setupMock != nil {
				tt.setupMock(repo, &input)
			}

			got, err := uc.CreateUser(ctx, &input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.assertResult != nil {
				tt.assertResult(t, got, tt.input)
			}
		})
	}
}

func TestUserUsecase_UpdateUser(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		input        *User
		fields       []string
		setupMock    func(repo *MockUserRepo, input *User, fields []string)
		wantErr      bool
		assertErr    func(t *testing.T, err error)
		assertResult func(t *testing.T, got *User, input *User)
	}{
		// 用户更新自己的基本信息（成功）
		{
			name: "success owner",
			ctx: jwt.NewContext(context.Background(), &auth.AuthClaims{
				UserID: 1,
				Role:   int32(pb.Role_USER),
			}),
			input: &User{
				ID:    1,
				Email: "new-email@example.com",
			},
			fields: []string{"email"},
			setupMock: func(repo *MockUserRepo, input *User, fields []string) {
				repo.EXPECT().Update(mock.Anything, input, fields).Return(input, nil).Once()
			},
			assertResult: func(t *testing.T, got *User, input *User) {
				require.NotNil(t, got)
				assert.Equal(t, input.ID, got.ID)
				assert.Equal(t, input.Email, got.Email)
			},
		},
		// 管理员更新其他用户的基本信息（成功）
		{
			name: "success admin update other user",
			ctx: jwt.NewContext(context.Background(), &auth.AuthClaims{
				UserID: 99,
				Role:   int32(pb.Role_ADMIN),
			}),
			input: &User{
				ID:       2,
				Username: "updated-name",
			},
			fields: []string{"username"},
			setupMock: func(repo *MockUserRepo, input *User, fields []string) {
				repo.EXPECT().Update(mock.Anything, input, fields).Return(input, nil).Once()
			},
			assertResult: func(t *testing.T, got *User, input *User) {
				require.NotNil(t, got)
				assert.Equal(t, input.ID, got.ID)
				assert.Equal(t, input.Username, got.Username)
			},
		},
		// 未认证用户尝试更新信息（未认证错误）
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			input:   &User{ID: 1, Email: "new-email@example.com"},
			fields:  []string{"email"},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsUnauthenticated(err))
			},
		},
		// 用户尝试更新其他用户的信息（访问被拒绝）
		{
			name: "access denied when not owner",
			ctx: jwt.NewContext(context.Background(), &auth.AuthClaims{
				UserID: 3,
				Role:   int32(pb.Role_USER),
			}),
			input:   &User{ID: 2, Email: "new-email@example.com"},
			fields:  []string{"email"},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsAccessDenied(err))
			},
		},
		// 用户尝试更新敏感字段（密码和角色），但不是管理员（成功，敏感字段被忽略）
		{
			name: "reject update when only sensitive fields",
			ctx: jwt.NewContext(context.Background(), &auth.AuthClaims{
				UserID: 1,
				Role:   int32(pb.Role_USER),
			}),
			input: &User{
				ID:       1,
				Password: "new-password",
				Role:     int32(pb.Role_ADMIN),
			},
			fields:  []string{"password", "role"},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsInvalidArgument(err))
			},
		},
		// 用户尝试只更新敏感字段（密码和角色），但不是管理员（失败，没有可更新字段）
		{
			name: "only sensitive fields or no fields left after filtering",
			ctx: jwt.NewContext(context.Background(), &auth.AuthClaims{
				UserID: 1,
				Role:   int32(pb.Role_USER),
			}),
			input: &User{
				ID:       1,
				Password: "new-password",
				Role:     int32(pb.Role_ADMIN),
			},
			fields:  []string{"password", "role"},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsInvalidArgument(err))
			},
		},
		// 数据库更新失败（返回错误）
		{
			name: "repo update failed",
			ctx: jwt.NewContext(context.Background(), &auth.AuthClaims{
				UserID: 1,
				Role:   int32(pb.Role_USER),
			}),
			input:  &User{ID: 1, Email: "new-email@example.com"},
			fields: []string{"email"},
			setupMock: func(repo *MockUserRepo, input *User, fields []string) {
				repo.EXPECT().Update(mock.Anything, input, fields).Return(nil, errors.New("update failed")).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "update failed")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			uc := NewUserUsecase(repo, log.DefaultLogger)

			input := *tt.input
			fields := append([]string(nil), tt.fields...)
			if tt.setupMock != nil {
				tt.setupMock(repo, &input, fields)
			}

			got, err := uc.UpdateUser(tt.ctx, &input, fields)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.assertResult != nil {
				tt.assertResult(t, got, tt.input)
			}
		})
	}
}

func TestUserUsecase_DeleteUser(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		userID    int64
		setupMock func(repo *MockUserRepo, userID int64)
		wantErr   bool
		assertErr func(t *testing.T, err error)
	}{
		// 用户删除自己的账号（成功）
		{
			name:   "success owner delete",
			ctx:    jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 1, Role: int32(pb.Role_USER)}),
			userID: 1,
			setupMock: func(repo *MockUserRepo, userID int64) {
				repo.EXPECT().Delete(mock.Anything, userID).Return(nil).Once()
			},
		},
		// 管理员删除其他用户（成功）
		{
			name:   "success admin delete other user",
			ctx:    jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 99, Role: int32(pb.Role_ADMIN)}),
			userID: 2,
			setupMock: func(repo *MockUserRepo, userID int64) {
				repo.EXPECT().Delete(mock.Anything, userID).Return(nil).Once()
			},
		},
		// 未认证用户尝试删除账号（未认证错误）
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			userID:  1,
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsUnauthenticated(err))
			},
		},
		// 用户尝试删除其他用户（访问被拒绝）
		{
			name:    "access denied when not owner",
			ctx:     jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 3, Role: int32(pb.Role_USER)}),
			userID:  2,
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsAccessDenied(err))
			},
		},
		// 仓库层删除失败（返回错误）
		{
			name:   "repo delete failed",
			ctx:    jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 1, Role: int32(pb.Role_USER)}),
			userID: 1,
			setupMock: func(repo *MockUserRepo, userID int64) {
				repo.EXPECT().Delete(mock.Anything, userID).Return(errors.New("delete failed")).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "delete failed")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			uc := NewUserUsecase(repo, log.DefaultLogger)

			if tt.setupMock != nil {
				tt.setupMock(repo, tt.userID)
			}

			err := uc.DeleteUser(tt.ctx, tt.userID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUserUsecase_GetUser(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		userID       int64
		setupMock    func(repo *MockUserRepo, userID int64)
		wantErr      bool
		assertErr    func(t *testing.T, err error)
		assertResult func(t *testing.T, got *User, userID int64)
	}{
		// 成功获取用户信息 (返回成功)
		{
			name:   "success",
			ctx:    jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 1, Role: int32(pb.Role_USER)}),
			userID: 1,
			setupMock: func(repo *MockUserRepo, userID int64) {
				repo.EXPECT().Get(mock.Anything, userID).Return(&User{ID: userID}, nil).Once()
			},
			assertResult: func(t *testing.T, got *User, userID int64) {
				require.NotNil(t, got)
				assert.Equal(t, userID, got.ID)
			},
		},
		// 用户不存在 (返回用户未找到错误)
		{
			name:   "user not found",
			ctx:    jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 1, Role: int32(pb.Role_USER)}),
			userID: 2,
			setupMock: func(repo *MockUserRepo, userID int64) {
				repo.EXPECT().Get(mock.Anything, userID).Return(nil, pb.ErrorUserNotFound("用户 ID %d 不存在", userID)).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsUserNotFound(err))
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			uc := NewUserUsecase(repo, log.DefaultLogger)

			if tt.setupMock != nil {
				tt.setupMock(repo, tt.userID)
			}

			got, err := uc.GetUser(tt.ctx, tt.userID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.assertResult != nil {
				tt.assertResult(t, got, tt.userID)
			}
		})
	}
}

func TestUserUsecase_Register(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		input        *User
		setupMock    func(repo *MockUserRepo, input *User)
		wantErr      bool
		assertErr    func(t *testing.T, err error)
		assertResult func(t *testing.T, got *User, input *User)
	}{
		{
			name: "success",
			input: &User{
				Email:    "register-user@example.com",
				Password: "plain-password",
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				plainPassword := input.Password
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(nil, nil).Once()
				repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(u *User) bool {
					return u != nil &&
						u.Email == input.Email &&
						u.Password != plainPassword &&
						util.CheckPasswordHash(plainPassword, u.Password)
				})).Return(&User{ID: 10, Email: input.Email}, nil).Once()
			},
			assertResult: func(t *testing.T, got *User, input *User) {
				require.NotNil(t, got)
				assert.Equal(t, int64(10), got.ID)
				assert.Equal(t, input.Email, got.Email)
			},
		},
		{
			name: "find by email returns error but no user",
			input: &User{
				Email:    "register-find-error@example.com",
				Password: "plain-password",
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(nil, errors.New("db timeout")).Once()
				repo.EXPECT().Create(mock.Anything, mock.AnythingOfType("*biz.User")).Return(&User{ID: 11, Email: input.Email}, nil).Once()
			},
			assertResult: func(t *testing.T, got *User, input *User) {
				require.NotNil(t, got)
				assert.Equal(t, int64(11), got.ID)
				assert.Equal(t, input.Email, got.Email)
			},
		},
		{
			name: "email conflict",
			input: &User{
				Email:    "exist@example.com",
				Password: "plain-password",
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(&User{ID: 12, Email: input.Email}, nil).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsEmailConflict(err))
			},
		},
		{
			name: "hash password failed",
			input: &User{
				Email:    "register-hash-failed@example.com",
				Password: strings.Repeat("a", 73),
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(nil, nil).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsInternalServerError(err))
			},
		},
		{
			name: "repo create failed",
			input: &User{
				Email:    "register-repo-failed@example.com",
				Password: "plain-password",
			},
			setupMock: func(repo *MockUserRepo, input *User) {
				repo.EXPECT().FindByEmail(mock.Anything, input.Email).Return(nil, nil).Once()
				repo.EXPECT().Create(mock.Anything, mock.AnythingOfType("*biz.User")).Return(nil, errors.New("create failed")).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "create failed")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			uc := NewUserUsecase(repo, log.DefaultLogger)

			input := *tt.input
			if tt.setupMock != nil {
				tt.setupMock(repo, &input)
			}

			got, err := uc.Register(ctx, &input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.assertResult != nil {
				tt.assertResult(t, got, tt.input)
			}
		})
	}
}

func TestUserUsecase_ListUser(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		page         int64
		pageSize     int64
		setupMock    func(repo *MockUserRepo, page, pageSize int64)
		wantErr      bool
		assertErr    func(t *testing.T, err error)
		assertResult func(t *testing.T, got *pagination.PageResult[*User])
	}{
		{
			name:     "success admin",
			ctx:      jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 99, Role: int32(pb.Role_ADMIN)}),
			page:     1,
			pageSize: 2,
			setupMock: func(repo *MockUserRepo, page, pageSize int64) {
				repo.EXPECT().List(mock.Anything, page, pageSize).Return([]*User{
					{ID: 1, Email: "u1@example.com"},
					{ID: 2, Email: "u2@example.com"},
				}, int64(2), nil).Once()
			},
			assertResult: func(t *testing.T, got *pagination.PageResult[*User]) {
				require.NotNil(t, got)
				assert.Equal(t, int64(2), got.Total)
				assert.Len(t, got.List, 2)
				assert.Equal(t, int64(1), got.List[0].ID)
				assert.Equal(t, int64(2), got.List[1].ID)
			},
		},
		{
			name:     "unauthenticated",
			ctx:      context.Background(),
			page:     1,
			pageSize: 10,
			wantErr:  true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsUnauthenticated(err))
			},
		},
		{
			name:     "access denied when non-admin",
			ctx:      jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 1, Role: int32(pb.Role_USER)}),
			page:     1,
			pageSize: 10,
			wantErr:  true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, pb.IsAccessDenied(err))
			},
		},
		{
			name:     "repo list failed",
			ctx:      jwt.NewContext(context.Background(), &auth.AuthClaims{UserID: 99, Role: int32(pb.Role_ADMIN)}),
			page:     2,
			pageSize: 5,
			setupMock: func(repo *MockUserRepo, page, pageSize int64) {
				repo.EXPECT().List(mock.Anything, page, pageSize).Return(nil, int64(0), errors.New("list failed")).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "list failed")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			uc := NewUserUsecase(repo, log.DefaultLogger)

			if tt.setupMock != nil {
				tt.setupMock(repo, tt.page, tt.pageSize)
			}

			got, err := uc.ListUser(tt.ctx, tt.page, tt.pageSize)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.assertResult != nil {
				tt.assertResult(t, got)
			}
		})
	}
}

func TestUserUsecase_FindByEmail(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		email        string
		setupMock    func(repo *MockUserRepo, email string)
		wantErr      bool
		assertErr    func(t *testing.T, err error)
		assertResult func(t *testing.T, got *User)
	}{
		{
			name:  "success",
			email: "find@example.com",
			setupMock: func(repo *MockUserRepo, email string) {
				repo.EXPECT().FindByEmail(mock.Anything, email).Return(&User{ID: 1, Email: email}, nil).Once()
			},
			assertResult: func(t *testing.T, got *User) {
				require.NotNil(t, got)
				assert.Equal(t, int64(1), got.ID)
				assert.Equal(t, "find@example.com", got.Email)
			},
		},
		{
			name:  "repo error",
			email: "find-error@example.com",
			setupMock: func(repo *MockUserRepo, email string) {
				repo.EXPECT().FindByEmail(mock.Anything, email).Return(nil, errors.New("find failed")).Once()
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "find failed")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepo(t)
			uc := NewUserUsecase(repo, log.DefaultLogger)

			if tt.setupMock != nil {
				tt.setupMock(repo, tt.email)
			}

			got, err := uc.FindByEmail(ctx, tt.email)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.assertResult != nil {
				tt.assertResult(t, got)
			}
		})
	}
}

func TestFilterSensitiveFields(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		want   []string
	}{
		{
			name:   "mixed fields",
			fields: []string{"email", "password", "username", "role"},
			want:   []string{"email", "username"},
		},
		{
			name:   "only sensitive fields",
			fields: []string{"password", "role"},
			want:   []string{},
		},
		{
			name:   "empty fields",
			fields: []string{},
			want:   []string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := filterSensitiveFields(tt.fields)
			assert.Equal(t, tt.want, got)
		})
	}
}
