package controller

import (
	"log"
	"spine-user-demo/service"

	"github.com/NARUBROWN/spine/core"
)

type UserController struct {
	userService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService: userService}
}

// 읽기
func (c *UserController) GetUser(ctx core.Context, id int, name string, q UserQuery) any {
	log.Printf("PathVariable과 QueryParam를 같이 쓸 수 있어요, PathVariable (id): %d, QueryParam: %d", id, q.ID)
	if user, ok := c.userService.Get(q.ID); ok {
		return user
	}
	return map[string]string{"error": "유저를 찾을 수 없습니다."}
}

// 생성
func (c *UserController) CreateUser(ctx core.Context, req CreateUserRequest) any {
	return c.userService.Create(req.ID, req.Name)
}

// 수정
func (c *UserController) UpdateUser(ctx core.Context, q UserQuery, req UpdateUserRequest) any {
	if user, ok := c.userService.Update(q.ID, req.Name); ok {
		return user
	}
	return map[string]string{"error": "유저를 찾을 수 없습니다."}
}

// 삭제
func (c *UserController) DeleteUser(ctx core.Context, q UserQuery) any {
	c.userService.Delete(q.ID)
	return map[string]string{"status": "삭제됨"}
}
