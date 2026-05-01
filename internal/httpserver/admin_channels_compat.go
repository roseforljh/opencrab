package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type adminCompatChannelStore struct {
	mu             sync.RWMutex
	nextChannelID  int64
	nextModelID    int64
	nextRouteID    int64
	channels       map[int64]*adminCompatChannel
	modelIDsByAlias map[string]int64
	routeIDsByKey  map[string]int64
}

type adminCompatChannel struct {
	ID              int64
	Name            string
	Provider        string
	Endpoint        string
	APIKey          string
	Enabled         bool
	ModelIDs        []string
	RPMLimit        int
	MaxInflight     int
	SafetyFactor    float64
	EnabledForAsync bool
	DispatchWeight  int
	UpdatedAt       string
}

type adminChannelUpsertPayload struct {
	Name            string   `json:"name"`
	Provider        string   `json:"provider"`
	Endpoint        string   `json:"endpoint"`
	APIKey          string   `json:"api_key"`
	Enabled         bool     `json:"enabled"`
	ModelIDs        []string `json:"model_ids"`
	RPMLimit        int      `json:"rpm_limit"`
	MaxInflight     int      `json:"max_inflight"`
	SafetyFactor    float64  `json:"safety_factor"`
	EnabledForAsync bool     `json:"enabled_for_async"`
	DispatchWeight  int      `json:"dispatch_weight"`
}

type adminChannelTestPayload struct {
	Model string `json:"model"`
}

type channelSnapshot struct {
	items           []*adminCompatChannel
	nextChannelID   int64
	nextModelID     int64
	nextRouteID     int64
	modelIDsByAlias map[string]int64
	routeIDsByKey   map[string]int64
}

var compatChannels = newAdminCompatChannelStore()

func newAdminCompatChannelStore() *adminCompatChannelStore {
	return &adminCompatChannelStore{
		nextChannelID:   1,
		nextModelID:     1,
		nextRouteID:     1,
		channels:        make(map[int64]*adminCompatChannel),
		modelIDsByAlias: make(map[string]int64),
		routeIDsByKey:   make(map[string]int64),
	}
}

func adminChannelsListHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": compatChannels.listChannels()})
}

func adminChannelsCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeAdminChannelUpsertPayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	channel, err := compatChannels.createChannel(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = persistCompatState()
	writeJSON(w, http.StatusCreated, channel)
}

func adminChannelsUpdateHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminChannelID(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	payload, err := decodeAdminChannelUpsertPayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	channel, err := compatChannels.updateChannel(id, payload)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	_ = persistCompatState()
	writeJSON(w, http.StatusOK, channel)
}

func adminChannelsDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminChannelID(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := compatChannels.deleteChannel(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_ = persistCompatState()
	w.WriteHeader(http.StatusNoContent)
}

func adminChannelTestHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminChannelID(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	channel, ok := compatChannels.getChannel(id)
	if !ok {
		http.Error(w, "渠道不存在", http.StatusNotFound)
		return
	}
	defer r.Body.Close()
	var payload adminChannelTestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		http.Error(w, "请求体格式不正确", http.StatusBadRequest)
		return
	}
	model := strings.TrimSpace(payload.Model)
	if model == "" {
		if len(channel.ModelIDs) == 0 {
			http.Error(w, "当前渠道没有可测试的模型 ID", http.StatusBadRequest)
			return
		}
		model = strings.TrimSpace(channel.ModelIDs[0])
	}
	message, err := runCompatChannelTest(channel, model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"model":   model,
		"message": message,
	})
}

func adminModelsListHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": compatChannels.listModels()})
}

func adminModelsDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseAdminEntityID(r.PathValue("id"), "无效模型编号")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := compatChannels.deleteModel(id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	_ = persistCompatState()
	w.WriteHeader(http.StatusNoContent)
}

func adminModelRoutesListHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": compatChannels.listModelRoutes()})
}

