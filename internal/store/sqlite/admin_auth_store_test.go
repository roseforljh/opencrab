package sqlite

import (
	"context"
	"errors"
	"testing"

	"opencrab/internal/domain"
)

func TestSetupAdminPasswordAndVerify(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	state, err := SetupAdminPassword(context.Background(), db, "hunter2-password")
	if err != nil {
		t.Fatalf("setup admin password: %v", err)
	}
	if !state.Initialized || state.SessionSecret == "" || state.PasswordHash == "" {
		t.Fatalf("unexpected admin auth state: %#v", state)
	}

	loaded, err := GetAdminAuthState(context.Background(), db)
	if err != nil {
		t.Fatalf("get admin auth state: %v", err)
	}
	if !loaded.Initialized || loaded.InitializedAt == "" {
		t.Fatalf("unexpected loaded auth state: %#v", loaded)
	}

	verified, err := VerifyAdminPassword(context.Background(), db, "hunter2-password")
	if err != nil {
		t.Fatalf("verify admin password: %v", err)
	}
	if verified.SessionSecret == "" {
		t.Fatal("expected session secret after verify")
	}

	if _, err := VerifyAdminPassword(context.Background(), db, "wrong-password"); !errors.Is(err, ErrInvalidAdminPassword) {
		t.Fatalf("expected invalid password error, got %v", err)
	}
	if _, err := SetupAdminPassword(context.Background(), db, "another-password"); !errors.Is(err, ErrAdminPasswordAlreadyInitialized) {
		t.Fatalf("expected already initialized error, got %v", err)
	}
}

func TestChangeAdminPasswordAndSecondaryPasswordLifecycle(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := SetupAdminPassword(context.Background(), db, "origin-main-password"); err != nil {
		t.Fatalf("setup admin password: %v", err)
	}

	updated, err := ChangeAdminPassword(context.Background(), db, domain.AdminPasswordChangeInput{
		CurrentPassword: "origin-main-password",
		NewPassword:     "next-main-password",
		ConfirmPassword: "next-main-password",
	})
	if err != nil {
		t.Fatalf("change admin password: %v", err)
	}
	if updated.SessionSecret == "" {
		t.Fatal("expected rotated session secret")
	}
	if _, err := VerifyAdminPassword(context.Background(), db, "origin-main-password"); !errors.Is(err, ErrInvalidAdminPassword) {
		t.Fatalf("expected old password invalid, got %v", err)
	}

	secondary, err := UpdateAdminSecondaryPassword(context.Background(), db, domain.AdminSecondaryPasswordUpdateInput{
		Enabled:              true,
		CurrentAdminPassword: "next-main-password",
		NewPassword:          "secondary-password",
		ConfirmPassword:      "secondary-password",
	})
	if err != nil {
		t.Fatalf("enable secondary password: %v", err)
	}
	if !secondary.Enabled || !secondary.Configured {
		t.Fatalf("unexpected secondary state: %#v", secondary)
	}
	if err := VerifySecondaryPassword(context.Background(), db, "secondary-password"); err != nil {
		t.Fatalf("verify secondary password: %v", err)
	}

	secondary, err = UpdateAdminSecondaryPassword(context.Background(), db, domain.AdminSecondaryPasswordUpdateInput{
		Enabled:                  true,
		CurrentAdminPassword:     "next-main-password",
		CurrentSecondaryPassword: "secondary-password",
		NewPassword:              "secondary-password-next",
		ConfirmPassword:          "secondary-password-next",
	})
	if err != nil {
		t.Fatalf("change secondary password: %v", err)
	}
	if err := VerifySecondaryPassword(context.Background(), db, "secondary-password-next"); err != nil {
		t.Fatalf("verify changed secondary password: %v", err)
	}

	secondary, err = UpdateAdminSecondaryPassword(context.Background(), db, domain.AdminSecondaryPasswordUpdateInput{
		Enabled:              false,
		CurrentAdminPassword: "next-main-password",
	})
	if err != nil {
		t.Fatalf("disable secondary password: %v", err)
	}
	if secondary.Enabled {
		t.Fatalf("expected secondary disabled, got %#v", secondary)
	}
	if err := VerifySecondaryPassword(context.Background(), db, ""); err != nil {
		t.Fatalf("expected disabled secondary password skipped, got %v", err)
	}
}
