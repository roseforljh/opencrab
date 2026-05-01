package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type adminCompatAPIKeyStore struct {
	mu         sync.RWMutex
	nextID     int64
	items      map[int64]*adminCompatAPIKey
	nameIndex  map[string]int64
}

type adminCompatAPIKey struct {
	ID           int64
	Name         string
	RawKey       string
	Enabled      bool
	ChannelNames []string
	ModelAliases []string
}

type adminAPIKeyCreatePayload struct {
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	ChannelNames []string `json:"channel_names"`
	ModelAliases []string `json:"model_aliases"`
}

type adminAPIKeyUpdatePayload struct {
	Enabled *bool `json:"enabled"`
}

type apiKeySnapshot struct {
	items  []*adminCompatAPIKey
	nextID int64
}

var compatAPIKeys = newAdminCompatAPIKeyStore()

func newAdminCompatAPIKeyStore() *adminCompatAPIKeyStore {
	return &adminCompatAPIKeyStore{
		nextID:    1,
		items:     make(map[int64]*adminCompatAPIKey),
		nameIndex: make(map[string]int64),
	}
}

func adminAPIKeysListHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": compatAPIKeys.list()})
}

func adminAPIKeysCreateHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload adminAPIKeyCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "请求体格式不正确", http.StatusBadRequest)
		return
	}
	item, err := compatAPIKeys.create(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = persistCompatState()
	writeJSON(w, http.StatusCreated, item)
}

func adminAPIKeysUpdateHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminEntityID(r.PathValue("id"), "无效密钥编号")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	var payload adminAPIKeyUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "请求体格式不正确", http.StatusBadRequest)
		return
	}
	item, err := compatAPIKeys.update(id, payload)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	_ = persistCompatState()
	writeJSON(w, http.StatusOK, item)
}

func adminAPIKeysDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminEntityID(r.PathValue("id"), "无效密钥编号")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := compatAPIKeys.delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_ = persistCompatState()
	w.WriteHeader(http.StatusNoContent)
}

func (s *adminCompatAPIKeyStore) list() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]*adminCompatAPIKey, 0, len(s.items))
	for _, item := range s.items {
		clone := *item
		clone.ChannelNames = append([]string(nil), item.ChannelNames...)
		clone.ModelAliases = append([]string(nil), item.ModelAliases...)
		keys = append(keys, &clone)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].ID < keys[j].ID })
	items := make([]map[string]any, 0, len(keys))
	for _, item := range keys {
		items = append(items, apiKeyToListAPI(item))
	}
	return items
}

func (s *adminCompatAPIKeyStore) snapshot() apiKeySnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*adminCompatAPIKey, 0, len(s.items))
	for _, item := range s.items {
		clone := *item
		clone.ChannelNames = append([]string(nil), item.ChannelNames...)
		clone.ModelAliases = append([]string(nil), item.ModelAliases...)
		items = append(items, &clone)
	}
	return apiKeySnapshot{items: items, nextID: s.nextID}
}

func (s *adminCompatAPIKeyStore) restore(items []*adminCompatAPIKey, nextID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[int64]*adminCompatAPIKey)
	s.nameIndex = make(map[string]int64)
	maxID := int64(0)
	for _, item := range items {
		clone := *item
		clone.ChannelNames = append([]string(nil), item.ChannelNames...)
		clone.ModelAliases = append([]string(nil), item.ModelAliases...)
		s.items[item.ID] = &clone
		s.nameIndex[strings.ToLower(item.Name)] = item.ID
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

func (s *adminCompatAPIKeyStore) create(payload adminAPIKeyCreatePayload) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = "new-api-key"
	}
	lowerName := strings.ToLower(name)
	if _, ok := s.nameIndex[lowerName]; ok {
		return nil, fmt.Errorf("同名密钥已存在")
	}
	rawKey, err := generateCompatRawAPIKey()
	if err != nil {
		return nil, fmt.Errorf("生成密钥失败: %w", err)
	}
	item := &adminCompatAPIKey{
		ID:           s.nextID,
		Name:         name,
		RawKey:       rawKey,
		Enabled:      payload.Enabled,
		ChannelNames: uniqueTrimmed(payload.ChannelNames),
		ModelAliases: uniqueTrimmed(payload.ModelAliases),
	}
	s.items[item.ID] = item
	s.nameIndex[lowerName] = item.ID
	s.nextID++
	return apiKeyToCreateAPI(item), nil
}

func (s *adminCompatAPIKeyStore) update(id int64, payload adminAPIKeyUpdatePayload) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("密钥不存在")
	}
	if payload.Enabled != nil {
		item.Enabled = *payload.Enabled
	}
	return apiKeyToListAPI(item), nil
}

func (s *adminCompatAPIKeyStore) delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return fmt.Errorf("密钥不存在")
	}
	delete(s.nameIndex, strings.ToLower(item.Name))
	delete(s.items, id)
	return nil
}

func generateCompatRawAPIKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "oc_" + hex.EncodeToString(buf), nil
}

func uniqueTrimmed(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func apiKeyToListAPI(item *adminCompatAPIKey) map[string]any {
	return map[string]any{
		"id":            item.ID,
		"name":          item.Name,
		"enabled":       item.Enabled,
		"channel_names": append([]string(nil), item.ChannelNames...),
		"model_aliases": append([]string(nil), item.ModelAliases...),
	}
}

func apiKeyToCreateAPI(item *adminCompatAPIKey) map[string]any {
	return map[string]any{
		"id":            item.ID,
		"name":          item.Name,
		"raw_key":       item.RawKey,
		"enabled":       item.Enabled,
		"channel_names": append([]string(nil), item.ChannelNames...),
		"model_aliases": append([]string(nil), item.ModelAliases...),
	}
}

func parseAdminEntityID(raw string, message string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New(message)
	}
	return id, nil
}
