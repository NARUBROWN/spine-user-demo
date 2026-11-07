package dto

type CreateUserResponse struct {
	ID    int    `json:"id" example:"1" description:"사용자 ID"`
	Name  string `json:"name" example:"홍길동" description:"사용자 이름"`
	Email string `json:"email" example:"user@example.com" description:"이메일 주소"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
