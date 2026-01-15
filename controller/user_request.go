package controller

type CreateUserRequest struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UpdateUserRequest struct {
	Name string `json:"name"`
}
