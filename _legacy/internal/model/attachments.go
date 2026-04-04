package model

import "time"

// Attachment 记录上传图片元数据，post_id=0 表示暂存未绑定文章
type Attachment struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int64     `json:"user_id" gorm:"not null;index"`
	PostID    int64     `json:"post_id" gorm:"not null;default:0;index"`
	URL       string    `json:"url" gorm:"type:varchar(255);not null;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
