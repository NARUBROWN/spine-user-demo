package application

import "testing"

func TestSecurityScenarioRequiresToken(t *testing.T) {
	t.Setenv("SPINE_DEMO_ENABLE_SECURITY_SCENARIOS", "1")
	t.Setenv("SPINE_DEMO_SECURITY_TOKEN", "")
	if _, _, err := BuildApplication(); err == nil {
		t.Fatal("security scenario must reject an empty token")
	}
}

func TestBuildApplicationWithoutDatabase(t *testing.T) {
	t.Setenv("SPINE_DEMO_MYSQL_DSN", "")
	app, options, err := BuildApplication()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Validate(options); err != nil {
		t.Fatalf("offline application validation: %v", err)
	}
}

func TestApplicationAddressPrecedence(t *testing.T) {
	t.Setenv("SPINE_PORT", "18084")
	t.Setenv("SPINE_DEMO_ADDRESS", "")
	_, options, err := BuildApplication()
	if err != nil || options.Address != ":18084" {
		t.Fatalf("CLI address = %q, error = %v", options.Address, err)
	}
	t.Setenv("SPINE_DEMO_ADDRESS", "127.0.0.1:19084")
	_, options, err = BuildApplication()
	if err != nil || options.Address != "127.0.0.1:19084" {
		t.Fatalf("explicit address = %q, error = %v", options.Address, err)
	}
}

func TestBuildApplicationConfiguresNativeOpenAPI(t *testing.T) {
	t.Setenv("SPINE_DEMO_MYSQL_DSN", "")
	app, options, err := BuildApplication()
	if err != nil {
		t.Fatal(err)
	}
	if options.HTTP == nil || !options.HTTP.OpenAPI.Enabled || options.HTTP.OpenAPI.Path != "/openapi.json" || len(options.HTTP.OpenAPI.UI) != 1 {
		t.Fatalf("OpenAPI options: %+v", options.HTTP)
	}
	if err = app.Validate(options); err != nil {
		t.Fatal(err)
	}
}
