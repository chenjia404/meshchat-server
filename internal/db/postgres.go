package db

import (
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// OpenPostgres initializes the primary PostgreSQL connection.
func OpenPostgres(dsn string, logger *slog.Logger) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: gormlogger.New(logWriter{logger: logger}, gormlogger.Config{
			SlowThreshold:             300 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		}),
	})
}

type logWriter struct {
	logger *slog.Logger
}

func (w logWriter) Printf(format string, args ...any) {
	w.logger.Debug("gorm", slog.String("format", format), slog.Any("args", args))
}
