# spine-user-demo

간단한 사용자 CRUD 예제(Spine 프레임워크).

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
