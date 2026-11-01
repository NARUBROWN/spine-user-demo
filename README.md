# spine-user-demo

Spine의 HTTP·DI·요청 단위 트랜잭션 계약을 보여주는 사용자 CRUD 예제(Spine 프레임워크 + Bun ORM + Swagger).

**[Spine CLI](https://github.com/NARUBROWN/spine-cli)로 빌드하고 검증한 예제입니다.**
Spine CLI로 개발 서버 실행, 코드 변경 감지와 자동 재빌드, 실행 계약 검사, 구성 요소 생성을 이어갈 수 있습니다.
새로운 Spine 프로젝트를 만들거나 이 예제를 확장하려면 [Spine CLI의 명령과 사용 예제](https://github.com/NARUBROWN/spine-cli#readme)를 확인하세요.

## Spine CLI로 실행하기

Go와 Spine CLI를 설치하고 [MySQL 연결 설정과 최초 마이그레이션](#실행-방법)을 마쳤다면,
이 저장소 루트에서 다음 명령으로 개발 서버를 실행할 수 있습니다.

```sh
spine dev
```

서버는 기본적으로 `http://localhost:8080`에서 실행됩니다. 코드를 수정하면 CLI가 변경을 감지해
자동으로 다시 빌드하고, 빌드가 성공하면 서버를 새 코드로 교체합니다.
개발 서버를 실행 중인 터미널과 별도로, 다음 명령으로 실행 파일을 빌드하거나 계약과 코드를 검증할 수 있습니다.

```sh
spine build
spine verify
```

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
- Go 1.27.3 이상 (Go 1.27.x 검증 기준: 1.27.3)

## Spine CLI 프로젝트 구조

Spine CLI 설치와 전체 사용법은 [공식 CLI 저장소](https://github.com/NARUBROWN/spine-cli#readme)를 참고하세요.
이 예제의 계약 검증에는 이번 호환 보완이 포함된 개발 작업본이 필요합니다. 해당 `spine-cli` 소스
디렉터리에서 `go install ./cmd/spine`으로 설치합니다.
공개 `v0.1.4` 태그에는 환경변수 등록과 패키지별 핸들러 분석 보완이 포함되어 있지 않습니다.

```text
.
├── spine.toml
├── main.go                       # 플래그 처리, 환경 검증, 앱 실행
├── internal/
│   ├── application/
│   │   ├── application.go        # BuildApplication: DI·인터셉터·라우트·옵션
│   │   └── application_test.go
│   └── database/
│       ├── database.go           # DSN 검증과 실행 시점의 Bun 생성자
│       └── database_test.go
├── migrations/
│   ├── migrations.go            # SQL 임베딩과 명시적 마이그레이션 실행
│   └── *.sql
├── controller/ dto/ entity/ interceptor/ repository/ routes/ service/
├── e2e/
└── scripts/e2e.sh
```

기존 main의 앱 구성은 internal/application, DB 코드는 internal/database, SQL 임베딩과 실행은
migrations/migrations.go로 옮겼습니다. main_test.go의 DSN 검증도 database 패키지로 이동했습니다.
기존 도메인 패키지와 SQL 파일의 위치는 유지했습니다.

DB 연결이나 마이그레이션 없이 다음 검사를 실행합니다.

```sh
spine context --root . --json
spine contract check --root . --json
spine verify --root . --json
spine contract probe --root . --json
```

기존 DisableRecover 설정은 HTTP_RECOVERY_DISABLED 경고로 표시되며, --warnings-as-errors를
붙이면 실패합니다. 테스트 라우트는 SPINE_DEMO_ENABLE_FAILURE_SCENARIOS=1일 때만 계약과 앱에
포함됩니다. CLI 검사와 서버 실행에 같은 환경변수를 전달해야 합니다. 부분 probe는 DB 연결이나
실제 HTTP·인터셉터 동작을 확인하지 않으므로 아래 Docker E2E도 별도로 실행합니다.

## 실행 방법
1. MySQL DSN 설정
   ```sh
   export SPINE_DEMO_MYSQL_DSN='root:change-me@tcp(localhost:3306)/spine_demo?parseTime=true&loc=Local'
   ```
2. 의존성 다운로드
   ```sh
   go mod download
   ```
3. 최초 실행 전 마이그레이션
   ```sh
   go run . -migrate
   ```
4. CLI로 서버 실행
   ```sh
   spine dev
   # 또는 spine run / go run .
   ```
5. 서버 주소: `http://localhost:8080`
6. Swagger UI: `http://localhost:8080/swagger/index.html`

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
