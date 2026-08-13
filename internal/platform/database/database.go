package database

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/config"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(ctx context.Context, cfg config.Config) (*gorm.DB, func() error, error) {
	query := url.Values{}
	query.Set("database", cfg.Database)
	query.Set("encrypt", "false")
	query.Set("TrustServerCertificate", strconv.FormatBool(true))
	query.Set("app name", "syncline-collection-tracking-migrate")
	dsn := (&url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(cfg.Username, cfg.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		RawQuery: query.Encode(),
	}).String()

	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("open SQL Server connection: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get SQL connection: %w", err)
	}
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetMaxOpenConns(5)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("ping SQL Server: %w", err)
	}
	return db.WithContext(ctx), sqlDB.Close, nil
}
