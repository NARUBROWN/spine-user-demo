# spine-user-demo

Spine의 HTTP, DI, 요청 단위 트랜잭션 계약을 보여주는 사용자 CRUD 예제(Spine 프레임워크 + Bun ORM + Swagger).

**[Spine CLI](https://github.com/NARUBROWN/spine-cli)로 빌드하고 검증한 예제입니다.**
Spine CLI로 개발 서버 실행, 코드 변경 감지와 자동 재빌드, 실행 계약 검사, 구성 요소 생성을 이어갈 수 있습니다.
새로운 Spine 프로젝트를 만들거나 이 예제를 확장하려면 [Spine CLI의 명령과 사용 예제](https://github.com/NARUBROWN/spine-cli#readme)를 확인하세요.

## 예제 채널 선택

| 채널 | 브랜치 | 용도 |
| --- | --- | --- |
| Preview | [`preview`](https://github.com/NARUBROWN/spine-user-demo/tree/preview) | 개발 중인 변경과 실험 버전을 확인합니다. |
| RC | [`rc`](https://github.com/NARUBROWN/spine-user-demo/tree/rc) | 정식 공개 전 릴리스 후보를 확인합니다. |
| Stable | [`main`](https://github.com/NARUBROWN/spine-user-demo/tree/main) | 정식 의존성을 사용하는 공개 예제이며 기본 브랜치입니다. |

예제 코드는 `preview → rc → main` 순서로 승격합니다. 예제 태그는 `vX.Y.Z-preview.N`, `vX.Y.Z-rc.N`, `vX.Y.Z`로 구분하며, **GitHub Release는 Stable에서만 발행합니다.** 실제 Spine 버전은 선택한 브랜치의 `go.mod`를 확인하세요. [채널 검사와 발행 절차](RELEASE_CHANNELS.md)를 참고하세요.

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

이 Preview 소스 예제는 Spine `v0.7.0-preview.5` 후보의 인터셉터 생명주기와 HTTP 범위 등록 계약을 따릅니다. HTTP 응답 직렬화와 상태, 쿠키 검증이 끝난 뒤 `BeforeResponse`가 호출되므로, 응답 준비 실패도 요청 단위 트랜잭션 롤백에 반영됩니다.

추가로 다음 동작을 실제 HTTP 계약으로 제공합니다.

- 이름과 이메일, 사용자 ID를 검증하고 잘못된 입력은 `400 Bad Request`로 반환합니다.
- 존재하지 않는 사용자는 `404 Not Found`, MySQL 고유 키와 충돌하는 이메일은 `409 Conflict`로 반환합니다.
- `App.Validate`로 리스너를 열기 전에 애플리케이션 구성을 검사합니다.
- HTTP 범위 인터셉터가 요청 ID와 처리 시간을 영문 로그로 남깁니다. 유효한 `X-Request-ID`가 있으면 이어서 사용합니다.
- 성공 응답은 커밋하고 컨트롤러 오류와 응답 준비 오류는 롤백하는 경계를 Docker E2E로 검증합니다.

역할별 등록은 같은 DI 컨테이너를 공유합니다. `ControllerConstructor`는 생성 메서드와 `RegisterRoutes`를, `InterceptorConstructor`는 생성 메서드와 기존 인터셉터 계약을 검사합니다. 인터셉터 적용은 기존 `InterceptorFor`로 지정합니다. 이 API는 미공개 코어 후보 소스에서 검증했습니다.

```go
app.Constructor(database.NewBunDB)
app.RepositoryConstructor(&repository.UserRepository{})
app.ServiceConstructor(&service.UserService{})
app.ControllerConstructor(&controller.UserController{})
app.InterceptorConstructor(&interceptor.TxInterceptor{})
```

생성자 의존성 예시 (Repository -> Service):
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

핵심 특징:
- 컨테이너에 등록될 컴포넌트는 **`Constructor` 메서드**를 가져야 하며, 의존성은 생성자 파라미터로 명시합니다.
- Controller는 `context.Context`, 쿼리 구조체 포인터와 본문 DTO를 조합해 입력을 받습니다.
- `input.Query[dto.UserQuery]`는 `ID`를 기본 이름 `id`로 바인딩하며 누락과 양수 여부는 핸들러에서 검사합니다.
- Repository는 `bun.IDB`를 사용해 `bun.DB`와 `bun.Tx`를 동일한 인터페이스로 처리합니다.

예시:
```go
type UserQuery struct { ID *int64 }

func (c *UserController) RegisterRoutes(r *api.Registry) {
    r.GET("/users", func(ctx context.Context, q input.Query[dto.UserQuery]) (httpx.Response[dto.CreateUserResponse], error) {
        userId := (*q).UserID()
        if userId <= 0 {
            return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest("id는 1 이상의 정수여야 합니다.")
        }
        user, err := c.userService.Get(ctx, userId)
        if err != nil {
            return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
        }
        return httpx.Response[dto.CreateUserResponse]{Body: user}, nil
    })
}
```

숫자 변환에 실패하거나 범위를 넘는 ID는 바인딩 단계에서 HTTP 400으로 거부됩니다. 같은 `id`를 여러 번 전달하는 요청도 HTTP 400으로 거부합니다. 이전 `query.Values`는 첫 번째 값만 사용했습니다.

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

## 권장 환경
- 최소 Go 버전: 1.25.5
- 검증한 Go 버전: 1.26.0

## Spine CLI 프로젝트 구조

공개 Spine CLI Preview의 플랫폼 조립 방식과 Spine `v0.7.0-preview.2`에 맞춰 기존 API와 처리 로직을 구성했습니다.
Go `1.25.5` 이상에서 다음 명령으로 CLI Preview를 설치할 수 있습니다. 이번 검증에는 Go `1.26.0`을 사용했습니다.

```sh
go install github.com/NARUBROWN/spine-cli/cmd/spine@preview
spine version
```

`@preview`는 설치 시점의 CLI Preview 브랜치를 사용합니다. CI 검증 버전은 `.github/example-channels.json`에 고정합니다.

HTTP 앱은 Spine `v0.7.0-preview.2`, Echo 플랫폼 `v0.2.0-preview.1`, Echo `v5.4.4`를 사용합니다.
설치와 전체 명령은 [공식 CLI 저장소](https://github.com/NARUBROWN/spine-cli#readme)를 참고하세요.

`spine new`의 기본 `layered-v1` 구조를 사용합니다. `spine g resource user`와 DTO 생성기로
만든 뼈대에 기존 Bun CRUD, 검증, 오류 매핑과 트랜잭션 처리를 옮겼습니다.
`spine g route`로 기존 HTTP 메서드와 경로를 명시적으로 등록했습니다.

```text
.
├── spine.toml                     # layered-v1, Echo 플랫폼, 구성 패키지
├── main.go                        # 마이그레이션 플래그, 환경 검증과 실행
├── database/domain/user.go        # 기존 Bun 모델과 테이블 매핑
├── internal/
│   ├── application/               # 플랫폼, 생성자, 인터셉터, 라우트 연결
│   ├── database/                  # DSN 검증과 지연 Bun 생성자
│   └── interceptor/               # HTTP 로깅과 요청 단위 트랜잭션
├── src/
│   ├── controller/                # 사용자 HTTP 핸들러
│   ├── service/                   # CRUD와 오류 매핑
│   ├── repository/                # Bun 쿼리
│   ├── dto/                       # 요청, 응답과 입력 검증
│   └── route/route.go             # 사용자 라우트와 선택적 실패 시나리오
├── migrations/                    # 기존 SQL과 마이그레이션 실행
├── docs/                          # Swagger 문서
├── e2e/
└── scripts/e2e.sh
```

생성자는 `BuildApplication`에서 연결하고, HTTP 경로는 `route.Register(app)`로 등록합니다.
`spine.New(echoplatform.Option(echoplatform.WithEcho(...)))`가 Echo v5 플랫폼과 기존 `/swagger/*` 경로를 구성합니다.
`BuildApplication`은 DB에 접속하거나 리스너를 열지 않습니다.

`spine.toml`에는 `recipe = "platform-contracts-v0.7.0-preview.2"`, `websocket_platform = "none"`,
`validator = "none"`을 기록합니다. 기존 DTO 검증과 HTTP 오류 처리를 유지하며 별도 검증기는 선택하지 않습니다.
`spine update`는 이 레시피의 의존성만 갱신하므로 기존 앱의 구성 코드는 직접 옮겨야 합니다.

```sh
spine update --dry-run --json
# changes, diff, base_hash를 검토한 뒤 적용합니다.
spine update --if-match <base_hash> --json
```

새 리소스의 생성 계획도 같은 방식으로 확인할 수 있습니다.

```sh
spine g resource product --dry-run --json
```

출력의 `changes`, `diff`, `base_hash`를 확인한 뒤 같은 명령에 `--if-match <base_hash>`를
전달합니다. 리소스 생성기는 domain, repository, service, controller를 생성하고 생성자를
연결합니다. DTO와 HTTP 경로는 별도로 추가하며, Bun 쿼리와 트랜잭션은 직접 구현합니다.

DB 연결이나 마이그레이션 없이 다음 검사를 실행합니다.

```sh
spine context --root . --json
spine contract check --root . --warnings-as-errors --json
spine verify --root . --warnings-as-errors --json
spine build --root .
spine contract probe --root . --json
```

`./scripts/check.sh`는 위 검사와 CLI 빌드, race 검사를 함께 실행합니다.
다른 위치의 CLI를 사용하려면 `SPINE_CLI_BIN=/경로/spine ./scripts/check.sh`로 지정합니다.

HTTP 기본 panic 복구가 활성화됩니다. 테스트 라우트는 SPINE_DEMO_ENABLE_FAILURE_SCENARIOS=1일 때만 계약과 앱에
포함됩니다. CLI 검사와 서버 실행에 같은 환경변수를 전달해야 합니다. 부분 probe는 DB 연결이나
실제 HTTP, 인터셉터 동작을 확인하지 않으므로 아래 Docker E2E도 별도로 실행합니다.

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
| 존재하지 않는 사용자 조회, 수정, 삭제 | `404 Not Found` |
| 이미 등록된 이메일로 생성 | `409 Conflict` |
| 예상하지 못한 저장소, 응답 준비 오류 | 본문에 내부 원인을 노출하지 않는 `500 Internal Server Error` |

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
- 선택적 보호 라우트의 인증 실패 시 사용자 행이 추가되지 않음
- 인증되지 않은 잘못된 본문은 바인딩 전에 거부되고, 유효한 인증의 잘못된 본문은 400으로 처리됨

컨트롤러·응답 오류 경로는 일반 실행에서는 노출되지 않습니다. 스크립트가 `SPINE_DEMO_ENABLE_FAILURE_SCENARIOS=1`을 설정할 때만 `/scenarios/transactions/*` 라우트가 등록됩니다.

보안 검증용 `POST /scenarios/security/users`는 `SPINE_DEMO_ENABLE_SECURITY_SCENARIOS=1`과 비어 있지 않은 `SPINE_DEMO_SECURITY_TOKEN`을 함께 지정해야 등록됩니다. `Authorization: Bearer <토큰>`을 검사하고, 본문 처리는 인증 성공 후에 진행합니다. 이 토큰은 테스트용 공유 자격 증명이며 사용자별 인증·인가 예제는 아닙니다. 기존 `/users` 경로에는 이 인증이 적용되지 않습니다. E2E 스크립트는 격리된 로컬 실행에만 사용할 임시 테스트 토큰을 설정합니다.

`internal/interceptor`의 HTTP 파이프라인 테스트는 인증 실패와 잘못된 본문에서 컨트롤러 미실행, 작은 본문 제한의 적용, 동시 요청의 컨텍스트 격리를 검사합니다. 해당 테스트는 MySQL 없이 Spine과 Echo 플랫폼의 `Prepare` 경로를 사용합니다. 직접 외부 인프라에 연결해 E2E를 실행할 때는 서버에도 보안 시나리오 활성화 변수와 토큰을 지정한 뒤 다음 환경 변수를 사용합니다.

```sh
SPINE_DEMO_MYSQL_DSN='root:spine@tcp(127.0.0.1:13307)/spine_demo?parseTime=true&loc=Local' \
SPINE_DEMO_E2E_BASE_URL='http://127.0.0.1:18080' \
SPINE_DEMO_E2E_SECURITY_TOKEN='<서버에 설정한 테스트 토큰>' \
go test -tags=e2e -v ./e2e
```

스크립트는 종료할 때 테스트 서버와 MySQL 컨테이너 및 볼륨을 정리합니다. 별도 포트가 필요하면 `SPINE_DEMO_E2E_MYSQL_PORT`와 `SPINE_DEMO_E2E_HTTP_PORT`를 설정할 수 있습니다.

## 컨트롤러 경로 등록 후보

이 소스는 코어 v0.7.0-preview.5 후보의 `api.Controller.RegisterRoutes`를 사용합니다. 공개 코어 태그가 준비되기 전에는 짝이 맞는 로컬 소스 조합에서 검증합니다. 기본 User 앱과 기존 DB를 자동으로 이전하거나 재기동하지 않습니다.

`src/controller/user_controller.go`가 CRUD 경로와 콜백을 함께 선언하고 `internal/application`은 구체 생성자를 등록합니다. 별도 src/route/route.go는 사용하지 않습니다. Query와 Body는 `input.Query[dto.UserQuery]`, `input.Body[dto.CreateUserRequest]` 등으로 받으며 기존 DTO 정규화와 검증, 트랜잭션, 오류 응답 및 선택적 실패와 보안 시나리오를 유지합니다.

준비 전 `spine.DescribeHTTP`는 생성자를 실행하지 않고 ErrRoutesNotCollected를 반환합니다. CLI 정적 내보내기를 사용하거나, 앱을 명시적으로 Prepare한 뒤 DescribePreparedHTTP를 조회하세요. DB와 브로커 초기화는 명시한 준비 과정의 일부입니다.

기존 Swagger UI와 추적 중인 Swagger 스냅샷은 유지합니다. Swaggo의 메서드 주석 수집이 익명 콜백을 자동으로 해석한다고 가정하지 않습니다. 새 코드의 경로와 요청 타입 변경은 `spine contract export --format openapi --source static --output openapi.json`으로 확인하세요. 수동 Validate의 업무 규칙과 동적 상태 코드는 자동 추론하지 않습니다.
