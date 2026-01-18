# spine-user-demo

간단한 사용자 CRUD 예제(Spine 프레임워크).

## 둘러보기
Spine는 실행 흐름을 숨기지 않고, Controller 시그니처를 API 계약으로 삼는 실행 파이프라인 중심 프레임워크입니다.  
컨테이너 구성, 입력 해석, 반환 처리의 책임이 분리되어 있어 구조가 명확하고 테스트가 쉬운 것이 장점입니다.

생성자 의존성 예시 (Repository -> Service):
```go
type UserRepository struct{}

func NewUserRepository() *UserRepository {
    return &UserRepository{}
}

type UserService struct {
    repo *UserRepository
}

func NewUserService(repo *UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

핵심 특징:
- 컨테이너에 등록될 컴포넌트는 **생성자 함수**를 가져야 하며, 의존성은 생성자 파라미터로 명시합니다.
- Controller는 `path.Int`, `path.String`, `path.Boolean` 같은 의미 타입으로 **Path 입력을 선언**합니다.
- Query는 `query.Values`로 **가변 파라미터**를 받고, 필요 시 직접 해석할 수 있습니다.

예시:
```go
func (c *UserController) GetUser(userId path.Int, active path.Boolean, q query.Values) (User, error) {
    id := userId.Value
    enabled := active.Value
    keyword := q.String("q", "")
    // ...
}
```

DTO와 에러 처리 예시:
```go
type CreateUserRequest struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func (c *UserController) CreateUser(req CreateUserRequest) (User, error) {
    if req.ID <= 0 {
        return User{}, httperr.BadRequest("유효하지 않은 ID입니다")
    }
    if req.Name == "" {
        return User{}, httperr.BadRequest("이름이 비어 있습니다")
    }
    return c.userService.Create(req.ID, req.Name), nil
}
```

인터셉터 예시:
- CORS는 Spine에서 제공하는 인터셉터를 사용하고, 나머지는 사용자가 직접 구현합니다.
- 아래는 요청/응답 로깅을 위한 사용자 구현 예시입니다.

```go
type LoggingInterceptor struct{}

func (i *LoggingInterceptor) PreHandle(ctx core.ExecutionContext, meta core.HandlerMeta) error {
    log.Printf("[Logging Interceptor][REQ] %s %s -> %s.%s", ctx.Method(), ctx.Path(), meta.ControllerType.Name(), meta.Method.Name)
    return nil
}

func (i *LoggingInterceptor) PostHandle(ctx core.ExecutionContext, meta core.HandlerMeta) {
    log.Printf("[Logging Interceptor][RES] %s %s OK", ctx.Method(), ctx.Path())
}

func (i *LoggingInterceptor) AfterCompletion(ctx core.ExecutionContext, meta core.HandlerMeta, err error) {
    if err != nil {
        log.Printf("[Logging Interceptor][ERR] %s %s : %v", ctx.Method(), ctx.Path(), err)
    }
}
```

등록 예시:
```go
app.Interceptor(
    cors.New(cors.Config{
        AllowOrigins: []string{"*"},
        AllowMethods: []string{"GET", "POST", "OPTIONS"},
        AllowHeaders: []string{"Content-Type"},
    }),
    &interceptor.LoggingInterceptor{},
)
```

## 요구사항
- Go 1.25.5+

## 실행 방법
1. 의존성 다운로드
   ```sh
   go mod download
   ```
2. 서버 실행
   ```sh
   go run .
   ```
3. 서버 주소: `http://localhost:8080`

## 간단한 호출 예시
- 조회
  ```sh
  curl "http://localhost:8080/users/1?id=1"
  ```
- 생성
  ```sh
  curl -X POST "http://localhost:8080/users" \
    -H "Content-Type: application/json" \
    -d '{"id":2,"name":"alice"}'
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
