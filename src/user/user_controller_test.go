package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"spine-user-demo/src/user/dto"

	"github.com/NARUBROWN/spine/pkg/httperr"
	"github.com/NARUBROWN/spine/pkg/query"
	"github.com/uptrace/bun"
)

type fakeUserService struct {
	getErr error
}

func (s *fakeUserService) Get(context.Context, int) (dto.CreateUserResponse, error) {
	return dto.CreateUserResponse{}, s.getErr
}

func (*fakeUserService) Create(context.Context, bun.IDB, string, string) (dto.CreateUserResponse, error) {
	return dto.CreateUserResponse{}, nil
}

func (*fakeUserService) Update(context.Context, bun.IDB, int, string) (dto.CreateUserResponse, error) {
	return dto.CreateUserResponse{}, nil
}

func (*fakeUserService) Delete(context.Context, bun.IDB, int) error { return nil }

type emptyCtx struct{}

func (emptyCtx) Get(string) (any, bool) { return nil, false }

func TestGetUserRejectsInvalidID(t *testing.T) {
	controller := &UserController{userService: &fakeUserService{}}
	_, err := controller.GetUser(context.Background(), query.Values{})
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestGetUserMapsNotFound(t *testing.T) {
	controller := &UserController{userService: &fakeUserService{getErr: ErrUserNotFound}}
	_, err := controller.GetUser(context.Background(), query.NewValues(map[string][]string{"id": {"123"}}))
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestCreateUserValidatesBeforeTransactionLookup(t *testing.T) {
	controller := &UserController{userService: &fakeUserService{}}
	_, err := controller.CreateUser(context.Background(), &dto.CreateUserRequest{}, emptyCtx{})
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestMapServiceErrorMapsDuplicateEmail(t *testing.T) {
	assertHTTPStatus(t, mapServiceError(ErrEmailAlreadyUsed), http.StatusConflict)
}

func assertHTTPStatus(t *testing.T, err error, want int) {
	t.Helper()
	var httpErr *httperr.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error = %T %v, want *httperr.HTTPError", err, err)
	}
	if httpErr.Status != want {
		t.Fatalf("status = %d, want %d", httpErr.Status, want)
	}
}
