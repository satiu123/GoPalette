package model

import "time"

type Comment struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ArticleID int64     `json:"article_id" gorm:"not null;index"`
	UserID    int64     `json:"user_id" gorm:"not null;index"`
	Content   string    `json:"content" gorm:"type:varchar(500);not null"`
	ParentID  int64     `json:"parent_id" gorm:"default:0"` // 0 表示一级评论
	User      User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	CreatedAt time.Time `json:"created_at"`
}
