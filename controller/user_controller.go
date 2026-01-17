package controller

import (
	"spine-user-demo/repository"
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
func (c *UserController) GetUser(q query.Values) (repository.User, error) {
	userId := int(q.Int("id", 0))
	if user, ok := c.userService.Get(userId); ok {
		return user, nil
	}
	return repository.User{}, httperr.NotFound("유저를 찾을 수 없습니다.")
}

// 생성
func (c *UserController) CreateUser(req CreateUserRequest) any {
	return c.userService.Create(req.ID, req.Name)
}

// 수정
func (c *UserController) UpdateUser(q query.Values, req UpdateUserRequest) (repository.User, error) {
	userId := int(q.Int("id", 0))
	if user, ok := c.userService.Update(userId, req.Name); ok {
		return user, nil
	}
	return repository.User{}, httperr.NotFound("유저를 찾을 수 없습니다.")
}

// 삭제
func (c *UserController) DeleteUser(q query.Values) any {
	userId := int(q.Int("id", 0))
	c.userService.Delete(userId)
	return map[string]string{"status": "삭제됨"}
}