func decodeAdminChannelUpsertPayload(r *http.Request) (adminChannelUpsertPayload, error) {
	defer r.Body.Close()
	var payload adminChannelUpsertPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return adminChannelUpsertPayload{}, fmt.Errorf("请求体格式不正确")
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Provider = strings.TrimSpace(payload.Provider)
	payload.Endpoint = strings.TrimSpace(payload.Endpoint)
	if payload.Name == "" {
		return adminChannelUpsertPayload{}, fmt.Errorf("渠道名称不能为空")
	}
	if payload.Provider == "" {
		return adminChannelUpsertPayload{}, fmt.Errorf("兼容类型不能为空")
	}
	if payload.Endpoint == "" {
		return adminChannelUpsertPayload{}, fmt.Errorf("请求地址不能为空")
	}
	cleanModels := make([]string, 0, len(payload.ModelIDs))
	seen := make(map[string]struct{})
	for _, modelID := range payload.ModelIDs {
		trimmed := strings.TrimSpace(modelID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		cleanModels = append(cleanModels, trimmed)
	}
	if len(cleanModels) == 0 {
		return adminChannelUpsertPayload{}, fmt.Errorf("至少添加一个模型 ID")
	}
	payload.ModelIDs = cleanModels
	if payload.RPMLimit <= 0 {
		payload.RPMLimit = 1000
	}
	if payload.MaxInflight <= 0 {
		payload.MaxInflight = 32
	}
	if payload.SafetyFactor <= 0 {
		payload.SafetyFactor = 0.9
	}
	if payload.DispatchWeight <= 0 {
		payload.DispatchWeight = 100
	}
	return payload, nil
}

func parseAdminChannelID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("无效渠道编号")
	}
	return id, nil
}

func (s *adminCompatChannelStore) createChannel(payload adminChannelUpsertPayload) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, channel := range s.channels {
		if strings.EqualFold(channel.Name, payload.Name) {
			return nil, fmt.Errorf("同名渠道已存在")
		}
	}
	channel := &adminCompatChannel{
		ID:              s.nextChannelID,
		Name:            payload.Name,
		Provider:        payload.Provider,
		Endpoint:        payload.Endpoint,
		APIKey:          payload.APIKey,
		Enabled:         payload.Enabled,
		ModelIDs:        append([]string(nil), payload.ModelIDs...),
		RPMLimit:        payload.RPMLimit,
		MaxInflight:     payload.MaxInflight,
		SafetyFactor:    payload.SafetyFactor,
		EnabledForAsync: payload.EnabledForAsync,
		DispatchWeight:  payload.DispatchWeight,
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	s.channels[channel.ID] = channel
	s.nextChannelID++
	return channelToAPI(channel), nil
}

func (s *adminCompatChannelStore) updateChannel(id int64, payload adminChannelUpsertPayload) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel, ok := s.channels[id]
	if !ok {
		return nil, fmt.Errorf("渠道不存在")
	}
	for existingID, existing := range s.channels {
		if existingID != id && strings.EqualFold(existing.Name, payload.Name) {
			return nil, fmt.Errorf("同名渠道已存在")
		}
	}
	channel.Name = payload.Name
	channel.Provider = payload.Provider
	channel.Endpoint = payload.Endpoint
	if strings.TrimSpace(payload.APIKey) != "" {
		channel.APIKey = payload.APIKey
	}
	channel.Enabled = payload.Enabled
	channel.ModelIDs = append([]string(nil), payload.ModelIDs...)
	channel.RPMLimit = payload.RPMLimit
	channel.MaxInflight = payload.MaxInflight
	channel.SafetyFactor = payload.SafetyFactor
	channel.EnabledForAsync = payload.EnabledForAsync
	channel.DispatchWeight = payload.DispatchWeight
	channel.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return channelToAPI(channel), nil
}

func (s *adminCompatChannelStore) deleteChannel(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[id]; !ok {
		return fmt.Errorf("渠道不存在")
	}
	delete(s.channels, id)
	return nil
}

func (s *adminCompatChannelStore) deleteModel(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alias := ""
	for key, value := range s.modelIDsByAlias {
		if value == id {
			alias = key
			break
		}
	}
	if alias == "" {
		return fmt.Errorf("模型不存在")
	}

	removed := false
	for _, channel := range s.channels {
		filtered := channel.ModelIDs[:0]
		for _, modelID := range channel.ModelIDs {
			if modelID == alias {
				removed = true
				continue
			}
			filtered = append(filtered, modelID)
		}
		channel.ModelIDs = append([]string(nil), filtered...)
	}

	if !removed {
		return fmt.Errorf("模型不存在")
	}

	delete(s.modelIDsByAlias, alias)
	for key := range s.routeIDsByKey {
		if strings.HasSuffix(key, "::"+alias) {
			delete(s.routeIDsByKey, key)
		}
	}
	return nil
}

