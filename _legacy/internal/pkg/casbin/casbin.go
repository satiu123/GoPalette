package casbin

import (
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

var Enforcer *casbin.Enforcer

//go:embed rbac_model.conf
var m string

func InitializeCasbin(db *gorm.DB) error {
	// 创建一个适配器，使用 Gorm 连接到数据库
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return fmt.Errorf("无法创建 Casbin 适配器: %w", err)
	}

	// 创建一个 Casbin enforcer，加载模型和策略。
	m, err := model.NewModelFromString(m)
	if err != nil {
		return fmt.Errorf("无法加载 Casbin 模型: %w", err)
	}
	Enforcer, err = casbin.NewEnforcer(m, adapter)
	if err != nil {
		return fmt.Errorf("无法创建 Casbin enforcer: %w", err)
	}
	Enforcer.EnableAutoSave(true)

	// 加载策略到内存
	err = Enforcer.LoadPolicy()
	if err != nil {
		return fmt.Errorf("无法加载 Casbin 策略: %w", err)
	}

	if err = initPolicies(); err != nil {
		return err
	}

	slog.Info("Casbin 已成功初始化")
	return nil
}

// 初始化默认策略
func initPolicies() error {
	policies, err := Enforcer.GetPolicy()
	if err != nil {
		return fmt.Errorf("读取 Casbin 策略失败: %w", err)
	}
	if len(policies) == 0 {
		defaultPolicies := [][]string{
			{"user", "/api/articles", "POST"},
			{"user", "/api/articles/:id", "PUT"},
			{"user", "/api/articles/:id", "DELETE"},
			{"user", "/api/comments/:id", "DELETE"},
			{"user", "/api/users/me", "GET"},
			{"user", "/api/users/me", "PUT"},
			{"user", "/api/users/me/articles", "GET"},
			{"user", "/api/users/me/comments", "GET"},
			{"user", "/api/users/me/comments/received", "GET"},
			{"user", "/api/upload", "POST"},
			{"admin", "/api/categories", "POST"},
			{"admin", "/api/categories/:id", "DELETE"},
			{"admin", "/api/tags", "POST"},
			{"admin", "/api/tags/:id", "DELETE"},
			{"admin", "/api/admin/articles", "GET"},
			{"admin", "/api/admin/comments", "GET"},
		}

		added, addErr := Enforcer.AddPolicies(defaultPolicies)
		if addErr != nil {
			return fmt.Errorf("添加默认 Casbin 策略失败: %w", addErr)
		}
		if !added {
			slog.Warn("默认 Casbin 策略未被添加（可能已存在）")
		}

		groupings := [][]string{{"admin", "user"}}
		if _, grpErr := Enforcer.AddGroupingPolicies(groupings); grpErr != nil {
			return fmt.Errorf("添加 Casbin 角色继承失败: %w", grpErr)
		}

		slog.Info("已添加默认 Casbin 策略")
	} else {
		slog.Info("Casbin 已加载现有策略")
	}

	return nil
}
