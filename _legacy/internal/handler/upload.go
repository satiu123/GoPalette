package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/satiu123/GoPalette/internal/pkg/response"
	"github.com/satiu123/GoPalette/internal/pkg/storage"
	"github.com/satiu123/GoPalette/internal/service"
)

// UploadHandler 处理文件上传
type UploadHandler struct {
	store             storage.Storage
	attachmentService *service.AttachmentService
}

// NewUploadHandler 创建上传 handler，注入存储实现（支持本地/OSS 等）
func NewUploadHandler(s storage.Storage, attachmentService *service.AttachmentService) *UploadHandler {
	return &UploadHandler{store: s, attachmentService: attachmentService}
}

// Upload 上传图片
// @Summary      上传图片
// @Description  上传图片文件（JPEG/PNG/WebP/GIF），单张 ≤ 5MB，返回可访问 URL
// @Tags         upload
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        image  formData  file  true  "图片文件"
// @Success      200  {object}  response.Response{data=map[string]string}
// @Failure      400  {object}  response.Response
// @Router       /upload [post]
func (h *UploadHandler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请选择要上传的图片")
		return
	}

	const maxSize = 5 << 20 // 5 MB
	if fileHeader.Size > maxSize {
		response.Error(c, http.StatusBadRequest, "图片大小不能超过 5MB")
		return
	}

	ct := fileHeader.Header.Get("Content-Type")
	switch ct {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
	default:
		response.Error(c, http.StatusBadRequest, "仅支持 JPEG/PNG/WebP/GIF 格式")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "文件读取失败")
		return
	}
	defer file.Close()

	url, err := h.store.Save(file, fileHeader.Filename)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "文件保存失败")
		return
	}

	userID := int64(c.GetInt("userID"))
	if err := h.attachmentService.CreateTemporary(c.Request.Context(), userID, url); err != nil {
		response.Error(c, http.StatusInternalServerError, "附件元数据保存失败")
		return
	}

	response.Success(c, gin.H{"url": url})
}
