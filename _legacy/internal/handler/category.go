package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/service"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(s *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: s}
}

// List 分类列表
// @Summary      获取所有分类
// @Tags         categories
// @Produce      json
// @Success      200  {object}  response.Response
// @Router       /categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.categoryService.ListCategories(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, categories)
}

// Create 创建分类
// @Summary      创建分类
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body CreateCategoryReq true "分类名称"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req CreateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	category, err := h.categoryService.CreateCategory(c.Request.Context(), req.Name)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, category)
}

// Delete 删除分类
// @Summary      删除分类
// @Tags         categories
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "分类 ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的分类 ID")
		return
	}
	if err := h.categoryService.DeleteCategory(c.Request.Context(), id); err != nil {
		if err.Error() == "分类不存在" {
			response.Error(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "分类已删除"})
}
