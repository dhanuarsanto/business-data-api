package config

import (
	"log"
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
	origins := strings.Split(c.AllowedOrigins, ",")
	var cleaned []string
	for _, o := range origins {
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

func LoadConfig() *Config {
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Gagal mem-parsing konfigurasi: %v", err)
	}

	return &cfg
}
