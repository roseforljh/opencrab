package model

import (
	"encoding/json"
	"strings"
)

type TaskPrivateData struct {
	Key string `json:"key,omitempty"`
}

type Task struct {
	ID          int64           `json:"id"`
	TaskID      string          `json:"task_id"`
	Action      string          `json:"action"`
	Status      string          `json:"status"`
	Progress    string          `json:"progress"`
	ResultURL   string          `json:"result_url,omitempty"`
	Data        json.RawMessage `json:"data"`
	PrivateData TaskPrivateData `json:"private_data" gorm:"-"`
}

func (t *Task) GetUpstreamTaskID() string {
	if t == nil {
		return ""
	}
	return strings.TrimSpace(t.TaskID)
}

func (t *Task) GetResultURL() string {
	if t == nil {
		return ""
	}
	return strings.TrimSpace(t.ResultURL)
}
