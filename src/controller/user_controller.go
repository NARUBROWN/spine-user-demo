package controller

import (
	"context"
	"errors"
	"net/http"
	"os"

	"spine-user-demo/internal/interceptor"

	"spine-user-demo/src/dto"
	"spine-user-demo/src/service"

	"github.com/NARUBROWN/spine/pkg/api"
	"github.com/NARUBROWN/spine/pkg/httperr"
	"github.com/NARUBROWN/spine/pkg/httpx"
	"github.com/NARUBROWN/spine/pkg/input"
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

func (*UserController) Constructor(userService *service.UserService) *UserController {
	return &UserController{userService: userService}
}

// RegisterRoutes는 경로와 요청 처리 코드를 함께 등록합니다.
func (c *UserController) RegisterRoutes(r *api.Registry) {
	users := r.Group("/users", api.Tags("Users"))
	users.GET("/", func(
		ctx spine.Ctx,
		q input.Query[dto.UserQuery],
	) (httpx.Response[dto.CreateUserResponse], error) {
		userId := (*q).UserID()
		if userId <= 0 {
			return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest("id는 1 이상의 정수여야 합니다.")
		}

		user, err := c.userService.Get(ctx.ExecutionContext().Context(), userId)
		if err != nil {
			return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
		}

		return httpx.Response[dto.CreateUserResponse]{
			Body: user,
		}, nil
	}, api.Summary("사용자 조회"))
	users.POST("/", func(
		ctx spine.Ctx,
		req input.Body[dto.CreateUserRequest],
	) (httpx.Response[dto.CreateUserResponse], error) {
		if err := (*req).Validate(); err != nil {
			return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest(err.Error())
		}
		tx, err := transactionFrom(ctx)
		if err != nil {
			return httpx.Response[dto.CreateUserResponse]{}, err
		}
		user, err := c.userService.Create(ctx.ExecutionContext().Context(), tx, req.Name, req.Email)
		if err != nil {
			return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
		}
		return httpx.Response[dto.CreateUserResponse]{
			Body: user,
		}, nil
	}, api.Summary("사용자 생성"))
	users.PUT("/", func(
		ctx spine.Ctx,
		q input.Query[dto.UserQuery],
		req input.Body[dto.UpdateUserRequest],
	) (httpx.Response[dto.CreateUserResponse], error) {
		if err := (*req).Validate(); err != nil {
			return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest(err.Error())
		}
		userId := (*q).UserID()
		if userId <= 0 {
			return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest("id는 1 이상의 정수여야 합니다.")
		}
		tx, err := transactionFrom(ctx)
		if err != nil {
			return httpx.Response[dto.CreateUserResponse]{}, err
		}

		user, err := c.userService.Update(ctx.ExecutionContext().Context(), tx, userId, req.Name)
		if err != nil {
			return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
		}

		return httpx.Response[dto.CreateUserResponse]{
			Body: user,
		}, nil
	}, api.Summary("사용자 수정"))
	users.DELETE("/", func(
		ctx spine.Ctx,
		q input.Query[dto.UserQuery],
	) error {
		userId := (*q).UserID()
		if userId <= 0 {
			return httperr.BadRequest("id는 1 이상의 정수여야 합니다.")
		}
		tx, err := transactionFrom(ctx)
		if err != nil {
			return err
		}
		if err := c.userService.Delete(ctx.ExecutionContext().Context(), tx, userId); err != nil {
			return mapServiceError(err)
		}
		return nil
	}, api.Summary("사용자 삭제"))
	if os.Getenv("SPINE_DEMO_ENABLE_FAILURE_SCENARIOS") == "1" {
		r.POST("/scenarios/transactions/controller-error", func(
			ctx spine.Ctx,
			req input.Body[dto.CreateUserRequest],
		) error {
			if err := (*req).Validate(); err != nil {
				return httperr.BadRequest(err.Error())
			}
			tx, err := transactionFrom(ctx)
			if err != nil {
				return err
			}
			if _, err := c.userService.Create(ctx.ExecutionContext().Context(), tx, req.Name, req.Email); err != nil {
				return mapServiceError(err)
			}
			return httperr.BadRequest("의도적으로 컨트롤러 오류를 발생시켰습니다.")
		})
		r.POST("/scenarios/transactions/response-error", func(
			ctx spine.Ctx,
			req input.Body[dto.CreateUserRequest],
		) (httpx.Response[dto.CreateUserResponse], error) {
			if err := (*req).Validate(); err != nil {
				return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest(err.Error())
			}
			tx, err := transactionFrom(ctx)
			if err != nil {
				return httpx.Response[dto.CreateUserResponse]{}, err
			}
			user, err := c.userService.Create(ctx.ExecutionContext().Context(), tx, req.Name, req.Email)
			if err != nil {
				return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
			}
			return httpx.Response[dto.CreateUserResponse]{
				Body: user,
				Options: httpx.ResponseOptions{
					Cookies: []httpx.Cookie{{Name: "invalid\r\ncookie", Value: "value"}},
				},
			}, nil
		})
	}
	if os.Getenv("SPINE_DEMO_ENABLE_SECURITY_SCENARIOS") == "1" {
		r.POST("/scenarios/security/users", func(
			ctx spine.Ctx,
			req input.Body[dto.CreateUserRequest],
		) (httpx.Response[dto.CreateUserResponse], error) {
			if err := (*req).Validate(); err != nil {
				return httpx.Response[dto.CreateUserResponse]{}, httperr.BadRequest(err.Error())
			}
			tx, err := transactionFrom(ctx)
			if err != nil {
				return httpx.Response[dto.CreateUserResponse]{}, err
			}
			user, err := c.userService.Create(ctx.ExecutionContext().Context(), tx, req.Name, req.Email)
			if err != nil {
				return httpx.Response[dto.CreateUserResponse]{}, mapServiceError(err)
			}
			return httpx.Response[dto.CreateUserResponse]{
				Body: user,
			}, nil
		}, api.Interceptors((*interceptor.SecurityTokenInterceptor)(nil)))
	}
}

func transactionFrom(spineCtx spine.Ctx) (bun.IDB, error) {
	v, exists := spineCtx.ExecutionContext().Get("tx")
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
