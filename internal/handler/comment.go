package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/service"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(s *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: s}
}

// List 评论列表
// @Summary      获取文章评论
// @Description  按文章 ID 获取所有评论（支持嵌套回复）
// @Tags         comments
// @Produce      json
// @Param        id  path  int  true  "文章 ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /articles/{id}/comments [get]
func (h *CommentHandler) List(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}
	comments, err := h.commentService.ListComments(c.Request.Context(), articleID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, comments)
}

// Create 发表评论
// @Summary      发表评论
// @Description  登录用户对文章进行评论，支持嵌套回复（parent_id != 0）
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int            true  "文章 ID"
// @Param        body  body  CreateCommentReq true  "评论内容"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /articles/{id}/comments [post]
func (h *CommentHandler) Create(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}

	var req CreateCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID := int64(c.GetInt("userID"))
	comment, err := h.commentService.CreateComment(c.Request.Context(), articleID, userID, req.ParentID, req.Content)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, comment)
}

// Delete 删除评论
// @Summary      删除评论
// @Description  评论本人或 admin 可删除
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "评论 ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /comments/{id} [delete]
func (h *CommentHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的评论 ID")
		return
	}

	requesterID := int64(c.GetInt("userID"))
	role := c.GetString("role")

	if err := h.commentService.DeleteComment(c.Request.Context(), id, requesterID, role); err != nil {
		switch err.Error() {
		case "评论不存在":
			response.Error(c, http.StatusNotFound, err.Error())
		case "无权限删除此评论":
			response.Error(c, http.StatusForbidden, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	response.Success(c, gin.H{"message": "评论已删除"})
}
