// Package config loads and validates configuration before opening infrastructure.
package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/goccy/go-yaml"
)

type Config struct {
	EnvFile string `yaml:"env_file"`
	Server  struct {
		Addr    string `yaml:"addr"`
		GinMode string `yaml:"gin_mode"`
	} `yaml:"server"`
	Database struct {
		DSN  string `yaml:"dsn"`
		Name string `yaml:"name"`
	} `yaml:"database"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`
	Agent struct {
		BaseURL string `yaml:"base_url"`
		Token   string `yaml:"internal_api_token"`
	} `yaml:"agent"`
}

// Load 按进程环境变量、YAML、可选 .env 的优先级加载并校验配置。
// Load uses process environment > YAML > optional .env, without setting globals.
// An explicit path must exist; a missing default config.yaml enables env-only
// startup. env_file is relative to the YAML file, not the working directory.
func Load(path string) (Config, error) {
	explicit := path != ""
	if !explicit {
		path = "config.yaml"
	}
	data, err := readFile(path)
	if err != nil && !(errors.Is(err, os.ErrNotExist) && !explicit) {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if len(data) > 0 {
		if err := decodeYAML(data, &cfg); err != nil {
			return Config{}, err
		}
	}
	values := map[string]string{}
	if cfg.EnvFile != "" {
		envPath := cfg.EnvFile
		if !filepath.IsAbs(envPath) {
			envPath = filepath.Join(filepath.Dir(path), envPath)
		}
		values, err = readEnv(envPath)
		if err != nil {
			return Config{}, err
		}
	}
	// Seed from .env and decode YAML over it, then apply process overrides.
	cfg = Config{}
	cfg.Server.GinMode = "debug"
	if err := applyEnv(&cfg, func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}); err != nil {
		return Config{}, err
	}
	if len(data) > 0 {
		if err := decodeYAML(data, &cfg); err != nil {
			return Config{}, err
		}
	}
	if err := applyEnv(&cfg, os.LookupEnv); err != nil {
		return Config{}, err
	}
	// An explicit process DSN selects its own database, including in production.
	if _, overridden := os.LookupEnv("DATABASE_DSN"); overridden {
		cfg.Database.Name = ""
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// decodeYAML 严格解析单个 YAML 配置文档，并避免在解析错误中泄漏配置内容。
func decodeYAML(data []byte, cfg *Config) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data), yaml.Strict())
	if err := decoder.Decode(cfg); err != nil {
		// Parser errors can include source lines containing secrets.
		return errors.New("invalid config YAML: check syntax, field names and value types")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("config YAML must contain exactly one document")
	}
	return nil
}

// applyEnv 通过给定的查询函数读取环境变量并覆盖对应配置项。
func applyEnv(cfg *Config, lookup func(string) (string, bool)) error {
	for _, binding := range []struct {
		key   string
		value *string
	}{
		{"API_ADDR", &cfg.Server.Addr}, {"GIN_MODE", &cfg.Server.GinMode},
		{"DATABASE_DSN", &cfg.Database.DSN}, {"REDIS_ADDR", &cfg.Redis.Addr},
		{"REDIS_PASSWORD", &cfg.Redis.Password}, {"JWT_SECRET", &cfg.JWT.Secret},
		{"AGENT_BASE_URL", &cfg.Agent.BaseURL}, {"INTERNAL_API_TOKEN", &cfg.Agent.Token},
	} {
		if value, ok := lookup(binding.key); ok {
			*binding.value = value
		}
	}
	if value, ok := lookup("REDIS_DB"); ok && value != "" {
		db, err := strconv.Atoi(value)
		if err != nil {
			return errors.New("REDIS_DB must be a non-negative integer")
		}
		cfg.Redis.DB = db
	}
	return nil
}

// validate 规范化并校验服务地址、数据库、Redis、JWT 和 Agent 配置。
func (cfg *Config) validate() error {
	cfg.Server.Addr = strings.TrimSpace(cfg.Server.Addr)
	if !validAddress(cfg.Server.Addr, true) {
		return errors.New("server.addr / API_ADDR must be host:port or :port (1-65535)")
	}
	switch cfg.Server.GinMode {
	case "debug", "release", "test":
	default:
		return errors.New("server.gin_mode / GIN_MODE must be debug, release or test")
	}
	if cfg.Database.DSN == "" {
		return errors.New("database.dsn / DATABASE_DSN must be set")
	}
	db, err := mysql.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return errors.New("database.dsn / DATABASE_DSN is invalid")
	}
	if cfg.Database.Name != "" {
		db.DBName = cfg.Database.Name
	}
	if db.DBName == "" {
		return errors.New("database name must be set")
	}
	cfg.Database.DSN = db.FormatDSN()
	cfg.Redis.Addr = strings.TrimSpace(cfg.Redis.Addr)
	if !validAddress(cfg.Redis.Addr, false) {
		return errors.New("redis.addr / REDIS_ADDR must be host:port (1-65535)")
	}
	if cfg.Redis.DB < 0 {
		return errors.New("redis.db / REDIS_DB must be a non-negative integer")
	}
	cfg.JWT.Secret = strings.TrimSpace(cfg.JWT.Secret)
	if len(cfg.JWT.Secret) < 32 {
		return errors.New("jwt.secret / JWT_SECRET must contain at least 32 characters")
	}
	cfg.Agent.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.Agent.BaseURL), "/")
	cfg.Agent.Token = strings.TrimSpace(cfg.Agent.Token)
	if cfg.Agent.BaseURL != "" {
		endpoint, err := url.Parse(cfg.Agent.BaseURL)
		if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
			return errors.New("agent.base_url / AGENT_BASE_URL must be an HTTP(S) URL without credentials, query or fragment")
		}
	}
	return nil
}

// validAddress 检查地址是否包含有效端口，并按参数决定是否允许空主机名。
func validAddress(address string, allowEmptyHost bool) bool {
	host, port, err := net.SplitHostPort(address)
	if err != nil || (!allowEmptyHost && host == "") {
		return false
	}
	number, err := strconv.Atoi(port)
	return err == nil && number >= 1 && number <= 65535
}

// readFile 读取不超过 1 MiB 的配置文件并移除 UTF-8 BOM。
func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	const limit = 1 << 20
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, errors.New("configuration file exceeds 1 MiB")
	}
	return bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), nil
}

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// readEnv 解析可选 .env 文件中的键值配置，文件不存在时返回空映射。
func readEnv(path string) (map[string]string, error) {
	values := map[string]string{}
	data, err := readFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return values, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read env_file: %w", err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		key, value, ok := strings.Cut(text, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !ok || !envKey.MatchString(key) {
			return nil, fmt.Errorf("env_file line %d: expected KEY=VALUE", line)
		}
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.New("could not read env_file")
	}
	return values, nil
}
