package application

import (
	"os"
	"time"

	"spine-user-demo/internal/database"
	"spine-user-demo/internal/interceptor"
	"spine-user-demo/src/controller"
	"spine-user-demo/src/repository"
	"spine-user-demo/src/service"

	"github.com/NARUBROWN/spine"
	echoplatform "github.com/NARUBROWN/spine-echo-platform"
	"github.com/NARUBROWN/spine/interceptor/cors"
	"github.com/NARUBROWN/spine/pkg/boot"
	"github.com/labstack/echo/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "spine-user-demo/docs"
)

// BuildApplication은 서버와 CLI가 공유하는 구성 진입점이며 DB 연결이나 리스너를 시작하지 않습니다.
func BuildApplication() (spine.App, boot.Options, error) {
	if err := controller.ValidateEnvironment(); err != nil {
		return nil, boot.Options{}, err
	}
	app := spine.New(echoplatform.Option(echoplatform.WithEcho(func(e *echo.Echo) {
		e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))
	})))
	app.Constructor(database.NewBunDB)
	app.RepositoryConstructor(&repository.UserRepository{})
	app.ServiceConstructor(&service.UserService{})
	app.ControllerConstructor(&controller.UserController{})
	app.InterceptorConstructor(&interceptor.TxInterceptor{})
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
		HTTP:                   &boot.HTTPOptions{},
	}, nil
}