func (s *adminCompatChannelStore) getChannel(id int64) (*adminCompatChannel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channel, ok := s.channels[id]
	if !ok {
		return nil, false
	}
	clone := *channel
	clone.ModelIDs = append([]string(nil), channel.ModelIDs...)
	return &clone, true
}

func (s *adminCompatChannelStore) resolveClaudeChannel(model string) (*adminCompatChannel, bool) {
	alias := strings.TrimSpace(model)
	if alias == "" {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched *adminCompatChannel
	for _, channel := range s.channels {
		if !channel.Enabled || !strings.EqualFold(strings.TrimSpace(channel.Provider), "claude") {
			continue
		}
		for _, modelID := range channel.ModelIDs {
			if strings.TrimSpace(modelID) != alias {
				continue
			}
			if matched == nil || channel.ID < matched.ID {
				clone := *channel
				clone.ModelIDs = append([]string(nil), channel.ModelIDs...)
				matched = &clone
			}
			break
		}
	}

	if matched == nil {
		return nil, false
	}
	return matched, true
}

func (s *adminCompatChannelStore) listChannels() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channels := make([]*adminCompatChannel, 0, len(s.channels))
	for _, channel := range s.channels {
		clone := *channel
		clone.ModelIDs = append([]string(nil), channel.ModelIDs...)
		channels = append(channels, &clone)
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i].ID < channels[j].ID })
	items := make([]map[string]any, 0, len(channels))
	for _, channel := range channels {
		items = append(items, channelToAPI(channel))
	}
	return items
}

func (s *adminCompatChannelStore) snapshot() channelSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*adminCompatChannel, 0, len(s.channels))
	for _, item := range s.channels {
		clone := *item
		clone.ModelIDs = append([]string(nil), item.ModelIDs...)
		items = append(items, &clone)
	}
	modelIDsByAlias := make(map[string]int64, len(s.modelIDsByAlias))
	for key, value := range s.modelIDsByAlias {
		modelIDsByAlias[key] = value
	}
	routeIDsByKey := make(map[string]int64, len(s.routeIDsByKey))
	for key, value := range s.routeIDsByKey {
		routeIDsByKey[key] = value
	}
	return channelSnapshot{
		items:           items,
		nextChannelID:   s.nextChannelID,
		nextModelID:     s.nextModelID,
		nextRouteID:     s.nextRouteID,
		modelIDsByAlias: modelIDsByAlias,
		routeIDsByKey:   routeIDsByKey,
	}
}

func (s *adminCompatChannelStore) restore(items []*adminCompatChannel, nextChannelID int64, nextModelID int64, nextRouteID int64, modelIDsByAlias map[string]int64, routeIDsByKey map[string]int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels = make(map[int64]*adminCompatChannel)
	maxChannelID := int64(0)
	for _, item := range items {
		clone := *item
		clone.ModelIDs = append([]string(nil), item.ModelIDs...)
		s.channels[item.ID] = &clone
		if item.ID > maxChannelID {
			maxChannelID = item.ID
		}
	}
	if nextChannelID <= maxChannelID {
		nextChannelID = maxChannelID + 1
	}
	if nextChannelID <= 0 {
		nextChannelID = 1
	}
	if nextModelID <= 0 {
		nextModelID = 1
	}
	if nextRouteID <= 0 {
		nextRouteID = 1
	}
	s.nextChannelID = nextChannelID
	s.nextModelID = nextModelID
	s.nextRouteID = nextRouteID
	s.modelIDsByAlias = make(map[string]int64)
	for key, value := range modelIDsByAlias {
		s.modelIDsByAlias[key] = value
	}
	s.routeIDsByKey = make(map[string]int64)
	for key, value := range routeIDsByKey {
		s.routeIDsByKey[key] = value
	}
}

