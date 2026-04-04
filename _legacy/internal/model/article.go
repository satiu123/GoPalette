package model

import "time"

type Article struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Slug       string    `json:"slug" gorm:"type:varchar(220);not null;index"`
	Title      string    `json:"title" gorm:"type:varchar(200);not null;index"`
	Summary    string    `json:"summary" gorm:"type:varchar(500)"`
	Content    string    `json:"content" gorm:"type:longtext;not null"`
	AuthorID   int64     `json:"author_id" gorm:"not null;index"`
	CategoryID *int64    `json:"category_id" gorm:"index"`
	Status     string    `json:"status" gorm:"type:varchar(20);default:'draft'"` // draft | published
	ReadCount  int64     `json:"read_count" gorm:"default:0"`
	Author     User      `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	Category   Category  `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Tags       []Tag     `json:"tags,omitempty" gorm:"many2many:article_tags;"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
