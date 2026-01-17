package routes

import (
	"spine-user-demo/controller"

	"github.com/NARUBROWN/spine"
)

func RegisterUserRoutes(app spine.App) {
	app.Route("GET", "/users", (*controller.UserController).GetUser)
	app.Route("POST", "/users", (*controller.UserController).CreateUser)
	app.Route("PUT", "/users", (*controller.UserController).UpdateUser)
	app.Route("DELETE", "/users", (*controller.UserController).DeleteUser)

}
