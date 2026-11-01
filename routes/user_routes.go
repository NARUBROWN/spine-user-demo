package routes

import (
	"os"
	"spine-user-demo/controller"

	"github.com/NARUBROWN/spine"
)

func Register(app spine.App) error {
	app.Route("GET", "/users", (*controller.UserController).GetUser)
	app.Route("POST", "/users", (*controller.UserController).CreateUser)
	app.Route("PUT", "/users", (*controller.UserController).UpdateUser)
	app.Route("DELETE", "/users", (*controller.UserController).DeleteUser)

	if os.Getenv("SPINE_DEMO_ENABLE_FAILURE_SCENARIOS") == "1" {
		app.Route("POST", "/scenarios/transactions/controller-error", (*controller.UserController).CreateThenFail)
		app.Route("POST", "/scenarios/transactions/response-error", (*controller.UserController).CreateWithInvalidCookie)
	}
	return nil
}
