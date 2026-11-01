package database

import "testing"

func TestValidateEnvironment(t *testing.T) {
	t.Setenv("SPINE_DEMO_MYSQL_DSN", "")
	if err := ValidateEnvironment(); err == nil {
		t.Fatal("missing DSN must fail validation")
	}

	t.Setenv("SPINE_DEMO_MYSQL_DSN", "not a DSN")
	if err := ValidateEnvironment(); err == nil {
		t.Fatal("invalid DSN must fail validation")
	}

	t.Setenv("SPINE_DEMO_MYSQL_DSN", "root:spine@tcp(127.0.0.1:3306)/spine_demo?parseTime=true")
	if err := ValidateEnvironment(); err != nil {
		t.Fatalf("valid DSN error = %v", err)
	}
}
