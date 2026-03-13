package model

import "time"

type Comment struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ArticleID int64     `json:"article_id" gorm:"not null;index"`
	UserID    *int64    `json:"user_id" gorm:"index"` // nullable，nil 表示匿名评论
	Content   string    `json:"content" gorm:"type:varchar(500);not null"`
	ParentID  int64     `json:"parent_id" gorm:"default:0;index"` // 0 表示一级评论，添加索引以加快删除性能
	User      *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Article   *Article  `json:"article,omitempty" gorm:"foreignKey:ArticleID"`
	CreatedAt time.Time `json:"created_at"`
}
