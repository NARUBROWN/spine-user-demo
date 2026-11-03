package route

import (
	"fmt"
	"os"

	"spine-user-demo/internal/interceptor"
	"spine-user-demo/src/controller"

	"github.com/NARUBROWN/spine"
	spineroute "github.com/NARUBROWN/spine/pkg/route"
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
	if os.Getenv("SPINE_DEMO_ENABLE_SECURITY_SCENARIOS") == "1" {
		token := os.Getenv("SPINE_DEMO_SECURITY_TOKEN")
		if token == "" {
			return fmt.Errorf("SPINE_DEMO_SECURITY_TOKEN is required for security scenarios")
		}
		app.Route("POST", "/scenarios/security/users", (*controller.UserController).CreateUser,
			spineroute.WithInterceptors(&interceptor.SecurityTokenInterceptor{Token: token}))
	}
	return nil
}
