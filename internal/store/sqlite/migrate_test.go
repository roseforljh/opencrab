package sqlite

import (
	"context"
	"testing"
)

func TestApplyMigrations(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
}
