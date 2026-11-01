package user

import (
	"os"

	"github.com/NARUBROWN/spine"
)

// Register는 사용자 CRUD의 생성자와 HTTP 라우트를 등록합니다.
func Register(app spine.App) error {
	app.Constructor(NewUserRepository, NewUserService, NewUserController)
	app.Route("GET", "/users", (*UserController).GetUser)
	app.Route("POST", "/users", (*UserController).CreateUser)
	app.Route("PUT", "/users", (*UserController).UpdateUser)
	app.Route("DELETE", "/users", (*UserController).DeleteUser)

	if os.Getenv("SPINE_DEMO_ENABLE_FAILURE_SCENARIOS") == "1" {
		app.Route("POST", "/scenarios/transactions/controller-error", (*UserController).CreateThenFail)
		app.Route("POST", "/scenarios/transactions/response-error", (*UserController).CreateWithInvalidCookie)
	}
	return nil
}