func (s *adminCompatChannelStore) listModels() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]map[string]any, 0)
	seen := make(map[string]struct{})
	aliases := make([]string, 0)
	for _, channel := range s.channels {
		for _, modelID := range channel.ModelIDs {
			if _, ok := seen[modelID]; ok {
				continue
			}
			seen[modelID] = struct{}{}
			aliases = append(aliases, modelID)
		}
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		id, ok := s.modelIDsByAlias[alias]
		if !ok {
			id = s.nextModelID
			s.modelIDsByAlias[alias] = id
			s.nextModelID++
		}
		items = append(items, map[string]any{
			"id":             id,
			"alias":          alias,
			"upstream_model": alias,
		})
	}
	return items
}

func (s *adminCompatChannelStore) listModelRoutes() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	routes := make([]map[string]any, 0)
	channels := make([]*adminCompatChannel, 0, len(s.channels))
	for _, channel := range s.channels {
		clone := *channel
		clone.ModelIDs = append([]string(nil), channel.ModelIDs...)
		channels = append(channels, &clone)
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i].ID < channels[j].ID })
	for _, channel := range channels {
		for index, modelID := range channel.ModelIDs {
			key := channel.Name + "::" + modelID
			routeID, ok := s.routeIDsByKey[key]
			if !ok {
				routeID = s.nextRouteID
				s.routeIDsByKey[key] = routeID
				s.nextRouteID++
			}
			routes = append(routes, map[string]any{
				"id":              routeID,
				"model_alias":     modelID,
				"channel_name":    channel.Name,
				"invocation_mode": "auto",
				"priority":        index + 1,
				"fallback_model":  "",
			})
		}
	}
	return routes
}

func channelToAPI(channel *adminCompatChannel) map[string]any {
	return map[string]any{
		"id":                channel.ID,
		"name":              channel.Name,
		"provider":          channel.Provider,
		"endpoint":          channel.Endpoint,
		"enabled":           channel.Enabled,
		"rpm_limit":         channel.RPMLimit,
		"max_inflight":      channel.MaxInflight,
		"safety_factor":     channel.SafetyFactor,
		"enabled_for_async": channel.EnabledForAsync,
		"dispatch_weight":   channel.DispatchWeight,
		"updated_at":        channel.UpdatedAt,
	}
}

func runCompatChannelTest(channel *adminCompatChannel, model string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(channel.Provider))
	switch provider {
	case "claude":
		return testClaudeChannel(channel, model)
	case "gemini":
		return testGeminiChannel(channel, model)
	default:
		return testOpenAICompatibleChannel(channel, model)
	}
}

func testOpenAICompatibleChannel(channel *adminCompatChannel, model string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens": 16,
	})
	request, err := http.NewRequest(http.MethodPost, joinURL(channel.Endpoint, "/chat/completions"), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造测试请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(channel.APIKey); apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
	if err := executeChannelTestRequest(request); err != nil {
		return "", err
	}
	return fmt.Sprintf("模型 %s 连接成功", model), nil
}

func testClaudeChannel(channel *adminCompatChannel, model string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": 16,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
	})
	request, err := http.NewRequest(http.MethodPost, joinURL(channel.Endpoint, "/v1/messages"), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造测试请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Anthropic-Version", "2023-06-01")
	if apiKey := strings.TrimSpace(channel.APIKey); apiKey != "" {
		request.Header.Set("X-API-Key", apiKey)
	}
	if err := executeChannelTestRequest(request); err != nil {
		return "", err
	}
	return fmt.Sprintf("Claude 模型 %s 连接成功", model), nil
}

func testGeminiChannel(channel *adminCompatChannel, model string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{{
			"parts": []map[string]string{{"text": "ping"}},
		}},
	})
	path := fmt.Sprintf("/models/%s:generateContent", url.PathEscape(model))
	request, err := http.NewRequest(http.MethodPost, joinURL(channel.Endpoint, path), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造测试请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(channel.APIKey); apiKey != "" {
		request.Header.Set("X-Goog-Api-Key", apiKey)
	}
	if err := executeChannelTestRequest(request); err != nil {
		return "", err
	}
	return fmt.Sprintf("Gemini 模型 %s 连接成功", model), nil
}

func executeChannelTestRequest(request *http.Request) error {
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("请求上游失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("上游返回 %d: %s", response.StatusCode, message)
}

func joinURL(base string, path string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(base), "/")
	trimmedPath := "/" + strings.TrimLeft(path, "/")
	return trimmedBase + trimmedPath
}
