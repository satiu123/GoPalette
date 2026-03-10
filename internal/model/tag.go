package model

type Tag struct {
	ID   int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name string `json:"name" gorm:"type:varchar(50);uniqueIndex;not null"`
}
