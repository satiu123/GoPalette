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

// Create POST /api/articles（需要 JWT）
func (h *ArticleHandler) Create(c *gin.Context) {
	var req CreateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
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
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, article)
}

// Get GET /api/articles/:id（公开）
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

// List GET /api/articles（公开，仅返回 published 状态）
func (h *ArticleHandler) List(c *gin.Context) {
	var req ListArticlesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	articles, total, err := h.articleService.ListArticles(c.Request.Context(), req.Page, req.PageSize,
		repository.ListArticlesFilter{
			CategoryID: req.CategoryID,
			TagID:      req.TagID,
			Status:     "published",
		})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{
		"total":    total,
		"articles": articles,
	})
}

// Update PUT /api/articles/:id（需要 JWT，仅作者或 admin 可操作）
func (h *ArticleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}

	var req UpdateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
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
			response.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	response.Success(c, article)
}

// Delete DELETE /api/articles/:id（需要 JWT，仅作者或 admin 可操作）
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
			response.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	response.Success(c, gin.H{"message": "文章已删除"})
}
