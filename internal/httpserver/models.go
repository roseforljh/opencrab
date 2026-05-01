package httpserver

import "net/http"

func modelsHandler(w http.ResponseWriter, r *http.Request) {
	items := compatChannels.listRuntimeModels(runtimeRouteFamilyOpenAI)
	data := make([]map[string]any, 0, len(items))
	for _, model := range items {
		data = append(data, map[string]any{
			"id":       model,
			"object":   "model",
			"owned_by": "openai",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   data,
	})
}
