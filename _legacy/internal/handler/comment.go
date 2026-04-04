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
		response.Internal(c, "获取评论失败")
		return
	}
	response.Success(c, comments)
}

// Create 发表评论
// @Summary      发表评论
// @Description  已登录用户或匿名访客均可评论；登录用户评论会关联账号，匿名评论显示「匿名用户」
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        id    path  int            true  "文章 ID"
// @Param        body  body  CreateCommentReq true  "评论内容"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /articles/{id}/comments [post]
func (h *CommentHandler) Create(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章 ID")
		return
	}

	var req CreateCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "")
		return
	}

	// JWT 可选：登录用户关联账号，匿名访客 userID=nil
	var userID *int64
	if id, exists := c.Get("userID"); exists {
		v := int64(id.(int))
		userID = &v
	}

	comment, err := h.commentService.CreateComment(c.Request.Context(), articleID, userID, req.ParentID, req.Content)
	if err != nil {
		response.Internal(c, "发表评论失败")
		return
	}
	response.Success(c, comment)
}

// AdminList 管理员查看全部评论
// @Summary      管理员获取所有评论
// @Description  分页返回全部评论，含文章标题和评论者信息
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int  false  "页码，默认 1"
// @Param        page_size  query  int  false  "每页数量，默认 20"
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /admin/comments [get]
func (h *CommentHandler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	comments, total, err := h.commentService.ListAllComments(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Internal(c, "获取评论列表失败")
		return
	}
	response.Success(c, gin.H{
		"comments": comments,
		"total":    total,
	})
}

// ListMine 当前登录用户发表的评论
// @Summary      获取我的评论
// @Description  分页返回当前登录用户发表的评论，含文章标题和评论时间
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int  false  "页码，默认 1"
// @Param        page_size  query  int  false  "每页数量，默认 10，最大 100"
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /comments/mine [get]
func (h *CommentHandler) ListMine(c *gin.Context) {
	userID := int64(c.GetInt("userID"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	comments, total, err := h.commentService.ListMyComments(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Internal(c, "获取我的评论失败")
		return
	}
	response.Success(c, gin.H{
		"comments": comments,
		"total":    total,
	})
}

// ListReceived 当前登录用户文章收到的评论
// @Summary      获取收到的评论
// @Description  分页返回当前登录用户文章收到的评论，含文章标题、评论者信息和评论时间
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int  false  "页码，默认 1"
// @Param        page_size  query  int  false  "每页数量，默认 10，最大 100"
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /comments/received [get]
func (h *CommentHandler) ListReceived(c *gin.Context) {
	userID := int64(c.GetInt("userID"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	comments, total, err := h.commentService.ListReceivedComments(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Internal(c, "获取收到的评论失败")
		return
	}
	response.Success(c, gin.H{
		"comments": comments,
		"total":    total,
	})
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
			response.Internal(c, "删除评论失败")
		}
		return
	}
	response.Success(c, gin.H{"message": "评论已删除"})
}
