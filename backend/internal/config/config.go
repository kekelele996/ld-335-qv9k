package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config 集中解析环境变量：数据库、JWT、API Key、限流、端口。
type Config struct {
	Port          string `env:"PORT" envDefault:"8080"`
	DBHost        string `env:"DB_HOST" envDefault:"localhost"`
	DBPort        string `env:"DB_PORT" envDefault:"5432"`
	DBName        string `env:"DB_NAME" envDefault:"gbinsureapi_db"`
	DBUser        string `env:"DB_USER" envDefault:"gbinsureapi_user"`
	DBPassword    string `env:"DB_PASSWORD" envDefault:"gbinsureapi_pwd"`
	DBSSLMode     string `env:"DB_SSLMODE" envDefault:"disable"`
	JWTSecret     string `env:"JWT_SECRET" envDefault:"change_me_to_a_long_random_string"`
	JWTExpireHours int  `env:"JWT_EXPIRE_HOURS" envDefault:"24"`
	APIKeySecret  string `env:"API_KEY_SECRET" envDefault:"change_me_to_a_long_random_string"`
	DefaultQPS    int    `env:"DEFAULT_QPS" envDefault:"10"`
	AdminUsername string `env:"ADMIN_USERNAME" envDefault:"admin"`
	AdminPassword string `env:"ADMIN_PASSWORD" envDefault:"admin123"`
	CORSOrigins   string `env:"APP_CORS_ORIGINS" envDefault:"http://localhost:19935"`
}

// Load 解析环境变量。
func Load() (Config, error) { return env.ParseAs[Config]() }

// DSN 构造 PostgreSQL 连接串。
func (c Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

// CORSOriginsList 解析逗号分隔的 CORS 来源白名单，生产默认不允许通配符。
func (c Config) CORSOriginsList() []string {
	raw := strings.TrimSpace(c.CORSOrigins)
	if raw == "" || raw == "*" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
