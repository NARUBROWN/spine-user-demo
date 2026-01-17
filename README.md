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
