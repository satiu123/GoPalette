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
	return db
}
