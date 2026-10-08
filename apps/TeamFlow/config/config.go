package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	Upload   UploadConfig   `mapstructure:"upload"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Port    int    `mapstructure:"port"`
	Version string `mapstructure:"version"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	Charset         string `mapstructure:"charset"`
	ParseTime       bool   `mapstructure:"parseTime"`
	Loc             string `mapstructure:"loc"`
	MaxIdleConns    int    `mapstructure:"maxIdleConns"`
	MaxOpenConns    int    `mapstructure:"maxOpenConns"`
	ConnMaxLifetime int    `mapstructure:"connMaxLifetime"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"poolSize"`
	MinIdleConns int    `mapstructure:"minIdleConns"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expireHours"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"maxSize"`
	MaxBackups int    `mapstructure:"maxBackups"`
	MaxAge     int    `mapstructure:"maxAge"`
	Compress   bool   `mapstructure:"compress"`
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxSize    int      `mapstructure:"maxSize"`
	AllowTypes []string `mapstructure:"allowTypes"`
	SavePath   string   `mapstructure:"savePath"`
}

// GlobalConfig 全局配置实例
var GlobalConfig *Config

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("加载环境变量失败: %w", err)
	}
	if configPath == "" {

		env := os.Getenv("APP_ENV")
		if env == "" {
			env = "development"
		}

		configPath = fmt.Sprintf(
			"config/config.%s.yaml",
			env,
		)

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = "config/config.yaml"
		}
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	v.SetConfigType("yaml")

	v.SetEnvPrefix("TEAMFLOW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.AutomaticEnv()

	if err := v.BindEnv(
		"database.password",
		"DB_PASSWORD",
	); err != nil {
		return nil, fmt.Errorf("绑定数据库密码环境变量失败: %w", err)
	}

	if err := v.BindEnv(
		"jwt.secret",
		"JWT_SECRET",
	); err != nil {
		return nil, fmt.Errorf("绑定 JWT 环境变量失败: %w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf(
			"读取配置文件失败: %w",
			err,
		)
	}

	var config Config

	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf(
			"解析配置文件失败: %w",
			err,
		)
	}
	config.JWT.Secret = os.ExpandEnv(config.JWT.Secret)
	GlobalConfig = &config

	return &config, nil
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.Charset,
		c.ParseTime,
		c.Loc,
	)
}

// GetAddr 获取 Redis 连接地址
func (c *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetExpireDuration 获取 JWT 过期时间
func (c *JWTConfig) GetExpireDuration() time.Duration {
	return time.Duration(c.ExpireHours) * time.Hour
}

// IsDevelopment 是否开发环境
func (c *AppConfig) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction 是否生产环境
func (c *AppConfig) IsProduction() bool {
	return c.Env == "production"
}
