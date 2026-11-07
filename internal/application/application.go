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
	playground "github.com/NARUBROWN/spine-validator-playground"
	"github.com/NARUBROWN/spine/interceptor/cors"
	"github.com/NARUBROWN/spine/pkg/boot"
	"github.com/NARUBROWN/spine/pkg/constructor"
	"github.com/NARUBROWN/spine/pkg/openapi/ui/swaggerui"
)

// BuildApplication은 서버와 CLI가 공유하는 구성 진입점이며 DB 연결이나 리스너를 시작하지 않습니다.
func BuildApplication() (spine.App, boot.Options, error) {
	if err := controller.ValidateEnvironment(); err != nil {
		return nil, boot.Options{}, err
	}
	validator, err := playground.New()
	if err != nil {
		return nil, boot.Options{}, err
	}
	app := spine.New(echoplatform.Option(), spine.WithValidator(validator))
	app.Constructor(
		constructor.Custom(database.NewBunDB),
		constructor.Repository(&repository.UserRepository{}),
		constructor.Service(&service.UserService{}),
		constructor.Controller(&controller.UserController{}),
		constructor.Interceptor(&interceptor.TxInterceptor{}),
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
		HTTP: &boot.HTTPOptions{
			OpenAPI: boot.OpenAPIOptions{
				Enabled: true,
				Title:   "Spine User Demo API",
				Version: "0.2.0",
				UI:      []boot.OpenAPIUI{swaggerui.New(swaggerui.Options{Title: "Spine User Demo API"})},
			},
		},
	}, nil
}
