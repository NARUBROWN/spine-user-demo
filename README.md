# spine-user-demo

Spine의 HTTP·DI·요청 단위 트랜잭션 계약을 보여주는 사용자 CRUD 예제(Spine 프레임워크 + Bun ORM + Swagger).

## 둘러보기
Spine는 실행 흐름을 숨기지 않고, Controller 시그니처를 API 계약으로 삼는 실행 파이프라인 중심 프레임워크입니다.  
컨테이너 구성, 입력 해석, 반환 처리의 책임이 분리되어 있어 구조가 명확하고 테스트가 쉬운 것이 장점입니다.

이 프로젝트는 BUN ORM 기반 CRUD로 MySQL 연결 및 마이그레이션을 구성하고, Repository에서 `bun.IDB`로 `Select/Insert/Update/Delete`를 수행합니다.  
요청 단위 트랜잭션은 Tx 인터셉터에서 시작해 `ExecutionContext`에 보관한 뒤, HTTP 응답을 쓰기 전 `BeforeResponse`에서 커밋/롤백합니다.

이 예제는 Spine `v0.5.1`의 인터셉터 생명주기와 HTTP 범위 등록 계약을 따릅니다. HTTP 응답 직렬화와 상태·쿠키 검증이 끝난 뒤 `BeforeResponse`가 호출되므로, 응답 준비 실패도 요청 단위 트랜잭션 롤백에 반영됩니다.

추가로 다음 동작을 실제 HTTP 계약으로 제공합니다.

- 이름과 이메일, 사용자 ID를 검증하고 잘못된 입력은 `400 Bad Request`로 반환합니다.
- 존재하지 않는 사용자는 `404 Not Found`, MySQL 고유 키와 충돌하는 이메일은 `409 Conflict`로 반환합니다.
- `App.Validate`로 리스너를 열기 전에 애플리케이션 구성을 검사합니다.
- HTTP 범위 인터셉터가 요청 ID와 처리 시간을 영문 로그로 남깁니다. 유효한 `X-Request-ID`가 있으면 이어서 사용합니다.
- 성공 응답은 커밋하고 컨트롤러 오류와 응답 준비 오류는 롤백하는 경계를 Docker E2E로 검증합니다.

생성자 의존성 예시 (Repository -> Service):
```go
type UserRepository struct {
    db bun.IDB
}

func NewUserRepository(db bun.IDB) *UserRepository {
    return &UserRepository{db: db}
}

type UserService struct {
    repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

핵심 특징:
- 컨테이너에 등록될 컴포넌트는 **생성자 함수**를 가져야 하며, 의존성은 생성자 파라미터로 명시합니다.
- Controller는 `context.Context`, `query.Values`, DTO를 조합해 입력을 받습니다.
- Query는 `query.Values`로 **가변 파라미터**를 받고, 필요 시 직접 해석할 수 있습니다.
- Repository는 `bun.IDB`를 사용해 `bun.DB`와 `bun.Tx`를 동일한 인터페이스로 처리합니다.

예시:
```go
func (c *UserController) GetUser(
    ctx context.Context,
    q query.Values,
) (dto.CreateUserResponse, error) {
    userId := int(q.Int("id", 0))

    user, err := c.userService.Get(ctx, userId)
    if err != nil {
        return dto.CreateUserResponse{}, httperr.NotFound("유저를 찾을 수 없습니다.")
    }

    return user, nil
}
```

DTO 예시:
```go
type CreateUserRequest struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type UpdateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type CreateUserResponse struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

인터셉터 예시:
- CORS는 Spine에서 제공하는 인터셉터를 사용하고, 나머지는 사용자가 직접 구현합니다.
- 아래는 요청/응답 로깅을 위한 사용자 구현 예시입니다.

```go
type LoggingInterceptor struct{}

func (i *LoggingInterceptor) PreHandle(ctx core.ExecutionContext, meta core.HandlerMeta) error {
    log.Printf(
        "[Logging Interceptor][REQ] %s %s",
        ctx.Method(),
        ctx.Path(),
    )
    return nil
}

func (i *LoggingInterceptor) PostHandle(ctx core.ExecutionContext, meta core.HandlerMeta) {
    controllerName := "<unresolved>"
    methodName := "<unresolved>"
    if meta.ControllerType != nil {
        controllerName = meta.ControllerType.Name()
    }
    if meta.Method.Name != "" {
        methodName = meta.Method.Name
    }

    log.Printf(
        "[Logging Interceptor][RES] %s %s -> %s.%s OK",
        ctx.Method(),
        ctx.Path(),
        controllerName,
        methodName,
    )
}

func (i *LoggingInterceptor) BeforeResponse(ctx core.ExecutionContext, meta core.HandlerMeta, executionErr error) error {
    return nil
}

func (i *LoggingInterceptor) AfterCompletion(ctx core.ExecutionContext, meta core.HandlerMeta, err error) {
    if err != nil {
        log.Printf(
            "[Logging Interceptor][ERR] %s %s : %v",
            ctx.Method(),
            ctx.Path(),
            err,
        )
    }
}
```

