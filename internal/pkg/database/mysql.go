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
	var indexCount int64
	db.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE table_schema=DATABASE() AND table_name='articles' AND index_name='idx_fulltext_title_content'").Scan(&indexCount)
	if indexCount == 0 {
		db.Exec("ALTER TABLE articles ADD FULLTEXT INDEX idx_fulltext_title_content (title, content)")
	}
	return db
}
