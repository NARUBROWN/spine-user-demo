package dto

import "testing"

func TestCreateUserRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request *CreateUserRequest
		wantErr bool
	}{
		{name: "valid", request: &CreateUserRequest{Name: " Alice ", Email: "alice@example.com"}},
		{name: "missing name", request: &CreateUserRequest{Email: "alice@example.com"}, wantErr: true},
		{name: "invalid email", request: &CreateUserRequest{Name: "Alice", Email: "Alice <alice@example.com>"}, wantErr: true},
		{name: "nil body", request: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateUserRequestValidateTrimsName(t *testing.T) {
	request := &UpdateUserRequest{Name: " Alice "}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	if request.Name != "Alice" {
		t.Fatalf("Name = %q, want Alice", request.Name)
	}
}
