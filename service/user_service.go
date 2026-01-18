package service

import (
	"context"

	"spine-user-demo/dto"
	"spine-user-demo/entity"
	"spine-user-demo/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Get(ctx context.Context, id int) (dto.CreateUserResponse, error) {
	result, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return dto.CreateUserResponse{}, err
	}
	return dto.CreateUserResponse{
		ID:    int(result.ID),
		Name:  result.Name,
		Email: result.Email,
	}, err
}

func (s *UserService) Create(ctx context.Context, name string, email string) (dto.CreateUserResponse, error) {
	user := &entity.User{Name: name, Email: email}
	err := s.repo.Save(ctx, user)
	if err != nil {
		return dto.CreateUserResponse{}, err
	}
	return dto.CreateUserResponse{
		ID:    int(user.ID),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *UserService) Update(ctx context.Context, id int, name string) (dto.CreateUserResponse, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return dto.CreateUserResponse{}, err
	}

	user.Name = name
	if err := s.repo.Update(ctx, user); err != nil {
		return dto.CreateUserResponse{}, err
	}

	return dto.CreateUserResponse{
		ID:    int(user.ID),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *UserService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
