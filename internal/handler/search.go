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

// Search 全文搜索
// @Summary      全文搜索
// @Description  基于 MySQL FULLTEXT 索引（倒排索引原理）搜索文章标题和正文
// @Tags         search
// @Produce      json
// @Param        q          query  string  true   "搜索关键词"
// @Param        page       query  int     false  "页码"
// @Param        page_size  query  int     false  "每页条数"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /search [get]
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
