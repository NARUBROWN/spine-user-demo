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

// GetUser godoc
// @Summary Get user
// @Description Get a user by id
// @Tags users
// @Param id query int true "User ID"
// @Success 200 {object} dto.CreateUserResponse
// @Router /users [get]
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

// CreateUser godoc
// @Summary Create user
// @Description Create a new user
// @Tags users
// @Param body body dto.CreateUserRequest true "Create user request"
// @Success 200 {object} dto.CreateUserResponse
// @Router /users [post]
func (c *UserController) CreateUser(
	ctx context.Context,
	req dto.CreateUserRequest,
) (dto.CreateUserResponse, error) {
	return c.userService.Create(ctx, req.Name, req.Email)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user name
// @Tags users
// @Param id query int true "User ID"
// @Param body body dto.UpdateUserRequest true "Update user request"
// @Success 200 {object} dto.CreateUserResponse
// @Router /users [put]
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

// DeleteUser godoc
// @Summary Delete user
// @Description Delete a user by id
// @Tags users
// @Param id query int true "User ID"
// @Success 200
// @Router /users [delete]
func (c *UserController) DeleteUser(
	ctx context.Context,
	q query.Values,
) error {
	userId := int(q.Int("id", 0))
	return c.userService.Delete(ctx, userId)
}
