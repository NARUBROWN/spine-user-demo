package config

import (
	"os"

	"github.com/NARUBROWN/spine/pkg/boot"
)

// ValidateEnvironment는 앱 조립 시 선택적 보안 시나리오의 필수 설정을 검사합니다.
func ValidateEnvironment() error {
	if os.Getenv("SPINE_DEMO_ENABLE_SECURITY_SCENARIOS") == "1" && os.Getenv("SPINE_DEMO_SECURITY_TOKEN") == "" {
		return &boot.ConfigError{Issues: []boot.ConfigIssue{{
			Path: "Environment.SPINE_DEMO_SECURITY_TOKEN", Code: "REQUIRED",
			Message: "SPINE_DEMO_SECURITY_TOKEN is required for security scenarios",
			Hint:    "Set SPINE_DEMO_SECURITY_TOKEN when enabling security scenarios",
		}}}
	}
	return nil
}
