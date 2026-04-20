package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"opencrab/internal/domain"
	"opencrab/internal/transport/httpserver"
)

type ResponseSessionStore struct {
	db *sql.DB
}

func NewResponseSessionStore(db *sql.DB) *ResponseSessionStore {
	return &ResponseSessionStore{db: db}
}

func (s *ResponseSessionStore) Get(responseID string) (httpserver.ResponseSession, bool) {
	var transcriptJSON string
	var inputItemsJSON string
	var responseJSON string
	var item httpserver.ResponseSession
	err := s.db.QueryRow(`SELECT response_id, session_id, model, transcript_json, input_items_json, response_json, updated_at FROM response_sessions WHERE response_id = ? LIMIT 1`, responseID).Scan(&item.ResponseID, &item.SessionID, &item.Model, &transcriptJSON, &inputItemsJSON, &responseJSON, &item.UpdatedAt)
	if err != nil {
		return httpserver.ResponseSession{}, false
	}
	if err := json.Unmarshal([]byte(transcriptJSON), &item.Messages); err != nil {
		return httpserver.ResponseSession{}, false
	}
	if inputItemsJSON != "" {
		item.InputItems = json.RawMessage(inputItemsJSON)
	}
	if responseJSON != "" {
		item.ResponseBody = json.RawMessage(responseJSON)
	}
	return item, true
}

func (s *ResponseSessionStore) Put(session httpserver.ResponseSession) {
	transcriptJSON, err := json.Marshal(session.Messages)
	if err != nil {
		return
	}
	updatedAt := session.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	_, _ = s.db.Exec(`INSERT INTO response_sessions(response_id, session_id, model, transcript_json, input_items_json, response_json, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(response_id) DO UPDATE SET session_id = excluded.session_id, model = excluded.model, transcript_json = excluded.transcript_json, input_items_json = excluded.input_items_json, response_json = excluded.response_json, updated_at = excluded.updated_at`, session.ResponseID, session.SessionID, session.Model, string(transcriptJSON), string(session.InputItems), string(session.ResponseBody), updatedAt.Format(time.RFC3339))
}

func (s *ResponseSessionStore) Delete(responseID string) {
	_, _ = s.db.Exec(`DELETE FROM response_sessions WHERE response_id = ?`, responseID)
}

func (s *ResponseSessionStore) GetContext(ctx context.Context, responseID string) (httpserver.ResponseSession, bool, error) {
	var transcriptJSON string
	var inputItemsJSON string
	var responseJSON string
	var updatedAt string
	item := httpserver.ResponseSession{}
	err := s.db.QueryRowContext(ctx, `SELECT response_id, session_id, model, transcript_json, input_items_json, response_json, updated_at FROM response_sessions WHERE response_id = ? LIMIT 1`, responseID).Scan(&item.ResponseID, &item.SessionID, &item.Model, &transcriptJSON, &inputItemsJSON, &responseJSON, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return httpserver.ResponseSession{}, false, nil
		}
		return httpserver.ResponseSession{}, false, fmt.Errorf("读取 response session 失败: %w", err)
	}
	if err := json.Unmarshal([]byte(transcriptJSON), &item.Messages); err != nil {
		return httpserver.ResponseSession{}, false, fmt.Errorf("解析 response session transcript 失败: %w", err)
	}
	if inputItemsJSON != "" {
		item.InputItems = json.RawMessage(inputItemsJSON)
	}
	if responseJSON != "" {
		item.ResponseBody = json.RawMessage(responseJSON)
	}
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return item, true, nil
}

func (s *ResponseSessionStore) PutContext(ctx context.Context, session httpserver.ResponseSession) error {
	transcriptJSON, err := json.Marshal(session.Messages)
	if err != nil {
		return fmt.Errorf("序列化 response session transcript 失败: %w", err)
	}
	updatedAt := session.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO response_sessions(response_id, session_id, model, transcript_json, input_items_json, response_json, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(response_id) DO UPDATE SET session_id = excluded.session_id, model = excluded.model, transcript_json = excluded.transcript_json, input_items_json = excluded.input_items_json, response_json = excluded.response_json, updated_at = excluded.updated_at`, session.ResponseID, session.SessionID, session.Model, string(transcriptJSON), string(session.InputItems), string(session.ResponseBody), updatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("写入 response session 失败: %w", err)
	}
	return nil
}

func transcriptFromUnified(messages []domain.UnifiedMessage) []domain.GatewayMessage {
	result := make([]domain.GatewayMessage, 0, len(messages))
	for _, message := range messages {
		result = append(result, domain.GatewayMessage{Role: message.Role, Parts: message.Parts, ToolCalls: message.ToolCalls, InputItem: message.InputItem, Metadata: message.Metadata})
	}
	return result
}
