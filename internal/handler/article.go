package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/repository"
	"github.com/satiu123/GoPalette/internal/service"
)

type ArticleHandler struct {
	articleService *service.ArticleService
}

func NewArticleHandler(s *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: s}
}

// Create 创建文章
// @Summary      创建文章
// @Description  登录用户发布文章（draft 或 published）
// @Tags         articles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body CreateArticleReq true "文章内容"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /articles [post]
func (h *ArticleHandler) Create(c *gin.Context) {
	var req CreateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "")
		return
	}

	authorID := int64(c.GetInt("userID"))

	article, err := h.articleService.CreateArticle(c.Request.Context(), authorID, service.CreateArticleInput{
		Title:      req.Title,
		Summary:    req.Summary,
		Content:    req.Content,
		CategoryID: req.CategoryID,
		TagIDs:     req.TagIDs,
		Status:     req.Status,
	})
	if err != nil {
		response.Internal(c, "创建文章失败")
		return
	}
	response.Success(c, article)
}

// Get 获取文章详情
// @Summary      获取文章详情
// @Description  按 ID 获取文章，同时自动自增阅读数
// @Tags         articles
// @Produce      json
// @Param        id  path  int  true  "文章 ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /articles/{id} [get]
func (h *ArticleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}

	article, err := h.articleService.GetArticle(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, article)
}

// List 文章列表
// @Summary      文章分页列表
// @Description  公开接口，支持按分类、标签过滤和分页
// @Tags         articles
// @Produce      json
// @Param        page         query  int   false  "页码（默认 1）"
// @Param        page_size    query  int   false  "每页条数（默认 10，最大 50）"
// @Param        category_id  query  int   false  "分类 ID"
// @Param        tag_id       query  int   false  "标签 ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /articles [get]
func (h *ArticleHandler) List(c *gin.Context) {
	var req ListArticlesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "")
		return
	}

	articles, total, err := h.articleService.ListArticles(c.Request.Context(), req.Page, req.PageSize,
		repository.ListArticlesFilter{
			CategoryID: req.CategoryID,
			TagID:      req.TagID,
			AuthorID:   req.AuthorID,
			Status:     "published",
		})
	if err != nil {
		response.Internal(c, "获取文章列表失败")
		return
	}
	response.Success(c, gin.H{
		"total":    total,
		"articles": articles,
	})
}

// ListMine 当前登录用户的文章列表（含草稿）
func (h *ArticleHandler) ListMine(c *gin.Context) {
	authorID := int64(c.GetInt("userID"))
	var req struct {
		Page     int `form:"page"`
		PageSize int `form:"page_size"`
	}
	_ = c.ShouldBindQuery(&req)
	articles, total, err := h.articleService.ListArticles(c.Request.Context(), req.Page, req.PageSize,
		repository.ListArticlesFilter{AuthorID: authorID})
	if err != nil {
		response.Internal(c, "获取文章列表失败")
		return
	}
	response.Success(c, gin.H{"total": total, "articles": articles})
}

// AdminList 管理员查看所有文章（含草稿，所有作者）
func (h *ArticleHandler) AdminList(c *gin.Context) {
	if c.GetString("role") != "admin" {
		response.Forbidden(c, "需要管理员权限")
		return
	}
	var req ListArticlesReq
	_ = c.ShouldBindQuery(&req)
	articles, total, err := h.articleService.ListArticles(c.Request.Context(), req.Page, req.PageSize,
		repository.ListArticlesFilter{
			CategoryID: req.CategoryID,
			TagID:      req.TagID,
			AuthorID:   req.AuthorID,
		})
	if err != nil {
		response.Internal(c, "获取文章列表失败")
		return
	}
	response.Success(c, gin.H{"total": total, "articles": articles})
}

// Update 更新文章
// @Summary      更新文章
// @Description  仅作者或 admin 可操作
// @Tags         articles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  int              true  "文章 ID"
// @Param        body body  UpdateArticleReq true  "更新内容"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /articles/{id} [put]
func (h *ArticleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}

	var req UpdateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "")
		return
	}

	requesterID := int64(c.GetInt("userID"))
	role := c.GetString("role")

	article, err := h.articleService.UpdateArticle(c.Request.Context(), id, requesterID, role,
		service.UpdateArticleInput{
			Title:      req.Title,
			Summary:    req.Summary,
			Content:    req.Content,
			CategoryID: req.CategoryID,
			TagIDs:     req.TagIDs,
			Status:     req.Status,
		})
	if err != nil {
		switch err.Error() {
		case "文章不存在":
			response.Error(c, http.StatusNotFound, err.Error())
		case "无权限修改此文章":
			response.Error(c, http.StatusForbidden, err.Error())
		default:
			response.Internal(c, "更新文章失败")
		}
		return
	}
	response.Success(c, article)
}

// Delete 删除文章
// @Summary      删除文章
// @Description  仅作者或 admin 可操作
// @Tags         articles
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "文章 ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /articles/{id} [delete]
func (h *ArticleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}

	requesterID := int64(c.GetInt("userID"))
	role := c.GetString("role")

	if err := h.articleService.DeleteArticle(c.Request.Context(), id, requesterID, role); err != nil {
		switch err.Error() {
		case "文章不存在":
			response.Error(c, http.StatusNotFound, err.Error())
		case "无权限删除此文章":
			response.Error(c, http.StatusForbidden, err.Error())
		default:
			response.Internal(c, "删除文章失败")
		}
		return
	}
	response.Success(c, gin.H{"message": "文章已删除"})
}
