//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestRuntimeOpenAPIIncludesValidatedDTOs(t *testing.T) {
	baseURL := requiredEnv(t, "SPINE_DEMO_E2E_BASE_URL")
	status, body := requestJSON(t, http.MethodGet, baseURL+"/openapi.json", "")
	assertStatus(t, status, http.StatusOK)
	var document struct {
		Paths map[string]map[string]struct {
			Tags       []string
			Summary    string
			Parameters []struct {
				Name, In string
				Required bool
				Schema   map[string]any
			}
			RequestBody struct {
				Required bool
				Content  map[string]struct{ Schema map[string]any }
			}
			Responses map[string]struct {
				Content map[string]struct{ Schema map[string]any }
			}
		}
		Components  struct{ Schemas map[string]map[string]any }
		Diagnostics []struct{ Code, Target string } `json:"x-spine-diagnostics"`
	}
	if err := json.Unmarshal([]byte(body), &document); err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Code != "RESPONSE_STATUS_DYNAMIC" && !(diagnostic.Code == "FIELD_EXAMPLE_VALIDATION_UNRESOLVED" && (diagnostic.Target == "dto.CreateUserRequest.Email" || diagnostic.Target == "dto.CreateUserRequest.email")) {
			t.Fatalf("unexpected schema diagnostic: %+v", diagnostic)
		}
	}
	for _, method := range []string{"get", "post", "put", "delete"} {
		op := document.Paths["/users"][method]
		if !slices.Contains(op.Tags, "Users") || op.Summary == "" {
			t.Fatalf("%s documentation metadata missing: %+v", method, op)
		}
	}
	if !strings.Contains(body, `"maximum":9223372036854775807`) || !strings.Contains(body, `"minimum":-9223372036854775808`) {
		t.Fatal("exact integer bounds missing")
	}
	if !strings.Contains(body, `"example":"user@example.com"`) {
		t.Fatal("DTO example missing")
	}
	for _, method := range []string{"get", "put", "delete"} {
		parameters := document.Paths["/users"][method].Parameters
		if len(parameters) != 1 || parameters[0].Name != "id" || parameters[0].In != "query" || !parameters[0].Required {
			t.Fatalf("%s query: %+v", method, parameters)
		}
		requireSchemaConstraint(t, parameters[0].Schema, "exclusiveMinimum", float64(0))
	}
	for _, method := range []string{"post", "put"} {
		op := document.Paths["/users"][method]
		request := op.RequestBody
		if !request.Required {
			t.Fatalf("%s request body is not required", method)
		}
		schema := document.Components.Schemas[schemaReference(request.Content["application/json"].Schema)]
		properties, _ := schema["properties"].(map[string]any)
		name, _ := properties["name"].(map[string]any)
		requireSchemaConstraint(t, name, "minLength", float64(1))
		requireSchemaConstraint(t, name, "maxLength", float64(100))
		required, _ := schema["required"].([]any)
		if !slices.Contains(required, any("name")) {
			t.Fatalf("%s name is not required: %+v", method, schema)
		}
		if method == "post" {
			email, _ := properties["email"].(map[string]any)
			requireSchemaConstraint(t, email, "format", "email")
			requireSchemaConstraint(t, email, "maxLength", float64(255))
			if !slices.Contains(required, any("email")) {
				t.Fatal("create email is not required")
			}
		}
		failure := op.Responses["400"].Content["application/json"].Schema
		failureProperties, _ := failure["properties"].(map[string]any)
		if failureProperties["fields"] == nil {
			t.Fatalf("%s validation response lacks field details", method)
		}
	}
	for _, method := range []string{"get", "post", "put"} {
		response := document.Paths["/users"][method].Responses["default"].Content["application/json"].Schema
		alternatives, _ := response["anyOf"].([]any)
		if len(alternatives) == 0 {
			t.Fatalf("%s response schema missing", method)
		}
		ref, _ := alternatives[0].(map[string]any)
		schema := document.Components.Schemas[schemaReference(ref)]
		properties, _ := schema["properties"].(map[string]any)
		for _, field := range []string{"id", "name", "email"} {
			if properties[field] == nil {
				t.Fatalf("%s response lacks %s", method, field)
			}
		}
	}
}

func schemaReference(schema map[string]any) string {
	ref, _ := schema["$ref"].(string)
	return strings.TrimPrefix(ref, "#/components/schemas/")
}

func requireSchemaConstraint(t *testing.T, schema map[string]any, key string, want any) {
	t.Helper()
	var contains func(map[string]any) bool
	contains = func(current map[string]any) bool {
		if reflect.DeepEqual(current[key], want) {
			return true
		}
		clauses, _ := current["allOf"].([]any)
		for _, clause := range clauses {
			if nested, ok := clause.(map[string]any); ok && contains(nested) {
				return true
			}
		}
		return false
	}
	if !contains(schema) {
		t.Fatalf("schema lacks %s=%v: %+v", key, want, schema)
	}
}
