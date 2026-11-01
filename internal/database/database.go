package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/NARUBROWN/spine/pkg/boot"
	"github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
)

func ValidateEnvironment() error {
	dsn := os.Getenv("SPINE_DEMO_MYSQL_DSN")
	if dsn == "" {
		return boot.ValidationError(boot.ConfigIssue{
			Path: "Environment.SPINE_DEMO_MYSQL_DSN", Code: "REQUIRED",
			Message: "MySQL DSN is required", Hint: "Set SPINE_DEMO_MYSQL_DSN before starting the application",
		})
	}
	if _, err := mysql.ParseDSN(dsn); err != nil {
		return boot.ValidationError(boot.ConfigIssue{
			Path: "Environment.SPINE_DEMO_MYSQL_DSN", Code: "INVALID_DSN",
			Message: "MySQL DSN is invalid", Hint: "Use a go-sql-driver/mysql compatible DSN",
		})
	}
	return nil
}

// NewBunDB는 앱 실행 시 DI 컨테이너가 호출하는 지연 생성자입니다.
func NewBunDB() *bun.DB {
	db, err := Open()
	if err != nil {
		panic(err)
	}
	return db
}

func Open() (*bun.DB, error) {
	dsn := os.Getenv("SPINE_DEMO_MYSQL_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("SPINE_DEMO_MYSQL_DSN is required")
	}
	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return bun.NewDB(sqldb, mysqldialect.New()), nil
}
