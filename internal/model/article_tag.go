package model

// ArticleTag 是 Article 与 Tag 的多对多联结表（由 GORM many2many 自动管理）
type ArticleTag struct {
	ArticleID int64 `gorm:"primaryKey"`
	TagID     int64 `gorm:"primaryKey"`
}
