package config

import "testing"

func TestValidateAcceptsExpectedEnvironment(t *testing.T) {
	cfg := Config{
		App: AppConfig{
			Name:        "OpenCrab",
			Environment: "development",
		},
		DB: DBConfig{
			Path: "./data/opencrab.db",
		},
		HTTP: HTTPConfig{
			Address: ":8080",
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("expected config to be valid, got error: %v", err)
	}
}

func TestValidateRejectsUnsupportedEnvironment(t *testing.T) {
	cfg := Config{
		App: AppConfig{
			Name:        "OpenCrab",
			Environment: "staging",
		},
		DB: DBConfig{
			Path: "./data/opencrab.db",
		},
		HTTP: HTTPConfig{
			Address: ":8080",
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation to fail for unsupported environment")
	}
}

func TestValidateRejectsInvalidAddress(t *testing.T) {
	cfg := Config{
		App: AppConfig{
			Name:        "OpenCrab",
			Environment: "development",
		},
		DB: DBConfig{
			Path: "./data/opencrab.db",
		},
		HTTP: HTTPConfig{
			Address: "8080",
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation to fail for invalid address")
	}
}
