package main

import (
	"spine-user-demo/controller"
	"spine-user-demo/repository"
	"spine-user-demo/routes"
	"spine-user-demo/service"

	"github.com/NARUBROWN/spine"
)

func main() {
	app := spine.New()

	// 생성자 등록
	app.Constructor(
		repository.NewUserRepository,
		service.NewUserService,
		controller.NewUserController,
	)

	// 유저 라우트 등록
	routes.RegisterUserRoutes(app)

	app.Run(":8080")
}
