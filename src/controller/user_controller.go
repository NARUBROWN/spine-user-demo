package controller

import (
	"context"
	"errors"
	"net/http"

	"spine-user-demo/src/dto"
	"spine-user-demo/src/service"

	"github.com/NARUBROWN/spine/pkg/httperr"
	"github.com/NARUBROWN/spine/pkg/httpx"
	"github.com/NARUBROWN/spine/pkg/spine"
	"github.com/uptrace/bun"
)

type UserController struct {
	userService userService
}

type userService interface {
	Get(context.Context, int) (dto.CreateUserResponse, error)
	Create(context.Context, bun.IDB, string, string) (dto.CreateUserResponse, error)
	Update(context.Context, bun.IDB, int, string) (dto.CreateUserResponse, error)
	Delete(context.Context, bun.IDB, int) error
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService: userService}
}

// GetUser는 ID로 사용자 정보를 조회합니다.
// @Summary Get user
// @Description Get a user by id
// @Tags users
// @Param id query int true "User ID"
// @Success 200 {object} dto.CreateUserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /users [get]
func (c *UserController) GetUser(
	ctx context.Context,
	q *dto.UserQuery,
) (httpx.Response[dto.CreateUserResponse], error) {
	userId := q.UserID()
	if userId <= 0 {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest("id는 1 이상의 정수여야 합니다.")
	}

	user, err := c.userService.Get(ctx, userId)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
	}

	return httpx.Response[dto.CreateUserResponse]{
		Body: user,
	}, nil
}

// CreateUser는 새 사용자를 생성합니다.
// @Summary Create user
// @Description Create a new user
// @Tags users
// @Param body body dto.CreateUserRequest true "Create user request"
// @Success 200 {object} dto.CreateUserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /users [post]
func (c *UserController) CreateUser(
	ctx context.Context,
	req *dto.CreateUserRequest,
	spineCtx spine.Ctx,
) (httpx.Response[dto.CreateUserResponse], error) {
	if err := req.Validate(); err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest(err.Error())
	}
	tx, err := transactionFrom(spineCtx)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, err
	}
	user, err := c.userService.Create(ctx, tx, req.Name, req.Email)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
	}
	return httpx.Response[dto.CreateUserResponse]{
		Body: user,
	}, nil
}

// UpdateUser는 기존 사용자의 이름을 수정합니다.
// @Summary Update user
// @Description Update user name
// @Tags users
// @Param id query int true "User ID"
// @Param body body dto.UpdateUserRequest true "Update user request"
// @Success 200 {object} dto.CreateUserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /users [put]
func (c *UserController) UpdateUser(
	ctx context.Context,
	q *dto.UserQuery,
	req *dto.UpdateUserRequest,
	spineCtx spine.Ctx,
) (httpx.Response[dto.CreateUserResponse], error) {
	if err := req.Validate(); err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest(err.Error())
	}
	userId := q.UserID()
	if userId <= 0 {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest("id는 1 이상의 정수여야 합니다.")
	}
	tx, err := transactionFrom(spineCtx)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, err
	}

	user, err := c.userService.Update(ctx, tx, userId, req.Name)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
	}

	return httpx.Response[dto.CreateUserResponse]{
		Body: user,
	}, nil
}

// DeleteUser는 ID로 사용자를 삭제합니다.
// @Summary Delete user
// @Description Delete a user by id
// @Tags users
// @Param id query int true "User ID"
// @Success 200
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /users [delete]
func (c *UserController) DeleteUser(
	ctx context.Context,
	q *dto.UserQuery,
	spineCtx spine.Ctx,
) error {
	userId := q.UserID()
	if userId <= 0 {
		return httperr.BadRequest("id는 1 이상의 정수여야 합니다.")
	}
	tx, err := transactionFrom(spineCtx)
	if err != nil {
		return err
	}
	if err := c.userService.Delete(ctx, tx, userId); err != nil {
		return mapServiceError(err)
	}
	return nil
}

// CreateThenFail은 컨트롤러 오류가 트랜잭션을 롤백하는지 확인하는 선택적 시나리오입니다.
func (c *UserController) CreateThenFail(
	ctx context.Context,
	req *dto.CreateUserRequest,
	spineCtx spine.Ctx,
) error {
	if err := req.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}
	tx, err := transactionFrom(spineCtx)
	if err != nil {
		return err
	}
	if _, err := c.userService.Create(ctx, tx, req.Name, req.Email); err != nil {
		return mapServiceError(err)
	}
	return httperr.BadRequest("의도적으로 컨트롤러 오류를 발생시켰습니다.")
}

// CreateWithInvalidCookie는 응답 준비 실패가 트랜잭션을 롤백하는지 확인하는 선택적 시나리오입니다.
func (c *UserController) CreateWithInvalidCookie(
	ctx context.Context,
	req *dto.CreateUserRequest,
	spineCtx spine.Ctx,
) (httpx.Response[dto.CreateUserResponse], error) {
	if err := req.Validate(); err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest(err.Error())
	}
	tx, err := transactionFrom(spineCtx)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, err
	}
	user, err := c.userService.Create(ctx, tx, req.Name, req.Email)
	if err != nil {
		return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
	}
	return httpx.Response[dto.CreateUserResponse]{
		Body: user,
		Options: httpx.ResponseOptions{
			Cookies: []httpx.Cookie{{Name: "invalid\r\ncookie", Value: "value"}},
		},
	}, nil
}

func transactionFrom(spineCtx spine.Ctx) (bun.IDB, error) {
	v, exists := spineCtx.Get("tx")
	if !exists {
		return nil, httperr.InternalServerError("트랜잭션이 존재하지 않습니다.")
	}
	tx, ok := v.(bun.IDB)
	if !ok {
		return nil, httperr.InternalServerError("트랜잭션 타입이 올바르지 않습니다.")
	}
	return tx, nil
}

func mapServiceError(err error) error {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		return httperr.NotFound("유저를 찾을 수 없습니다.")
	case errors.Is(err, service.ErrEmailAlreadyUsed):
		return &httperr.HTTPError{Status: http.StatusConflict, Message: "이미 사용 중인 이메일입니다."}
	default:
		return err
	}
}
