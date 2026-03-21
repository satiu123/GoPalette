package database

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/satiu123/GoPalette/internal/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type articleSlugRow struct {
	ID    int64
	Title string
	Slug  string
}

func sanitizeSlug(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}

func normalizeArticleSlugs(db *gorm.DB) error {
	var rows []articleSlugRow
	if err := db.Table("articles").Select("id", "title", "slug").Order("id ASC").Scan(&rows).Error; err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		slug := sanitizeSlug(row.Slug)
		if slug == "" {
			slug = sanitizeSlug(row.Title)
		}
		if slug == "" {
			slug = fmt.Sprintf("article-%d", row.ID)
		}

		candidate := slug
		if _, exists := seen[candidate]; exists {
			candidate = fmt.Sprintf("%s-%d", slug, row.ID)
			i := 2
			for {
				if _, dup := seen[candidate]; !dup {
					break
				}
				candidate = fmt.Sprintf("%s-%d-%d", slug, row.ID, i)
				i++
			}
		}

		seen[candidate] = struct{}{}
		if candidate == row.Slug {
			continue
		}
		if err := db.Table("articles").Where("id = ?", row.ID).Update("slug", candidate).Error; err != nil {
			return err
		}
	}

	return nil
}

func ensureArticleSlugUniqueIndex(db *gorm.DB) error {
	if err := normalizeArticleSlugs(db); err != nil {
		return err
	}

	var indexCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE table_schema=DATABASE() AND table_name='articles' AND index_name='idx_articles_slug'").Scan(&indexCount).Error; err != nil {
		return err
	}

	if indexCount > 0 {
		var nonUnique int64
		if err := db.Raw("SELECT NON_UNIQUE FROM information_schema.STATISTICS WHERE table_schema=DATABASE() AND table_name='articles' AND index_name='idx_articles_slug' LIMIT 1").Scan(&nonUnique).Error; err != nil {
			return err
		}
		if nonUnique == 1 {
			if err := db.Exec("DROP INDEX idx_articles_slug ON articles").Error; err != nil {
				return err
			}
			indexCount = 0
		}
	}

	if indexCount == 0 {
		if err := db.Exec("CREATE UNIQUE INDEX idx_articles_slug ON articles(slug)").Error; err != nil {
			return err
		}
	}

	return nil
}

func InitMySQL(models ...any) *gorm.DB {
	db, err := gorm.Open(mysql.Open(config.GlobalConfig.Database.DSN), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			panic("failed to auto migrate: " + err.Error())
		}
		if err := ensureArticleSlugUniqueIndex(db); err != nil {
			panic("failed to migrate article slug index: " + err.Error())
		}
	}
	var indexCount int64
	db.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE table_schema=DATABASE() AND table_name='articles' AND index_name='idx_fulltext_title_content'").Scan(&indexCount)
	if indexCount == 0 {
		db.Exec("ALTER TABLE articles ADD FULLTEXT INDEX idx_fulltext_title_content (title, content)")
	}
	return db
}
