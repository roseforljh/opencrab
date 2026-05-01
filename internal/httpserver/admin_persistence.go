package httpserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type compatStateFile struct {
	Channels         []*adminCompatChannel    `json:"channels"`
	NextChannelID    int64                    `json:"next_channel_id"`
	NextModelID      int64                    `json:"next_model_id"`
	NextRouteID      int64                    `json:"next_route_id"`
	ModelIDsByAlias  map[string]int64         `json:"model_ids_by_alias"`
	RouteIDsByKey    map[string]int64         `json:"route_ids_by_key"`
	APIKeys          []*adminCompatAPIKey     `json:"api_keys"`
	NextAPIKeyID     int64                    `json:"next_api_key_id"`
	RequestLogs      []*adminCompatRequestLog `json:"request_logs"`
	NextRequestLogID int64                    `json:"next_request_log_id"`
}

var compatPersistence = struct {
	mu   sync.Mutex
	path string
}{}

func InitCompatStorage(path string) error {
	compatPersistence.mu.Lock()
	defer compatPersistence.mu.Unlock()
	compatPersistence.path = path
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建持久化目录失败: %w", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return persistCompatStateLocked()
		}
		return fmt.Errorf("读取持久化文件失败: %w", err)
	}
	if len(content) == 0 {
		return persistCompatStateLocked()
	}
	var state compatStateFile
	if err := json.Unmarshal(content, &state); err != nil {
		backupPath := path + ".legacy-" + fmt.Sprintf("%d", os.Getpid()) + ".bak"
		if writeErr := os.WriteFile(backupPath, content, 0o644); writeErr != nil {
			return fmt.Errorf("解析持久化文件失败且备份旧文件失败: %w", err)
		}
		return persistCompatStateLocked()
	}
	loadCompatState(state)
	return nil
}

func persistCompatState() error {
	compatPersistence.mu.Lock()
	defer compatPersistence.mu.Unlock()
	return persistCompatStateLocked()
}

func persistCompatStateLocked() error {
	if compatPersistence.path == "" {
		return nil
	}
	state := snapshotCompatState()
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化持久化状态失败: %w", err)
	}
	tmpPath := compatPersistence.path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0o644); err != nil {
		return fmt.Errorf("写入临时持久化文件失败: %w", err)
	}
	if err := os.Rename(tmpPath, compatPersistence.path); err != nil {
		return fmt.Errorf("替换持久化文件失败: %w", err)
	}
	return nil
}

func snapshotCompatState() compatStateFile {
	channels := compatChannels.snapshot()
	apiKeys := compatAPIKeys.snapshot()
	requestLogs := compatRequestLogs.snapshot()
	return compatStateFile{
		Channels:         channels.items,
		NextChannelID:    channels.nextChannelID,
		NextModelID:      channels.nextModelID,
		NextRouteID:      channels.nextRouteID,
		ModelIDsByAlias:  channels.modelIDsByAlias,
		RouteIDsByKey:    channels.routeIDsByKey,
		APIKeys:          apiKeys.items,
		NextAPIKeyID:     apiKeys.nextID,
		RequestLogs:      requestLogs.items,
		NextRequestLogID: requestLogs.nextID,
	}
}

func loadCompatState(state compatStateFile) {
	compatChannels.restore(state.Channels, state.NextChannelID, state.NextModelID, state.NextRouteID, state.ModelIDsByAlias, state.RouteIDsByKey)
	compatAPIKeys.restore(state.APIKeys, state.NextAPIKeyID)
	compatRequestLogs.restore(state.RequestLogs, state.NextRequestLogID)
}
