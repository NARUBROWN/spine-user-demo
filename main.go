package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"os"
	"spine-user-demo/controller"
	"spine-user-demo/interceptor"
	"spine-user-demo/repository"
	"spine-user-demo/routes"
	"spine-user-demo/service"

	"github.com/NARUBROWN/spine"
	"github.com/NARUBROWN/spine/interceptor/cors"
	_ "github.com/go-sql-driver/mysql"
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
		db := newBunDB()
		if err := runMigrations(context.Background(), db); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migrations applied successfully.")
		return
	}

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
	app.Interceptor(
		/*
		 Constructor 등록 + WarmUp 단계에서 이미 생성된 TxInterceptor 인스턴스를
		 실행 파이프라인에서 사용하도록 타입으로 참조합니다.
		*/
		(*interceptor.TxInterceptor)(nil),
		cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "OPTIONS"},
			AllowHeaders: []string{"Content-Type"},
		}),
		&interceptor.LoggingInterceptor{},
	)

	// 유저 라우트 등록
	routes.RegisterUserRoutes(app)

	// 스웨거 UI 등록
	app.Transport(func(t any) {
		e := t.(*echo.Echo)
		e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))
	})

	app.Run(":8080")
}

func newBunDB() *bun.DB {
	sqldb, err := sql.Open(
		"mysql",
		"ID:PASSWORD@tcp(localhost:3306)/spine_demo?parseTime=true&loc=Local",
	)
	if err != nil {
		panic(err)
	}
	db := bun.NewDB(sqldb, mysqldialect.New())
	return db
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
