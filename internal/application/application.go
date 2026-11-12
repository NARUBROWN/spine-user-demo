package application

import (
	"os"
	"time"

	"spine-user-demo/internal/config"
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
	if err := config.ValidateEnvironment(); err != nil {
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
		constructor.Interceptor(&interceptor.LoggingInterceptor{}),
		constructor.Interceptor(&interceptor.SecurityTokenInterceptor{}),
		constructor.Custom(interceptor.NewCORSInterceptor),
	)
	app.InterceptorFor(boot.InterceptorHTTP,
		(*cors.CORSInterceptor)(nil),
		(*interceptor.TxInterceptor)(nil),
		(*interceptor.LoggingInterceptor)(nil),
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
				Path:    "/openapi.json",
				Title:   "Spine User Demo API",
				Version: "0.2.0",
				UI:      []boot.OpenAPIUI{swaggerui.New(swaggerui.Options{Title: "Spine User Demo API"})},
			},
		},
	}, nil
}
