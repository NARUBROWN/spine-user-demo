package application

import (
	"os"
	"time"

	"spine-user-demo/controller"
	"spine-user-demo/interceptor"
	"spine-user-demo/internal/database"
	"spine-user-demo/repository"
	"spine-user-demo/routes"
	"spine-user-demo/service"

	"github.com/NARUBROWN/spine"
	"github.com/NARUBROWN/spine/interceptor/cors"
	"github.com/NARUBROWN/spine/pkg/boot"
	"github.com/labstack/echo/v4"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "spine-user-demo/docs"
)

// BuildApplication은 서버와 CLI가 공유하는 구성 진입점이며 DB 연결이나 리스너를 시작하지 않습니다.
func BuildApplication() (spine.App, boot.Options, error) {
	app := spine.New()
	app.Constructor(
		database.NewBunDB,
		repository.NewUserRepository,
		service.NewUserService,
		controller.NewUserController,
		interceptor.NewTxInterceptor,
	)
	app.InterceptorFor(
		boot.InterceptorHTTP,
		cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Content-Type"},
		}),
		(*interceptor.TxInterceptor)(nil),
		&interceptor.LoggingInterceptor{},
	)
	if err := routes.Register(app); err != nil {
		return nil, boot.Options{}, err
	}
	app.Transport(func(transport any) {
		e := transport.(*echo.Echo)
		e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))
	})

	address := os.Getenv("SPINE_DEMO_ADDRESS")
	if address == "" {
		port := os.Getenv("SPINE_PORT")
		if port == "" {
			port = "8080"
		}
		address = ":" + port
	}
	return app, boot.Options{
		Address:                address,
		EnableGracefulShutdown: true,
		ShutdownTimeout:        10 * time.Second,
		HTTP:                   &boot.HTTPOptions{DisableRecover: true},
	}, nil
}
