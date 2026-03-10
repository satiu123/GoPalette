package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/service"
)

type TagHandler struct {
	tagService *service.TagService
}

func NewTagHandler(s *service.TagService) *TagHandler {
	return &TagHandler{tagService: s}
}

// List GET /api/tags（公开）
func (h *TagHandler) List(c *gin.Context) {
	tags, err := h.tagService.ListTags(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, tags)
}

// Create POST /api/tags（需要 JWT，admin）
func (h *TagHandler) Create(c *gin.Context) {
	var req CreateTagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	tag, err := h.tagService.CreateTag(c.Request.Context(), req.Name)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, tag)
}

// Delete DELETE /api/tags/:id（需要 JWT，admin）
func (h *TagHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的标签 ID")
		return
	}
	if err := h.tagService.DeleteTag(c.Request.Context(), id); err != nil {
		if err.Error() == "标签不存在" {
			response.Error(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "标签已删除"})
}
