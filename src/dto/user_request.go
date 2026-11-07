package dto

import (
	"fmt"
	"net/mail"
	"strings"
)

// UserQuery는 사용자 ID를 쿼리 문자열에서 받습니다.
type UserQuery struct {
	ID *int64 `validate:"required,gt=0"`
}

// UserID는 누락된 ID를 0으로 처리합니다.
func (q *UserQuery) UserID() int {
	if q == nil || q.ID == nil {
		return 0
	}
	return int(*q.ID)
}

type CreateUserRequest struct {
	ID    int    `json:"id"`
	Name  string `json:"name" validate:"required,max=100"`
	Email string `json:"email" validate:"required,email,max=255"`
}

type UpdateUserRequest struct {
	Name  string `json:"name" validate:"required,max=100"`
	Email string `json:"email"`
}

func (r *CreateUserRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("요청 본문이 비어 있습니다")
	}
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.TrimSpace(r.Email)
	if err := validateName(r.Name); err != nil {
		return err
	}
	return validateEmail(r.Email)
}

func (r *UpdateUserRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("요청 본문이 비어 있습니다")
	}
	r.Name = strings.TrimSpace(r.Name)
	return validateName(r.Name)
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("이름은 필수입니다")
	}
	if len([]rune(name)) > 100 {
		return fmt.Errorf("이름은 100자 이하여야 합니다")
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("이메일은 필수입니다")
	}
	if len(email) > 255 {
		return fmt.Errorf("이메일은 255자 이하여야 합니다")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return fmt.Errorf("올바른 이메일 형식이 아닙니다")
	}
	return nil
}
