package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/pkg/config"
	resp "github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/repository"
	"github.com/satiu123/GoPalette/internal/service"
)

type testAPIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type testUserRepo struct {
	findByUsernameFn func(ctx context.Context, username string) (*model.User, error)
	createFn         func(ctx context.Context, user *model.User) error
	findByIDFn       func(ctx context.Context, id int64) (*model.User, error)
	updateFn         func(ctx context.Context, user *model.User) error
}

func (f *testUserRepo) Create(ctx context.Context, user *model.User) error {
	if f.createFn != nil {
		return f.createFn(ctx, user)
	}
	return nil
}
func (f *testUserRepo) FindByID(ctx context.Context, id int64) (*model.User, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (f *testUserRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	if f.findByUsernameFn != nil {
		return f.findByUsernameFn(ctx, username)
	}
	return nil, nil
}
func (f *testUserRepo) Update(ctx context.Context, user *model.User) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, user)
	}
	return nil
}

type testTokenRepo struct {
	setFn    func(ctx context.Context, token string, userID int64, duration time.Duration) error
	getFn    func(ctx context.Context, token string) (int64, error)
	deleteFn func(ctx context.Context, token string) error
}

func (f *testTokenRepo) SetRefreshToken(ctx context.Context, token string, userID int64, duration time.Duration) error {
	if f.setFn != nil {
		return f.setFn(ctx, token, userID, duration)
	}
	return nil
}
func (f *testTokenRepo) GetUserIDByRefreshToken(ctx context.Context, token string) (int64, error) {
	if f.getFn != nil {
		return f.getFn(ctx, token)
	}
	return 0, nil
}
func (f *testTokenRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, token)
	}
	return nil
}

type testArticleRepo struct {
	createFn     func(ctx context.Context, article *model.Article) error
	findByIDFn   func(ctx context.Context, id int64) (*model.Article, error)
	findBySlugFn func(ctx context.Context, slug string) (*model.Article, error)
	searchFn     func(ctx context.Context, keyword string, page, pageSize int) ([]model.Article, int64, error)
}

func (f *testArticleRepo) Create(ctx context.Context, article *model.Article) error {
	if f.createFn != nil {
		return f.createFn(ctx, article)
	}
	return nil
}
func (f *testArticleRepo) FindByID(ctx context.Context, id int64) (*model.Article, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (f *testArticleRepo) FindBySlug(ctx context.Context, slug string) (*model.Article, error) {
	if f.findBySlugFn != nil {
		return f.findBySlugFn(ctx, slug)
	}
	return nil, nil
}
func (f *testArticleRepo) FindAll(ctx context.Context, page, pageSize int, filter repository.ListArticlesFilter) ([]model.Article, int64, error) {
	return nil, 0, nil
}
func (f *testArticleRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]model.Article, int64, error) {
	if f.searchFn != nil {
		return f.searchFn(ctx, keyword, page, pageSize)
	}
	return nil, 0, nil
}
func (f *testArticleRepo) Update(ctx context.Context, article *model.Article) error { return nil }
func (f *testArticleRepo) Delete(ctx context.Context, id int64) error               { return nil }
func (f *testArticleRepo) IncrReadCount(ctx context.Context, id int64) error        { return nil }

type testCommentRepo struct {
	rows []model.Comment
}