Tx 인터셉터 예시:
```go
func (i *TxInterceptor) PreHandle(ctx core.ExecutionContext, meta core.HandlerMeta) error {
    reqCtx := ctx.Context()
    if reqCtx == nil {
        return errors.New("execution context has no request context")
    }

    tx, err := i.db.BeginTx(reqCtx, nil)
    if err != nil {
        return err
    }

    ctx.Set("tx", tx)
    return nil
}

func (i *TxInterceptor) BeforeResponse(ctx core.ExecutionContext, meta core.HandlerMeta, executionErr error) error {
    v, ok := ctx.Get("tx")
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

func (i *TxInterceptor) AfterCompletion(ctx core.ExecutionContext, meta core.HandlerMeta, err error) {
}
```

등록 예시:
```go
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
```

## 요구사항
- Go 1.25.5+

## 실행 방법
1. MySQL DSN 설정
   ```sh
   export SPINE_DEMO_MYSQL_DSN='root:change-me@tcp(localhost:3306)/spine_demo?parseTime=true&loc=Local'
   ```
2. 의존성 다운로드
   ```sh
   go mod download
   ```
3. 서버 실행
   ```sh
   go run .
   ```
4. 서버 주소: `http://localhost:8080`
5. Swagger UI: `http://localhost:8080/swagger/index.html`

## 간단한 호출 예시
- 조회
  ```sh
  curl "http://localhost:8080/users?id=1"
  ```
- 생성
  ```sh
  curl -X POST "http://localhost:8080/users" \
    -H "Content-Type: application/json" \
    -d '{"name":"alice","email":"alice@example.com"}'
  ```
- 수정
  ```sh
  curl -X PUT "http://localhost:8080/users?id=2" \
    -H "Content-Type: application/json" \
    -d '{"name":"bob"}'
  ```
- 삭제
  ```sh
  curl -X DELETE "http://localhost:8080/users?id=2"
  ```

## HTTP 오류 계약

| 상황 | 상태 |
| --- | --- |
| 본문 누락, 빈 이름, 잘못된 이메일, `id <= 0` | `400 Bad Request` |
| 존재하지 않는 사용자 조회·수정·삭제 | `404 Not Found` |
| 이미 등록된 이메일로 생성 | `409 Conflict` |
| 예상하지 못한 저장소·응답 준비 오류 | 본문에 내부 원인을 노출하지 않는 `500 Internal Server Error` |

## Docker E2E

Docker와 `curl`이 설치된 환경에서는 한 명령으로 MySQL 8.4 컨테이너, 마이그레이션, 실제 HTTP 서버를 실행해 전체 계약을 검증할 수 있습니다.

```sh
./scripts/e2e.sh
```

E2E는 다음 경계를 실제 MySQL 행 수로 확인합니다.

- 정상 사용자 생성 후 트랜잭션 커밋
- 중복 이메일 `409` 후 기존 행 유지
- 컨트롤러가 오류를 반환한 뒤 삽입 행 롤백
- 잘못된 응답 쿠키 때문에 응답 준비가 실패한 뒤 삽입 행 롤백

마지막 두 경로는 일반 실행에서는 노출되지 않습니다. 스크립트가 `SPINE_DEMO_ENABLE_FAILURE_SCENARIOS=1`을 설정할 때만 `/scenarios/transactions/*` 라우트가 등록됩니다. 직접 외부 인프라에 연결해 테스트하려면 다음 환경 변수를 지정합니다.

```sh
SPINE_DEMO_MYSQL_DSN='root:spine@tcp(127.0.0.1:13307)/spine_demo?parseTime=true&loc=Local' \
SPINE_DEMO_E2E_BASE_URL='http://127.0.0.1:18080' \
go test -tags=e2e -v ./e2e
```

스크립트는 종료할 때 테스트 서버와 MySQL 컨테이너 및 볼륨을 정리합니다. 별도 포트가 필요하면 `SPINE_DEMO_E2E_MYSQL_PORT`와 `SPINE_DEMO_E2E_HTTP_PORT`를 설정할 수 있습니다.
