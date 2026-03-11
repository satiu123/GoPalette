package database

import (
	"github.com/satiu123/GoPalette/internal/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitMySQL(models ...any) *gorm.DB {
	db, err := gorm.Open(mysql.Open(config.GlobalConfig.Database.DSN), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			panic("failed to auto migrate: " + err.Error())
		}
	}
	// 为文章表的 title+content 创建 FULLTEXT 索引（体现倒排索引原理）
	// HasIndex 返回 false 时才执行，保证幂等
	if !db.Migrator().HasIndex(&struct{ Title, Content string }{}, "idx_fulltext_title_content") {
		db.Exec("ALTER TABLE articles ADD FULLTEXT INDEX idx_fulltext_title_content (title, content)")
	}
	return db
}

