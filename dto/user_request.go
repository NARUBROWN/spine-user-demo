package dto

import (
	"fmt"
	"net/mail"
	"strings"
)

type CreateUserRequest struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUserRequest struct {
	Name  string `json:"name"`
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
