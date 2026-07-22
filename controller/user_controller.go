package controller

import (
	"context"

	"spine-user-demo/dto"
	"spine-user-demo/service"

	"github.com/NARUBROWN/spine/pkg/httperr"
	"github.com/NARUBROWN/spine/pkg/httpx"
	"github.com/NARUBROWN/spine/pkg/query"
	"github.com/NARUBROWN/spine/pkg/spine"
	"github.com/uptrace/bun"
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
) (httpx.Response[dto.CreateUserResponse], error) {
	userId := int(q.Int("id", 0))

	user, err := c.userService.Get(ctx, userId)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.NotFound("유저를 찾을 수 없습니다.")
	}

	return httpx.Response[dto.CreateUserResponse]{
		Body: user,
	}, nil
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
	req *dto.CreateUserRequest,
	spineCtx spine.Ctx,
) (httpx.Response[dto.CreateUserResponse], error) {
	tx := bun.IDB(nil)

	v, exists := spineCtx.Get("tx")
	if !exists {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.InternalServerError("트랜잭션이 존재하지 않습니다.")
	}

	tx, ok := v.(bun.IDB)
	if !ok {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.InternalServerError("트랜잭션 타입이 올바르지 않습니다.")
	}

	if req == nil {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest("요청 본문이 비어 있습니다.")
	}
	user, err := c.userService.Create(ctx, tx, req.Name, req.Email)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, err
	}
	return httpx.Response[dto.CreateUserResponse]{
		Body: user,
	}, nil
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
	req *dto.UpdateUserRequest,
	spineCtx spine.Ctx,
) (httpx.Response[dto.CreateUserResponse], error) {
	tx := bun.IDB(nil)

	v, exists := spineCtx.Get("tx")
	if !exists {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.InternalServerError("트랜잭션이 존재하지 않습니다.")
	}

	tx, ok := v.(bun.IDB)

	if !ok {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.InternalServerError("트랜잭션 타입이 올바르지 않습니다.")
	}
	if req == nil {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest("요청 본문이 비어 있습니다.")
	}
	userId := int(q.Int("id", 0))

	user, err := c.userService.Update(ctx, tx, userId, req.Name)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.NotFound("유저를 찾을 수 없습니다.")
	}

	return httpx.Response[dto.CreateUserResponse]{
		Body: user,
	}, nil
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
	spineCtx spine.Ctx,
) error {
	tx := bun.IDB(nil)

	v, exists := spineCtx.Get("tx")
	if !exists {
		return httperr.InternalServerError("트랜잭션이 존재하지 않습니다.")
	}

	tx, ok := v.(bun.IDB)
	if !ok {
		return httperr.InternalServerError("트랜잭션 타입이 올바르지 않습니다.")
	}

	userId := int(q.Int("id", 0))
	return c.userService.Delete(ctx, tx, userId)
}
