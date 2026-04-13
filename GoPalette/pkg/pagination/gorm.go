package pagination

import "gorm.io/gorm"

func Paginate(p *PagingParam) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(p.GetOffset()).Limit(int(p.GetLimit()))
	}
}
