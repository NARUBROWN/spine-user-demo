package controller

import (
	"context"
	"errors"
	"github.com/NARUBROWN/spine"
	platform "github.com/NARUBROWN/spine-echo-platform"
	"github.com/NARUBROWN/spine/pkg/boot"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"spine-user-demo/src/dto"
	"spine-user-demo/src/service"

	"github.com/NARUBROWN/spine/pkg/httperr"
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
	assertControllerHTTP(t, controller, "GET", "/users", "", http.StatusBadRequest)
}

func TestGetUserMapsNotFound(t *testing.T) {
	controller := &UserController{userService: &fakeUserService{getErr: service.ErrUserNotFound}}
	assertControllerHTTP(t, controller, "GET", "/users?id=123", "", http.StatusNotFound)
}

func TestCreateUserValidatesBeforeTransactionLookup(t *testing.T) {
	controller := &UserController{userService: &fakeUserService{}}
	assertControllerHTTP(t, controller, "POST", "/users", `{}`, http.StatusBadRequest)
}

func TestMapServiceErrorMapsDuplicateEmail(t *testing.T) {
	assertHTTPStatus(t, mapServiceError(service.ErrEmailAlreadyUsed), http.StatusConflict)
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

func idPtr(id int64) *int64 { return &id }

func assertControllerHTTP(t *testing.T, controller *UserController, method, path, body string, want int) {
	t.Helper()
	app := platform.New()
	app.Constructor(func() *UserController { return controller })
	prepared, err := app.Prepare(t.Context(), boot.Options{HTTP: &boot.HTTPOptions{}})
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Close(context.Background())
	handler, err := spine.NetHTTPHandler(prepared)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != want {
		t.Fatalf("status=%d want=%d body=%s", response.Code, want, response.Body.String())
	}
}
