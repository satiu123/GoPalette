package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/service"
)

type SearchHandler struct {
	articleService *service.ArticleService
}

func NewSearchHandler(s *service.ArticleService) *SearchHandler {
	return &SearchHandler{articleService: s}
}

// Search GET /api/search?q=xxx&page=1&page_size=10（公开）
func (h *SearchHandler) Search(c *gin.Context) {
	var req SearchReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	articles, total, err := h.articleService.SearchArticles(c.Request.Context(), req.Q, req.Page, req.PageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{
		"total":    total,
		"articles": articles,
	})
}
