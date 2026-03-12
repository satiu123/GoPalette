package handler

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterReq struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role"` // 可选，默认 user
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type TokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserResp struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type CreateArticleReq struct {
	Title      string  `json:"title" binding:"required,max=200"`
	Summary    string  `json:"summary" binding:"max=500"`
	Content    string  `json:"content" binding:"required"`
	CategoryID int64   `json:"category_id"`
	TagIDs     []int64 `json:"tag_ids"`
	Status     string  `json:"status"` // draft | published
}

type UpdateArticleReq struct {
	Title      string  `json:"title" binding:"required,max=200"`
	Summary    string  `json:"summary" binding:"max=500"`
	Content    string  `json:"content" binding:"required"`
	CategoryID int64   `json:"category_id"`
	TagIDs     []int64 `json:"tag_ids"`
	Status     string  `json:"status"`
}

type ListArticlesReq struct {
	Page       int   `form:"page"`
	PageSize   int   `form:"page_size"`
	CategoryID int64 `form:"category_id"`
	TagID      int64 `form:"tag_id"`
	AuthorID   int64 `form:"author_id"`
}

type CreateCategoryReq struct {
	Name string `json:"name" binding:"required,max=50"`
}

type CreateTagReq struct {
	Name string `json:"name" binding:"required,max=50"`
}

type CreateCommentReq struct {
	Content  string `json:"content" binding:"required,max=500"`
	ParentID int64  `json:"parent_id"` // 0 表示一级评论
}

type SearchReq struct {
	Q        string `form:"q" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}
