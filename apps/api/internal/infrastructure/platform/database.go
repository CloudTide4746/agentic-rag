package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// openDatabase 解析 MySQL DSN，配置超时、UTC 时间及连接池，并验证数据库连接。
func openDatabase(ctx context.Context, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("DATABASE_DSN must be set")
	}
	driverConfig, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database DSN: %w", err)
	}
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	driverConfig.Timeout = 5 * time.Second
	driverConfig.ReadTimeout = 5 * time.Second
	driverConfig.WriteTimeout = 5 * time.Second
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{DSN: driverConfig.FormatDSN()}), &gorm.Config{
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database pool: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
