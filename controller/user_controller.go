package controller

import (
	"context"

	"spine-user-demo/dto"
	"spine-user-demo/service"

	"github.com/NARUBROWN/spine/pkg/httperr"
	"github.com/NARUBROWN/spine/pkg/query"
)

type UserController struct {
	userService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService: userService}
}

// 읽기
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

// 생성
func (c *UserController) CreateUser(
	ctx context.Context,
	req dto.CreateUserRequest,
) (dto.CreateUserResponse, error) {
	return c.userService.Create(ctx, req.Name, req.Email)
}

// 수정
func (c *UserController) UpdateUser(
	ctx context.Context,
	q query.Values,
	req dto.UpdateUserRequest,
) (dto.CreateUserResponse, error) {
	userId := int(q.Int("id", 0))

	user, err := c.userService.Update(ctx, userId, req.Name)
	if err != nil {
		return dto.CreateUserResponse{}, httperr.NotFound("유저를 찾을 수 없습니다.")
	}

	return user, nil
}

// 삭제
func (c *UserController) DeleteUser(
	ctx context.Context,
	q query.Values,
) error {
	userId := int(q.Int("id", 0))
	return c.userService.Delete(ctx, userId)
}
