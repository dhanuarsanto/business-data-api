package config

import (
	"fmt"
	"log"
	"net/netip"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv               string `env:"APP_ENV" envDefault:"development"`
	GlobalLocalOnly      bool   `env:"GLOBAL_LOCAL_ONLY" envDefault:"true"`
	PostgresWriteEnabled bool   `env:"POSTGRES_WRITE_ENABLED" envDefault:"false"`
	Port                 int    `env:"PORT" envDefault:"8080"`
	AllowedOrigins       string `env:"ALLOWED_ORIGINS" envDefault:"http://localhost:3000,http://localhost:5173"`
	TrustedProxies       string `env:"TRUSTED_PROXIES" envDefault:""`
	AllowDirectClients   bool   `env:"ALLOW_DIRECT_CLIENTS" envDefault:"false"`
	MaxBodyBytes         int64  `env:"MAX_BODY_BYTES" envDefault:"1048576"`
	CookieSecure         bool   `env:"COOKIE_SECURE" envDefault:"true"`
	APIKeysPath          string `env:"API_KEYS_PATH" envDefault:"api_keys.json"`

	PostgresMaxtopURL string `env:"POSTGRES_MAXTOP_URL,required"`
	MSSQLMaxtopURL    string `env:"MSSQL_MAXTOP_URL,required"`

	PostgresPandoraURL string `env:"POSTGRES_PANDORA_URL,required"`
	MSSQLPandoraURL    string `env:"MSSQL_PANDORA_URL,required"`

	PostgresToplinkURL string `env:"POSTGRES_TOPLINK_URL,required"`
	MSSQLToplinkURL    string `env:"MSSQL_TOPLINK_URL,required"`

	PostgresMaxConns          int           `env:"POSTGRES_MAX_CONNS" envDefault:"10"`
	PostgresMinConns          int           `env:"POSTGRES_MIN_CONNS" envDefault:"0"`
	PostgresMaxConnIdleTime   time.Duration `env:"POSTGRES_MAX_CONN_IDLE_TIME" envDefault:"5m"`
	PostgresMaxConnLifetime   time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME" envDefault:"30m"`
	PostgresHealthCheckPeriod time.Duration `env:"POSTGRES_HEALTH_CHECK_PERIOD" envDefault:"1m"`

	MSSQLMaxOpenConns    int           `env:"MSSQL_MAX_OPEN_CONNS" envDefault:"10"`
	MSSQLMaxIdleConns    int           `env:"MSSQL_MAX_IDLE_CONNS" envDefault:"5"`
	MSSQLMaxConnLifetime time.Duration `env:"MSSQL_MAX_CONN_LIFETIME" envDefault:"30m"`

	RateLimitGlobalRate      float64       `env:"RATE_LIMIT_GLOBAL_RATE" envDefault:"50"`
	RateLimitGlobalCapacity  int           `env:"RATE_LIMIT_GLOBAL_CAPACITY" envDefault:"100"`
	RateLimitLoginRate       float64       `env:"RATE_LIMIT_LOGIN_RATE" envDefault:"0.2"`
	RateLimitLoginCapacity   int           `env:"RATE_LIMIT_LOGIN_CAPACITY" envDefault:"5"`
	RateLimitCleanupInterval time.Duration `env:"RATE_LIMIT_CLEANUP_INTERVAL" envDefault:"3m"`

	ServerReadHeaderTimeout time.Duration `env:"SERVER_READ_HEADER_TIMEOUT" envDefault:"10s"`
	ServerReadTimeout       time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	ServerWriteTimeout      time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"60s"`
	ServerIdleTimeout       time.Duration `env:"SERVER_IDLE_TIMEOUT" envDefault:"120s"`
	ServerShutdownTimeout   time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" envDefault:"10s"`

	CORSMaxAge int `env:"CORS_MAX_AGE" envDefault:"300"`

	JWTSecret        string        `env:"JWT_SECRET,required"`
	JWTIssuer        string        `env:"JWT_ISSUER" envDefault:"business-data-api"`
	JWTTokenDuration time.Duration `env:"JWT_TOKEN_DURATION" envDefault:"24h"`
}

func (c *Config) GetAllowedOrigins() []string {
	var cleaned []string
	for o := range strings.SplitSeq(c.AllowedOrigins, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return []string{"http://localhost:3000", "http://localhost:5173"}
	}
	return cleaned
}

func (c *Config) ParseTrustedProxies() ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for raw := range strings.SplitSeq(c.TrustedProxies, ",") {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			p, err := netip.ParsePrefix(part)
			if err != nil {
				return nil, fmt.Errorf("CIDR trusted proxy tidak valid %q: %w", part, err)
			}
			prefixes = append(prefixes, p)
			continue
		}
		addr, err := netip.ParseAddr(part)
		if err != nil {
			return nil, fmt.Errorf("IP trusted proxy tidak valid %q: %w", part, err)
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return prefixes, nil
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Gagal mem-parsing konfigurasi: %v", err)
	}

	return &cfg
}
