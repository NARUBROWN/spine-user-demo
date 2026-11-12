package config

import (
	"errors"
	"testing"

	"github.com/NARUBROWN/spine/pkg/boot"
)

func TestValidateEnvironment(t *testing.T) {
	for _, tt := range []struct {
		name, enabled, token string
		wantError            bool
	}{
		{name: "disabled without token"},
		{name: "enabled without token", enabled: "1", wantError: true},
		{name: "enabled with token", enabled: "1", token: "test-token"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SPINE_DEMO_ENABLE_SECURITY_SCENARIOS", tt.enabled)
			t.Setenv("SPINE_DEMO_SECURITY_TOKEN", tt.token)
			t.Setenv("SPINE_DEMO_ENABLE_FAILURE_SCENARIOS", "")
			err := ValidateEnvironment()
			if tt.wantError {
				var configurationError *boot.ConfigError
				if !errors.As(err, &configurationError) || len(configurationError.Issues) != 1 || configurationError.Issues[0].Code != "REQUIRED" {
					t.Fatalf("expected a required configuration issue, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
