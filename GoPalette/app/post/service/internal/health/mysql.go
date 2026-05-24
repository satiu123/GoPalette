package health

import (
	"context"

	"gorm.io/gorm"
)

type MySQLChecker struct {
	db *gorm.DB
}

func NewMySQLChecker(db *gorm.DB) *MySQLChecker {
	return &MySQLChecker{
		db: db,
	}
}

func (c *MySQLChecker) Name() string {
	return "mysql"
}

func (c *MySQLChecker) Check(ctx context.Context) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.PingContext(ctx)
}
