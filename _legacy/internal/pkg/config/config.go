package config

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// 全局配置变量
var GlobalConfig *Config

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	CORS       CORSConfig       `mapstructure:"cors"`
	Attachment AttachmentConfig `mapstructure:"attachment"`
}

func LoadConfig() {
	viper.SetDefault("server.addr", ":8080")
	viper.SetDefault("server.env", "development")
	viper.SetDefault("jwt.access_token_ttl", "15m")
	viper.SetDefault("jwt.refresh_token_ttl", "168h")
	viper.SetDefault("database.dsn", "")
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("cors.allow_origins", []string{"http://localhost:3000", "http://localhost:5173"})
	viper.SetDefault("attachment.cleanup_spec", "@every 1h")
	viper.SetDefault("attachment.temp_ttl_hours", 24)

	// 设置配置文件路径和名称
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("GP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			slog.Warn("未找到配置文件，使用默认值和环境变量")
		} else {
			slog.Error("读取配置文件失败", "error", err)
		}
	}

	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		slog.Error("解析配置文件失败", "error", err)
	}

	// 兼容无前缀环境变量，便于容器编排和本地调试。
	applyLegacyEnvOverrides(GlobalConfig)

	if GlobalConfig.Attachment.CleanupSpec == "" {
		GlobalConfig.Attachment.CleanupSpec = "@every 1h"
	}
	if GlobalConfig.Attachment.TempTTLHours <= 0 {
		GlobalConfig.Attachment.TempTTLHours = 24
	}

	slog.Info("配置文件加载成功")

}

func applyLegacyEnvOverrides(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("SERVER_ADDR")); v != "" {
		cfg.Server.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv("SERVER_ENV")); v != "" {
		cfg.Server.Env = v
	}

	if v := strings.TrimSpace(os.Getenv("JWT_ACCESS_TOKEN_SECRET")); v != "" {
		cfg.JWT.AccessTokenSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("JWT_REFRESH_TOKEN_SECRET")); v != "" {
		cfg.JWT.RefreshTokenSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("JWT_ACCESS_TOKEN_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JWT.AccessTokenTTL = d
		}
	}
	if v := strings.TrimSpace(os.Getenv("JWT_REFRESH_TOKEN_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JWT.RefreshTokenTTL = d
		}
	}

	if v := strings.TrimSpace(os.Getenv("DATABASE_DSN")); v != "" {
		cfg.Database.DSN = v
	}

	if v := strings.TrimSpace(os.Getenv("REDIS_ADDR")); v != "" {
		cfg.Redis.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv("REDIS_PASSWORD")); v != "" {
		cfg.Redis.Password = v
	}
}

type ServerConfig struct {
	Addr string `mapstructure:"addr"`
	Env  string `mapstructure:"env"` // 环境变量，development 或 production
}
type JWTConfig struct {
	AccessTokenSecret  string        `mapstructure:"access_token_secret"`
	RefreshTokenSecret string        `mapstructure:"refresh_token_secret"`
	AccessTokenTTL     time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL    time.Duration `mapstructure:"refresh_token_ttl"`
}
type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

type AttachmentConfig struct {
	CleanupSpec  string `mapstructure:"cleanup_spec"`
	TempTTLHours int    `mapstructure:"temp_ttl_hours"`
}
