package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string `env:"HOST" envDefault:"localhost"`
	Port     int    `env:"PORT" envDefault:"1433"`
	Username string `env:"USER" envDefault:"dev"`
	Password string `env:"PASSWORD" envDefault:"trustno1"`
	Database string `env:"DATABASE" envDefault:"CTS_DEV"`
}

// BuildDSN func
func (c *Config) buildDSN() string {
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// ConnectDB func
func (c *Config) ConnectDB() *gorm.DB {
	dsn := c.buildDSN()

	dbConn, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// Plugins: dbplugin.DBPlugin,
	})
	if err != nil {
		panic(`fatal error: cannot connect to database`)
	}

	return dbConn
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse migration configuration: %w", err)
	}
	return cfg, nil
}

func Read() *Config {
	conf := Config{}
	ctx := context.Background()

	if err := envconfig.Process(ctx, &conf); err != nil {
		log.Fatal(err)
	}

	return &conf
}
