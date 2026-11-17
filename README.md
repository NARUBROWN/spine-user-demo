English | [Korean](README.ko.md)

# spine-user-demo

A user CRUD example demonstrating Spine's HTTP, dependency injection, and request-scoped transaction contracts with Bun ORM and built-in OpenAPI.

**Built and verified with [Spine CLI](https://github.com/NARUBROWN/spine-cli).** Use the CLI to run the development server, watch and rebuild code, check execution contracts, and generate components. See [CLI commands and examples](https://github.com/NARUBROWN/spine-cli#readme) to start or extend a project.

## Example channels

| Channel | Branch | Purpose |
| --- | --- | --- |
| Preview | [`preview`](https://github.com/NARUBROWN/spine-user-demo/tree/preview) | Development changes and experimental versions |
| RC | [`rc`](https://github.com/NARUBROWN/spine-user-demo/tree/rc) | Release candidates before stable publication |
| Stable | [`main`](https://github.com/NARUBROWN/spine-user-demo/tree/main) | Public example using stable dependencies; the default branch |

Example code is promoted through `preview → rc → main`. Tags use `vX.Y.Z-preview.N`, `vX.Y.Z-rc.N`, and `vX.Y.Z`. **GitHub Releases are published only for Stable.** Check the selected branch's `go.mod` for the actual Spine version. See [channel checks and publication](RELEASE_CHANNELS.md).

## Run with Spine CLI

After installing Go and Spine CLI and completing [MySQL configuration and the initial migration](#running), run the development server from the repository root:

```sh
spine dev
```

The server defaults to `http://localhost:8080`. The CLI detects code changes, rebuilds, and replaces the server after a successful build. In another terminal, build an executable or verify contracts and code with:

```sh
spine build
spine verify
```

## Overview

Spine makes execution flow explicit and uses controller signatures as API contracts. Container composition, input resolution, and return handling have separate responsibilities.

This project configures MySQL connections and migrations for Bun CRUD. Repositories use `bun.IDB` for `Select/Insert/Update/Delete`. A Tx interceptor starts a request-scoped transaction, stores it in `ExecutionContext`, and commits or rolls back in `BeforeResponse` before writing the HTTP response.

The example follows Spine `v0.7.6-preview.1` interceptor lifecycle and HTTP-scoped registration contracts. `BeforeResponse` runs after response serialization and status/cookie validation, so response preparation failures also roll back the transaction.

The HTTP contract includes:

- Name, email, and user ID validation, returning `400 Bad Request` for invalid input.
- `404 Not Found` for missing users and `409 Conflict` for email uniqueness conflicts in MySQL.
- Configuration checks with `App.Validate` before opening a listener.
- HTTP-scoped request ID and duration logging in English, retaining a valid incoming `X-Request-ID`.
- Docker E2E checks for committing successful responses and rolling back controller errors and response preparation failures.

`constructor` is `github.com/NARUBROWN/spine/pkg/constructor`. Register all five roles, including custom components, in one `app.Constructor(...)` call using the same DI container. `constructor.Controller` checks construction and `RegisterRoutes`; `constructor.Interceptor` checks construction and the interceptor contract. Apply interceptors through `InterceptorFor`. Verification uses published modules.

```go
app.Constructor(
    constructor.Custom(database.NewBunDB),
    constructor.Repository(&repository.UserRepository{}),
    constructor.Service(&service.UserService{}),
    constructor.Controller(&controller.UserController{}),
    constructor.Interceptor(&interceptor.TxInterceptor{}),
)
```

Constructor dependency example (Repository → Service):

```go
type UserRepository struct {
    db bun.IDB
}

func (*UserRepository) Constructor(db *bun.DB) *UserRepository {
    return &UserRepository{db: db}
}

type UserService struct {
    repo userRepository
}

func (*UserService) Constructor(repo *repository.UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

Key behavior:

- Components registered as object markers must implement a **`Constructor` method**, declaring dependencies as parameters. Existing constructor functions for external types can use `constructor.Custom(NewX)`.
- Controllers receive `spine.Ctx` and explicit Query/Body wrappers. Retrieve the standard context for services and repositories with `ctx.ExecutionContext().Context()`.
- `input.Query[dto.UserQuery]` binds `ID` under the default name `id`; playground checks required and positive values.
- Repositories use `bun.IDB` to handle `bun.DB` and `bun.Tx` through one interface.

Example:

```go
type UserQuery struct { ID *int64 `validate:"required,gt=0"` }

func (c *UserController) RegisterRoutes(r *api.Registry) {
    r.GET("/users", func(ctx spine.Ctx, q input.Query[dto.UserQuery]) (httpx.Response[dto.CreateUserResponse], error) {
        userId := (*q).UserID()
        if userId <= 0 {
            return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest("id\ub294 1 \uc774\uc0c1\uc758 \uc815\uc218\uc5ec\uc57c \ud569\ub2c8\ub2e4.")
        }
        user, err := c.userService.Get(ctx.ExecutionContext().Context(), userId)
        if err != nil {
            return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
        }
        return httpx.Response[dto.CreateUserResponse]{Body: user}, nil
    })
}
```

IDs that fail numeric conversion or overflow are rejected with HTTP 400 during binding. Repeated `id` parameters are also rejected with HTTP 400. The former `query.Values` path used only the first value.

DTO example:

```go
type CreateUserRequest struct {
    ID    int    `json:"id"`
    Name  string `json:"name" validate:"required,max=100"`
    Email string `json:"email" validate:"required,email,max=255"`
}

type UpdateUserRequest struct {
    Name  string `json:"name" validate:"required,max=100"`
    Email string `json:"email"`
}

type CreateUserResponse struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

Interceptor example:

- CORS uses Spine's supplied interceptor; other interceptors are application implementations.
- The following custom interceptor logs requests and responses.

```go
package interceptor

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/NARUBROWN/spine/core"
	spinectx "github.com/NARUBROWN/spine/pkg/spine"
)

type LoggingInterceptor struct{}

const (
	requestIDKey = "request_id"
	startedAtKey = "request_started_at"
)

func (i *LoggingInterceptor) PreHandle(ctx spinectx.Ctx, meta core.HandlerMeta) error {
	requestID := requestID(httpRequest(ctx).Header("X-Request-ID"))
	ctx.ExecutionContext().Set(requestIDKey, requestID)
	ctx.ExecutionContext().Set(startedAtKey, time.Now())
	log.Printf(
		"request started request_id=%s method=%s path=%s",
		requestID,
		httpRequest(ctx).Method(),
		httpRequest(ctx).Path(),
	)
	return nil
}

func (i *LoggingInterceptor) PostHandle(ctx spinectx.Ctx, meta core.HandlerMeta) {
	controllerName := "<unresolved>"
	methodName := "<unresolved>"
	if meta.ControllerType != nil {
		controllerName = meta.ControllerType.String()
	}
	if meta.MethodName != "" {
		methodName = meta.MethodName
	}

	log.Printf(
		"handler completed request_id=%s method=%s path=%s handler=%s.%s",
		contextString(ctx, requestIDKey, "unknown"),
		httpRequest(ctx).Method(),
		httpRequest(ctx).Path(),
		controllerName,
		methodName,
	)
}

func (i *LoggingInterceptor) BeforeResponse(
	ctx spinectx.Ctx,
	meta core.HandlerMeta,
	executionErr error,
) error {
	return nil
}

func (i *LoggingInterceptor) AfterCompletion(
	ctx spinectx.Ctx,
	meta core.HandlerMeta,
	err error,
) {
	if err != nil {
		log.Printf(
			"request failed request_id=%s method=%s path=%s duration=%s error=%v",
			contextString(ctx, requestIDKey, "unknown"),
			httpRequest(ctx).Method(),
			httpRequest(ctx).Path(),
			duration(ctx),
			err,
		)
		return
	}
	log.Printf(
		"request completed request_id=%s method=%s path=%s duration=%s",
		contextString(ctx, requestIDKey, "unknown"),
		httpRequest(ctx).Method(),
		httpRequest(ctx).Path(),
		duration(ctx),
	)
}

func requestID(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate != "" && len(candidate) <= 128 && strings.IndexFunc(candidate, unicode.IsControl) == -1 {
		return candidate
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return "unavailable"
}

func contextString(ctx spinectx.Ctx, key string, fallback string) string {
	value, ok := ctx.ExecutionContext().Get(key)
	if !ok {
		return fallback
	}
	result, ok := value.(string)
	if !ok {
		return fallback
	}
	return result
}

func duration(ctx spinectx.Ctx) time.Duration {
	value, ok := ctx.ExecutionContext().Get(startedAtKey)
	if !ok {
		return 0
	}
	startedAt, ok := value.(time.Time)
	if !ok {
		return 0
	}
	return time.Since(startedAt).Round(time.Microsecond)
}

func httpRequest(ctx spinectx.Ctx) spinectx.HTTPRequestView {
	request, ok := ctx.HTTPRequestContext()
	if !ok {
		panic("HTTP interceptor requires an HTTP request context")
	}
	return request
}

func (*LoggingInterceptor) Constructor() *LoggingInterceptor { return &LoggingInterceptor{} }
```

Transaction interceptor example:

```go
package interceptor

import (
	"errors"

	"github.com/NARUBROWN/spine/core"
	spinectx "github.com/NARUBROWN/spine/pkg/spine"
	"github.com/uptrace/bun"
)

type TxInterceptor struct {
	db *bun.DB
}

func (*TxInterceptor) Constructor(db *bun.DB) *TxInterceptor {
	return &TxInterceptor{db: db}
}

func (i *TxInterceptor) PreHandle(ctx spinectx.Ctx, meta core.HandlerMeta) error {
	reqCtx := ctx.ExecutionContext().Context()
	if reqCtx == nil {
		return errors.New("execution context has no request context")
	}

	tx, err := i.db.BeginTx(reqCtx, nil)
	if err != nil {
		return err
	}

	// Store tx in ExecutionContext
	ctx.ExecutionContext().Set("tx", tx)
	return nil
}

func (i *TxInterceptor) PostHandle(ctx spinectx.Ctx, meta core.HandlerMeta) {}

func (i *TxInterceptor) BeforeResponse(
	ctx spinectx.Ctx,
	meta core.HandlerMeta,
	executionErr error,
) error {
	v, ok := ctx.ExecutionContext().Get("tx")
	if !ok {
		return nil
	}

	tx, ok := v.(bun.Tx)
	if !ok {
		return errors.New("execution context has an invalid transaction")
	}

	if executionErr != nil {
		return tx.Rollback()
	}

	return tx.Commit()
}

func (i *TxInterceptor) AfterCompletion(
	ctx spinectx.Ctx,
	meta core.HandlerMeta,
	err error,
) {
}
```

Register all interceptors with `app.Constructor` and retrieve them from the container using typed nils. Register the CORS factory with `constructor.Custom`.

Registration example:

```go
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
```

## Requirements

- Minimum Go version: 1.26.0, required by the playground validator dependency.
- Verified Go version: 1.26.0.

## Spine CLI project layout

Uses Spine CLI v0.4.5-preview.5 platform composition and controller-owned route registration. Requires Go `1.26.0` or later. Install CLI v0.4.5-preview.5 with:

```sh
go install github.com/NARUBROWN/spine-cli/cmd/spine@v0.4.5-preview.5
spine version
```

Install Preview CLI `v0.4.5-preview.5`. CI pins the verified version in `.github/example-channels.json`.

The HTTP application uses Spine `v0.7.6-preview.1`, Echo platform `v0.2.1-preview.1`, playground validator `v0.2.1-preview.1`, and Echo `v5.4.4`. See the [official CLI repository](https://github.com/NARUBROWN/spine-cli#readme) for installation and all commands.

The project uses `spine new`'s default `layered-v1` layout. Existing Bun CRUD, validation, error mapping, and transactions were moved into the skeleton created by `spine g resource user` and the DTO generator. HTTP methods and paths are registered in the controller's `RegisterRoutes`.

```text
.
├── spine.toml                     # layered-v1, Echo platform, composition package
├── main.go                        # Migration flag, environment validation, startup
├── database/domain/user.go        # Existing Bun model and table mapping
├── internal/
│   ├── application/               # Platform, constructors, interceptors, route wiring
│   ├── database/                  # DSN validation and lazy Bun constructor
│   └── interceptor/               # HTTP logging and request-scoped transactions
├── src/
│   ├── controller/                # User HTTP handlers
│   ├── service/                   # CRUD and error mapping
│   ├── repository/                # Bun queries
│   └── dto/                       # Requests, responses, and input validation
├── migrations/                    # Existing SQL and migration execution
├── e2e/
└── scripts/e2e.sh
```

`BuildApplication` registers constructors by role; controller `RegisterRoutes` methods register HTTP routes. `spine.New(echoplatform.Option(), spine.WithValidator(validator))` selects Echo v5 and playground. `boot.HTTPOptions.OpenAPI` configures built-in OpenAPI and its UI. `BuildApplication` neither connects to the database nor opens a listener.

`spine.toml` records `recipe = "platform-contracts-preview"`, `websocket_platform = "none"`, and `validator = "playground"`. The validator created by `BuildApplication` is shared by request validation and specification generation. `spine update` updates only this recipe's dependencies; migrate existing application composition code manually.

```sh
spine update --dry-run --json
# Review changes, diff, and base_hash before applying.
spine update --if-match <base_hash> --json
```

Preview generation for a new resource in the same way:

```sh
spine g resource product --dry-run --json
```

Review `changes`, `diff`, and `base_hash`, then pass `--if-match <base_hash>` to the same command. The resource generator creates domain, repository, service, and controller components and connects their constructors. Add DTOs and HTTP routes separately; implement Bun queries and transactions yourself.

Run these checks without connecting to the database or running migrations:

```sh
spine context --root . --json
spine contract check --root . --warnings-as-errors --json
spine verify --root . --warnings-as-errors --json
spine build --root .
spine contract probe --root . --json
```

`./scripts/check.sh` runs these checks, the CLI build, and race checks. To use a CLI from another location, set `SPINE_CLI_BIN=/path/to/spine ./scripts/check.sh`.

`BuildApplication` calls `internal/config.ValidateEnvironment` to validate security-scenario tokens before application composition. Missing tokens return `boot.ConfigError`; controllers do not validate configuration keys.

For CLI analysis, test routes in `RegisterRoutes` directly compare environment variables to `"1"`. `SPINE_DEMO_ENABLE_FAILURE_SCENARIOS` enables error/rollback routes; `SPINE_DEMO_ENABLE_SECURITY_SCENARIOS` enables authentication checks. Pass the same variables to CLI checks and server execution. Partial probes do not check database connections, real HTTP, or interceptor behavior; also run Docker E2E below.

## Running

1. Configure the MySQL DSN:

   ```sh
   export SPINE_DEMO_MYSQL_DSN='root:change-me@tcp(localhost:3306)/spine_demo?parseTime=true&loc=Local'
   ```

2. Download dependencies:

   ```sh
   go mod download
   ```

3. Run migrations before the first start:

   ```sh
   go run . -migrate
   ```

4. Start the server with the CLI:

   ```sh
   spine dev
   # Or spine run / go run .
   ```

5. Server: `http://localhost:8080`.
6. Built-in API UI: `http://localhost:8080/docs`; OpenAPI JSON: `http://localhost:8080/openapi.json`.

## Request examples

Read:

```sh
curl "http://localhost:8080/users?id=1"
```

Create:

```sh
curl -X POST "http://localhost:8080/users"   -H "Content-Type: application/json"   -d '{"name":"alice","email":"alice@example.com"}'
```

Update:

```sh
curl -X PUT "http://localhost:8080/users?id=2"   -H "Content-Type: application/json"   -d '{"name":"bob"}'
```

Delete:

```sh
curl -X DELETE "http://localhost:8080/users?id=2"
```

## HTTP error contract

| Condition | Status |
| --- | --- |
| Missing body, empty name, invalid email, or `id <= 0` | `400 Bad Request` |
| Read/update/delete of a missing user | `404 Not Found` |
| Create with an existing email | `409 Conflict` |
| Unexpected storage or response preparation failure | `500 Internal Server Error`, without exposing internal causes in the body |

## Docker E2E

With Docker and `curl`, one command starts MySQL 8.4, runs migrations, and starts a real HTTP server to verify the full contract.

```sh
./scripts/e2e.sh
```

E2E checks transaction boundaries using actual MySQL row counts:

- Commit after successful user creation.
- Preserve existing rows after duplicate email `409`.
- Roll back inserted rows when the controller returns an error.
- Roll back inserted rows when an invalid response cookie causes response preparation to fail.
- Do not insert users when authentication fails on an optional protected route.
- Reject unauthenticated invalid bodies before binding; return 400 for invalid bodies with valid authentication.

Controller/response failure routes are absent during ordinary execution. `/scenarios/transactions/*` is registered only when the script sets `SPINE_DEMO_ENABLE_FAILURE_SCENARIOS=1`.

The security test route `POST /scenarios/security/users` requires both `SPINE_DEMO_ENABLE_SECURITY_SCENARIOS=1` and a nonempty `SPINE_DEMO_SECURITY_TOKEN`. It checks `Authorization: Bearer <token>` before processing the body. This is a shared test credential, not a per-user authentication/authorization example. Existing `/users` routes do not use this authentication. E2E scripts set a temporary token only for isolated local tests.

HTTP pipeline tests in `internal/interceptor` verify that controllers do not run on authentication failures or invalid bodies, small body limits are applied, and concurrent request contexts are isolated. They use Spine and Echo platform Prepare without MySQL. When running E2E against external infrastructure directly, enable security scenarios and the token on the server, then use:

```sh
SPINE_DEMO_MYSQL_DSN='root:spine@tcp(127.0.0.1:13307)/spine_demo?parseTime=true&loc=Local' \
SPINE_DEMO_E2E_BASE_URL='http://127.0.0.1:18080' \
SPINE_DEMO_E2E_SECURITY_TOKEN='<test-token-configured-on-server>' \
go test -tags=e2e -v ./e2e
```

The script cleans up the test server, MySQL container, and volumes on exit. Set `SPINE_DEMO_E2E_MYSQL_PORT` and `SPINE_DEMO_E2E_HTTP_PORT` for alternative ports.

## Controller route registration

Uses `api.Controller.RegisterRoutes` from published Spine `v0.7.6-preview.1`. It does not automatically migrate or restart the existing User application or database.

`src/controller/user_controller.go` declares CRUD routes and callbacks together; `internal/application` registers concrete constructors. There is no separate src/route/route.go. Query and body inputs use wrappers such as `input.Query[dto.UserQuery]` and `input.Body[dto.CreateUserRequest]`, retaining DTO normalization/validation, transactions, error responses, and optional failure/security scenarios.

Before preparation, `spine.DescribeHTTP` returns ErrRoutesNotCollected without executing constructors. Use CLI static export, or explicitly Prepare the application before calling DescribePreparedHTTP. Database and broker initialization are part of explicit preparation.

## Built-in OpenAPI and API UI

External specification generator dependencies, annotations, and generated snapshots have been removed. During preparation, the application generates an OpenAPI 3.1 specification from actual registrations and retains it in memory. Spine supplies the UI and static assets; no extra Echo handler or external CDN is required.

```go
validator, err := playground.New()
if err != nil {
    return nil, boot.Options{}, err
}
app := spine.New(echoplatform.Option(), spine.WithValidator(validator))

options := boot.Options{
    HTTP: &boot.HTTPOptions{
        OpenAPI: boot.OpenAPIOptions{
            Enabled: true,
            Title: "Spine User Demo API",
            Version: "0.2.0",
            UI: []boot.OpenAPIUI{
                swaggerui.New(swaggerui.Options{Title: "Spine User Demo API"}),
            },
        },
    },
}
```

Import `playground` from `github.com/NARUBROWN/spine-validator-playground` and `swaggerui` from `github.com/NARUBROWN/spine/pkg/openapi/ui/swaggerui`. JSON is served at `/openapi.json`; the UI is at `/docs`. The old `/swagger/*` route is not provided.

The runtime specification includes query `id`, create/update bodies, `CreateUserResponse`, and the HTTP 400 error schema. It reads these constraints from DTO `validate` tags; the same validator checks requests before handler execution.

| Input | Validation and specification |
| --- | --- |
| Query `id` | Required integer greater than `0` |
| Create/update `name` | Required string, maximum 100 characters |
| Create `email` | Required email string, maximum 255 characters |

Tag validation applies to original input. Existing manual `Validate()` then retains trimming, whitespace-only name rejection, email byte limits, and `net/mail` checks. Consequently, surrounding whitespace in email or a name longer than 100 characters before trimming is rejected with HTTP 400 by tag validation. Create request `id` is unused because the database allocates it; update request `email` is not changed.

Field validation errors include input location, JSON Pointer, and the failed rule in HTTP 400 `fields`. Export a static specification separately with:

```sh
spine contract export --format openapi --source static --output openapi.json
```

Additional manual `Validate()` rules, dynamic status codes, and authentication policies are not inferred automatically. Response DTOs appear under the `default` response; business errors such as `404` and `409` are not inferred as separate responses.

## Pinned dependencies

Uses Spine `v0.7.6-preview.1`, CLI `v0.4.5-preview.5`, Echo platform `v0.2.1-preview.1`, and playground validator `v0.2.1-preview.1`. Actual dependencies are pinned in `go.mod`. `spine.toml` retains the CLI-supported `platform-contracts-preview` recipe name and selects the `preview` dependency channel.
