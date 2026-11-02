package route

import (
	"os"

	"spine-user-demo/src/controller"

	"github.com/NARUBROWN/spine"
)

// Register는 HTTP 라우트를 등록합니다.
func Register(app spine.App) error {
	app.Route("DELETE", "/users", (*controller.UserController).DeleteUser)
	app.Route("PUT", "/users", (*controller.UserController).UpdateUser)
	app.Route("POST", "/users", (*controller.UserController).CreateUser)
	app.Route("GET", "/users", (*controller.UserController).GetUser)
	if os.Getenv("SPINE_DEMO_ENABLE_FAILURE_SCENARIOS") == "1" {
		app.Route("POST", "/scenarios/transactions/controller-error", (*controller.UserController).CreateThenFail)
		app.Route("POST", "/scenarios/transactions/response-error", (*controller.UserController).CreateWithInvalidCookie)
	}
	return nil
}
