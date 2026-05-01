package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type adminCompatRequestLogStore struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]*adminCompatRequestLog
}

type adminCompatRequestLog struct {
	ID                  int64  `json:"id"`
	RequestID           string `json:"request_id"`
	Model               string `json:"model"`
	Channel             string `json:"channel"`
	StatusCode          int    `json:"status_code"`
	LatencyMS           int64  `json:"latency_ms"`
	PromptTokens        int    `json:"prompt_tokens"`
	CompletionTokens    int    `json:"completion_tokens"`
	TotalTokens         int    `json:"total_tokens"`
	CachedTokens        int    `json:"cached_tokens"`
	CacheCreationTokens int    `json:"cache_creation_tokens"`
	CacheHit            bool   `json:"cache_hit"`
	RequestBody         string `json:"request_body"`
	ResponseBody        string `json:"response_body"`
	Details             string `json:"details"`
	CreatedAt           string `json:"created_at"`
}

type adminCompatRequestLogInput struct {
	Model               string
	Channel             string
	StatusCode          int
	LatencyMS           int64
	PromptTokens        int
	CompletionTokens    int
	TotalTokens         int
	CachedTokens        int
	CacheCreationTokens int
	CacheHit            bool
	RequestBody         string
	ResponseBody        string
	Details             map[string]any
}

type requestLogSnapshot struct {
	items  []*adminCompatRequestLog
	nextID int64
}

var compatRequestLogs = newAdminCompatRequestLogStore()

func newAdminCompatRequestLogStore() *adminCompatRequestLogStore {
	return &adminCompatRequestLogStore{nextID: 1, items: make(map[int64]*adminCompatRequestLog)}
}

func (s *adminCompatRequestLogStore) append(input adminCompatRequestLogInput) (*adminCompatRequestLog, error) {
	detailsJSON, err := json.Marshal(input.Details)
	if err != nil {
		return nil, fmt.Errorf("序列化请求日志详情失败: %w", err)
	}
	requestID, err := generateCompatRequestID()
	if err != nil {
		return nil, fmt.Errorf("生成请求日志 ID 失败: %w", err)
	}
	s.mu.Lock()
	item := &adminCompatRequestLog{
		ID:                  s.nextID,
		RequestID:           requestID,
		Model:               strings.TrimSpace(input.Model),
		Channel:             strings.TrimSpace(input.Channel),
		StatusCode:          input.StatusCode,
		LatencyMS:           input.LatencyMS,
		PromptTokens:        input.PromptTokens,
		CompletionTokens:    input.CompletionTokens,
		TotalTokens:         input.TotalTokens,
		CachedTokens:        input.CachedTokens,
		CacheCreationTokens: input.CacheCreationTokens,
		CacheHit:            input.CacheHit,
		RequestBody:         truncateLogBlob(input.RequestBody),
		ResponseBody:        truncateLogBlob(input.ResponseBody),
		Details:             string(detailsJSON),
		CreatedAt:           time.Now().UTC().Format(time.RFC3339),
	}
	s.items[item.ID] = item
	s.nextID++
	s.mu.Unlock()
	if err := persistCompatState(); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *adminCompatRequestLogStore) list(query string, category string) ([]map[string]any, int, int) {
	s.mu.RLock()
	items := make([]*adminCompatRequestLog, 0, len(s.items))
	for _, item := range s.items {
		clone := *item
		items = append(items, &clone)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })
	total := len(items)
	filteredItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if !matchesLogQuery(item, query) || !matchesLogCategoryCompat(item, category) {
			continue
		}
		filteredItems = append(filteredItems, requestLogToSummaryAPI(item))
	}
	return filteredItems, total, len(filteredItems)
}

func (s *adminCompatRequestLogStore) detail(id int64) (*adminCompatRequestLog, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return nil, false
	}
	clone := *item
	return &clone, true
}

func (s *adminCompatRequestLogStore) clear() error {
	s.mu.Lock()
	s.items = make(map[int64]*adminCompatRequestLog)
	s.nextID = 1
	s.mu.Unlock()
	return persistCompatState()
}

func (s *adminCompatRequestLogStore) snapshot() requestLogSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*adminCompatRequestLog, 0, len(s.items))
	for _, item := range s.items {
		clone := *item
		items = append(items, &clone)
	}
	return requestLogSnapshot{items: items, nextID: s.nextID}
}

func (s *adminCompatRequestLogStore) restore(items []*adminCompatRequestLog, nextID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[int64]*adminCompatRequestLog)
	maxID := int64(0)
	for _, item := range items {
		clone := *item
		s.items[item.ID] = &clone
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	if nextID <= maxID {
		nextID = maxID + 1
	}
	if nextID <= 0 {
		nextID = 1
	}
	s.nextID = nextID
}

func requestLogToSummaryAPI(item *adminCompatRequestLog) map[string]any {
	return map[string]any{
		"id":                    item.ID,
		"request_id":            item.RequestID,
		"model":                 item.Model,
		"channel":               item.Channel,
		"status_code":           item.StatusCode,
		"latency_ms":            item.LatencyMS,
		"prompt_tokens":         item.PromptTokens,
		"completion_tokens":     item.CompletionTokens,
		"total_tokens":          item.TotalTokens,
		"cached_tokens":         item.CachedTokens,
		"cache_creation_tokens": item.CacheCreationTokens,
		"cache_hit":             item.CacheHit,
		"details":               item.Details,
		"created_at":            item.CreatedAt,
	}
}

func requestLogToDetailAPI(item *adminCompatRequestLog) map[string]any {
	result := requestLogToSummaryAPI(item)
	result["request_body"] = item.RequestBody
	result["response_body"] = item.ResponseBody
	return result
}

func generateCompatRequestID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "req_" + hex.EncodeToString(buf), nil
}

func truncateLogBlob(value string) string {
	const max = 32768
	if len(value) <= max {
		return value
	}
	return value[:max] + "\n...[truncated]"
}

func matchesLogQuery(item *adminCompatRequestLog, query string) bool {
	normalized := strings.TrimSpace(strings.ToLower(query))
	if normalized == "" {
		return true
	}
	haystacks := []string{item.RequestID, item.Model, item.Channel, item.Details, strconv.Itoa(item.StatusCode)}
	for _, value := range haystacks {
		if strings.Contains(strings.ToLower(value), normalized) {
			return true
		}
	}
	return false
}

func matchesLogCategoryCompat(item *adminCompatRequestLog, category string) bool {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "failed":
		return item.StatusCode >= 400
	case "success":
		return item.StatusCode >= 200 && item.StatusCode < 400
	case "cached":
		return item.CacheHit
	case "bridged":
		var details map[string]any
		if err := json.Unmarshal([]byte(item.Details), &details); err == nil {
			provider, _ := details["provider"].(string)
			requestPath, _ := details["request_path"].(string)
			if requestPath == "/v1/chat/completions" && strings.EqualFold(provider, "openai") {
				return false
			}
			if requestPath == "/v1/messages" && strings.EqualFold(provider, "claude") {
				return false
			}
			if strings.HasPrefix(requestPath, "/v1beta/models/") && strings.EqualFold(provider, "gemini") {
				return false
			}
		}
		return true
	default:
		return true
	}
}
