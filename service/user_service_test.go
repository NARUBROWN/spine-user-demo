package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"spine-user-demo/entity"

	"github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
)

type fakeUserRepository struct {
	findResult *entity.User
	findErr    error
	saveErr    error
	deleteErr  error
}

func (r *fakeUserRepository) FindByID(context.Context, int) (*entity.User, error) {
	return r.findResult, r.findErr
}

func (r *fakeUserRepository) FindByIDWith(context.Context, bun.IDB, int) (*entity.User, error) {
	return r.findResult, r.findErr
}

func (r *fakeUserRepository) Save(context.Context, bun.IDB, *entity.User) error {
	return r.saveErr
}

func (*fakeUserRepository) Update(context.Context, bun.IDB, *entity.User) error { return nil }

func (r *fakeUserRepository) Delete(context.Context, bun.IDB, int) error {
	return r.deleteErr
}

func TestUserServiceMapsNotFound(t *testing.T) {
	service := &UserService{repo: &fakeUserRepository{findErr: sql.ErrNoRows, deleteErr: sql.ErrNoRows}}
	if _, err := service.Get(context.Background(), 1); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Get() error = %v, want ErrUserNotFound", err)
	}
	if err := service.Delete(context.Background(), nil, 1); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Delete() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserServiceMapsDuplicateEmail(t *testing.T) {
	service := &UserService{repo: &fakeUserRepository{
		saveErr: &mysql.MySQLError{Number: 1062, Message: "duplicate entry"},
	}}
	_, err := service.Create(context.Background(), nil, "Alice", "alice@example.com")
	if !errors.Is(err, ErrEmailAlreadyUsed) {
		t.Fatalf("Create() error = %v, want ErrEmailAlreadyUsed", err)
	}
}
