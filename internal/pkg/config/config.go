package config

import (
	"log/slog"
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
	// 设置配置文件路径和名称
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("读取配置文件失败", "error", err)
	}

	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		slog.Error("解析配置文件失败", "error", err)
	}
	if GlobalConfig.Attachment.CleanupSpec == "" {
		GlobalConfig.Attachment.CleanupSpec = "@every 1h"
	}
	if GlobalConfig.Attachment.TempTTLHours <= 0 {
		GlobalConfig.Attachment.TempTTLHours = 24
	}

	slog.Info("配置文件加载成功")

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
