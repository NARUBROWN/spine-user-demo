package main

import (
	"spine-user-demo/controller"
	"spine-user-demo/interceptor"
	"spine-user-demo/repository"
	"spine-user-demo/routes"
	"spine-user-demo/service"

	"github.com/NARUBROWN/spine"
	"github.com/NARUBROWN/spine/interceptor/cors"
)

func main() {
	app := spine.New()

	// 생성자 등록
	app.Constructor(
		repository.NewUserRepository,
		service.NewUserService,
		controller.NewUserController,
	)

	// 인터셉터 등록
	app.Interceptor(
		cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "OPTIONS"},
			AllowHeaders: []string{"Content-Type"},
		}),
		&interceptor.LoggingInterceptor{},
	)

	// 유저 라우트 등록
	routes.RegisterUserRoutes(app)

	app.Run(":8080")
}
