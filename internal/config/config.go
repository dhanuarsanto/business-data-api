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
	MaxBodyBytes         int64  `env:"MAX_BODY_BYTES" envDefault:"1048576"`
	CookieSecure         bool   `env:"COOKIE_SECURE" envDefault:"true"`

	PostgresMaxtopURL string `env:"POSTGRES_MAXTOP_URL,required"`
	MSSQLMaxtopURL    string `env:"MSSQL_MAXTOP_URL,required"`

	PostgresPandoraURL string `env:"POSTGRES_PANDORA_URL,required"`
	MSSQLPandoraURL    string `env:"MSSQL_PANDORA_URL,required"`

	PostgresToplinkURL string `env:"POSTGRES_TOPLINK_URL,required"`
	MSSQLToplinkURL    string `env:"MSSQL_TOPLINK_URL,required"`

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
