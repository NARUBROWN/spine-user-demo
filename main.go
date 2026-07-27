package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"spine-user-demo/controller"
	"spine-user-demo/interceptor"
	"spine-user-demo/repository"
	"spine-user-demo/routes"
	"spine-user-demo/service"
	"time"

	"github.com/NARUBROWN/spine"
	"github.com/NARUBROWN/spine/interceptor/cors"
	"github.com/NARUBROWN/spine/pkg/boot"
	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"

	_ "spine-user-demo/docs"

	"github.com/uptrace/bun/migrate"
)

//go:embed migrations/*.sql
var sqlMigrations embed.FS

// @title Spine User Demo API
// @version 0.2.0
// @description Spine + Swaggo
// @host localhost:8080
// @BasePath /
func main() {
	// CLI 플래그로 마이그레이션만 실행하는 옵션 추가
	migrateOnly := flag.Bool("migrate", false, "Run database migrations and exit")
	flag.Parse()

	// 마이그레이션 실행
	if *migrateOnly {
		fmt.Println("Running migrations...")
		db, err := openBunDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Database configuration failed: %v\n", err)
			os.Exit(1)
		}
		if err := runMigrations(context.Background(), db); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migrations applied successfully.")
		return
	}

	app := newApplication()
	options := applicationOptions()
	if err := errors.Join(app.Validate(options), validateEnvironment()); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(options); err != nil {
		fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}

func validateEnvironment() error {
	dsn := os.Getenv("SPINE_DEMO_MYSQL_DSN")
	if dsn == "" {
		return boot.ValidationError(boot.ConfigIssue{
			Path:    "Environment.SPINE_DEMO_MYSQL_DSN",
			Code:    "REQUIRED",
			Message: "MySQL DSN is required",
			Hint:    "Set SPINE_DEMO_MYSQL_DSN before starting the application",
		})
	}
	if _, err := mysql.ParseDSN(dsn); err != nil {
		return boot.ValidationError(boot.ConfigIssue{
			Path:    "Environment.SPINE_DEMO_MYSQL_DSN",
			Code:    "INVALID_DSN",
			Message: "MySQL DSN is invalid",
			Hint:    "Use a go-sql-driver/mysql compatible DSN",
		})
	}
	return nil
}

func newApplication() spine.App {
	app := spine.New()

	// 생성자 등록
	app.Constructor(
		newBunDB,
		repository.NewUserRepository,
		service.NewUserService,
		controller.NewUserController,
		interceptor.NewTxInterceptor,
	)

	// 인터셉터 등록
	app.InterceptorFor(
		boot.InterceptorHTTP,
		cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Content-Type"},
		}),
		/*
		 Constructor 등록 + WarmUp 단계에서 이미 생성된 TxInterceptor 인스턴스를
		 실행 파이프라인에서 사용하도록 타입으로 참조합니다.
		*/
		(*interceptor.TxInterceptor)(nil),
		&interceptor.LoggingInterceptor{},
	)

	// 유저 라우트 등록
	routes.RegisterUserRoutes(app)

	// 스웨거 UI 등록
	app.Transport(func(t any) {
		e := t.(*echo.Echo)
		e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))
	})
	return app
}

func applicationOptions() boot.Options {
	address := os.Getenv("SPINE_DEMO_ADDRESS")
	if address == "" {
		address = ":8080"
	}
	return boot.Options{
		Address:                address,
		EnableGracefulShutdown: true,
		ShutdownTimeout:        10 * time.Second,
		HTTP: &boot.HTTPOptions{
			DisableRecover: true,
		},
	}
}

func newBunDB() *bun.DB {
	db, err := openBunDB()
	if err != nil {
		panic(err)
	}
	return db
}

func openBunDB() (*bun.DB, error) {
	dsn := os.Getenv("SPINE_DEMO_MYSQL_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("SPINE_DEMO_MYSQL_DSN is required")
	}

	sqldb, err := sql.Open(
		"mysql",
		dsn,
	)
	if err != nil {
		return nil, err
	}
	db := bun.NewDB(sqldb, mysqldialect.New())
	return db, nil
}

func runMigrations(ctx context.Context, db *bun.DB) error {
	migs := migrate.NewMigrations()
	if err := migs.Discover(sqlMigrations); err != nil {
		return err
	}

	m := migrate.NewMigrator(db, migs)

	if err := m.Init(ctx); err != nil {
		return err
	}

	if _, err := m.Migrate(ctx); err != nil {
		return err
	}
	return nil
}
