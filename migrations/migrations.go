package migrations

import (
	"context"
	"embed"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

//go:embed *.sql
var sqlMigrations embed.FS

// Run은 명시적인 -migrate 실행에서만 데이터베이스 스키마를 변경합니다.
func Run(ctx context.Context, db *bun.DB) error {
	migrations := migrate.NewMigrations()
	if err := migrations.Discover(sqlMigrations); err != nil {
		return err
	}
	migrator := migrate.NewMigrator(db, migrations)
	if err := migrator.Init(ctx); err != nil {
		return err
	}
	_, err := migrator.Migrate(ctx)
	return err
}