func (f *testCommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	comment.ID = int64(len(f.rows) + 1)
	f.rows = append(f.rows, *comment)
	return nil
}
func (f *testCommentRepo) FindByArticleID(ctx context.Context, articleID int64) ([]model.Comment, error) {
	out := make([]model.Comment, 0)
	for _, c := range f.rows {
		if c.ArticleID == articleID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *testCommentRepo) FindByID(ctx context.Context, id int64) (*model.Comment, error) {
	for i := range f.rows {
		if f.rows[i].ID == id {
			return &f.rows[i], nil
		}
	}
	return nil, nil
}
func (f *testCommentRepo) Delete(ctx context.Context, id int64) error { return nil }
func (f *testCommentRepo) FindAll(ctx context.Context, page, pageSize int) ([]model.Comment, int64, error) {
	return f.rows, int64(len(f.rows)), nil
}
func (f *testCommentRepo) FindByUserID(ctx context.Context, userID int64, page, pageSize int) ([]model.Comment, int64, error) {
	return nil, 0, nil
}
func (f *testCommentRepo) FindReceivedByAuthorID(ctx context.Context, authorID int64, page, pageSize int) ([]model.Comment, int64, error) {
	return nil, 0, nil
}

func TestLoginAPIIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}

	config.GlobalConfig = &config.Config{JWT: config.JWTConfig{
		AccessTokenSecret:  "test-access",
		RefreshTokenSecret: "test-refresh",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
	}}

	userSvc := service.NewUserService(&testUserRepo{
		findByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
			return &model.User{ID: 1, Username: username, Password: string(hash), Role: "user"}, nil
		},
	}, &testTokenRepo{})

	r := gin.New()
	r.POST("/login", NewUserHandler(userSvc).Login)

	body := bytes.NewBufferString(`{"username":"alice","password":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body: %s", w.Code, w.Body.String())
	}

	var out testAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if out.Code != http.StatusOK {
		t.Fatalf("unexpected business code: %d", out.Code)
	}
}

func TestArticleCreateAPIIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &testArticleRepo{
		createFn: func(ctx context.Context, article *model.Article) error {
			article.ID = 100
			return nil
		},
	}
	articleSvc := service.NewArticleService(repo, nil)
	h := NewArticleHandler(articleSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 9)
		c.Next()
	})
	r.POST("/articles", h.Create)

	body := bytes.NewBufferString(`{"title":"hello","summary":"s","content":"<p>ok</p><script>alert(1)</script>","status":"published"}`)
	req := httptest.NewRequest(http.MethodPost, "/articles", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body: %s", w.Code, w.Body.String())
	}

	var out testAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if out.Code != http.StatusOK {
		t.Fatalf("unexpected business code: %d", out.Code)
	}

	if bytes.Contains(out.Data, []byte("<script>")) {
		t.Fatal("sanitized content should not contain script tag")
	}
}

func TestCommentAndSearchAPIIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	commentRepo := &testCommentRepo{}
	commentHandler := NewCommentHandler(service.NewCommentService(commentRepo))

	var gotKeyword string
	var gotPage int
	var gotPageSize int
	articleSvc := service.NewArticleService(&testArticleRepo{
		searchFn: func(ctx context.Context, keyword string, page, pageSize int) ([]model.Article, int64, error) {
			gotKeyword, gotPage, gotPageSize = keyword, page, pageSize
			return []model.Article{{ID: 1, Title: "Go 搜索"}}, 1, nil
		},
	}, nil)
	searchHandler := NewSearchHandler(articleSvc)

	r := gin.New()
	r.POST("/articles/:id/comments", commentHandler.Create)
	r.GET("/articles/:id/comments", commentHandler.List)
	r.GET("/search", searchHandler.Search)

	// 创建评论
	createReq := httptest.NewRequest(http.MethodPost, "/articles/1/comments", bytes.NewBufferString(`{"content":"nice"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("create comment status=%d body=%s", createW.Code, createW.Body.String())
	}

	// 列表查询
	listReq := httptest.NewRequest(http.MethodGet, "/articles/1/comments", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list comments status=%d body=%s", listW.Code, listW.Body.String())
	}

	var listOut testAPIResponse
	if err := json.Unmarshal(listW.Body.Bytes(), &listOut); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	if listOut.Code != http.StatusOK {
		t.Fatalf("unexpected list code: %d", listOut.Code)
	}

	// 搜索查询
	searchReq := httptest.NewRequest(http.MethodGet, "/search?q=go", nil)
	searchW := httptest.NewRecorder()
	r.ServeHTTP(searchW, searchReq)
	if searchW.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", searchW.Code, searchW.Body.String())
	}

	if gotKeyword != "go" || gotPage != 1 || gotPageSize != 10 {
		t.Fatalf("unexpected search params: keyword=%q page=%d size=%d", gotKeyword, gotPage, gotPageSize)
	}

	var searchOut resp.Response
	if err := json.Unmarshal(searchW.Body.Bytes(), &searchOut); err != nil {
		t.Fatalf("unmarshal search response failed: %v", err)
	}
	if searchOut.Code != http.StatusOK {
		t.Fatalf("unexpected search code: %d", searchOut.Code)
	}
}
