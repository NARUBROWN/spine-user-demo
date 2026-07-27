package service

import (
	"context"
	"database/sql"
	"errors"

	"spine-user-demo/dto"
	"spine-user-demo/entity"
	"spine-user-demo/repository"

	"github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrEmailAlreadyUsed = errors.New("email already used")
)

type userRepository interface {
	FindByID(context.Context, int) (*entity.User, error)
	FindByIDWith(context.Context, bun.IDB, int) (*entity.User, error)
	Save(context.Context, bun.IDB, *entity.User) error
	Update(context.Context, bun.IDB, *entity.User) error
	Delete(context.Context, bun.IDB, int) error
}

type UserService struct {
	repo userRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Get(ctx context.Context, id int) (dto.CreateUserResponse, error) {
	result, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.CreateUserResponse{}, ErrUserNotFound
		}
		return dto.CreateUserResponse{}, err
	}
	return dto.CreateUserResponse{
		ID:    int(result.ID),
		Name:  result.Name,
		Email: result.Email,
	}, err
}

func (s *UserService) Create(ctx context.Context, tx bun.IDB, name string, email string) (dto.CreateUserResponse, error) {
	user := &entity.User{Name: name, Email: email}
	err := s.repo.Save(ctx, tx, user)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return dto.CreateUserResponse{}, ErrEmailAlreadyUsed
		}
		return dto.CreateUserResponse{}, err
	}
	return dto.CreateUserResponse{
		ID:    int(user.ID),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *UserService) Update(ctx context.Context, tx bun.IDB, id int, name string) (dto.CreateUserResponse, error) {
	user, err := s.repo.FindByIDWith(ctx, tx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.CreateUserResponse{}, ErrUserNotFound
		}
		return dto.CreateUserResponse{}, err
	}

	user.Name = name
	if err := s.repo.Update(ctx, tx, user); err != nil {
		return dto.CreateUserResponse{}, err
	}

	return dto.CreateUserResponse{
		ID:    int(user.ID),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *UserService) Delete(ctx context.Context, tx bun.IDB, id int) error {
	err := s.repo.Delete(ctx, tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	return err
}
