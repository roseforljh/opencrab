package sqlite

import (
	"context"
	"testing"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/transport/httpserver"
)

func TestResponseSessionStoreRoundTrip(t *testing.T) {
	db, err := Open(t.TempDir() + "/opencrab.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	store := NewResponseSessionStore(db)
	expected := httpserver.ResponseSession{ResponseID: "resp_1", SessionID: "sess_1", Model: "gpt-5.4", Messages: []domain.GatewayMessage{{Role: "user", Parts: []domain.UnifiedPart{{Type: "text", Text: "ping"}}}, {Role: "assistant", Parts: []domain.UnifiedPart{{Type: "text", Text: "pong"}}}}, UpdatedAt: time.Now()}
	if err := store.PutContext(context.Background(), expected); err != nil {
		t.Fatalf("put context: %v", err)
	}
	got, found, err := store.GetContext(context.Background(), "resp_1")
	if err != nil {
		t.Fatalf("get context: %v", err)
	}
	if !found || got.Model != expected.Model || len(got.Messages) != 2 || got.Messages[1].Parts[0].Text != "pong" {
		t.Fatalf("unexpected session: %#v", got)
	}
}
