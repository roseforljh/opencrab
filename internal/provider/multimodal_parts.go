package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"opencrab/internal/domain"
)

type mediaDescriptor struct {
	MimeType string
	Data     string
	URL      string
	FileURI  string
	FileID   string
	Filename string
	Format   string
}

func cloneRawMap(src map[string]json.RawMessage) map[string]json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]json.RawMessage, len(src))
	for key, value := range src {
		dst[key] = append(json.RawMessage(nil), value...)
	}
	return dst
}

func rawJSONString(value string) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

func decodeStringRaw(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func setRawString(metadata map[string]json.RawMessage, key string, value string) map[string]json.RawMessage {
	if strings.TrimSpace(value) == "" {
		return metadata
	}
	if metadata == nil {
		metadata = map[string]json.RawMessage{}
	}
	metadata[key] = rawJSONString(value)
	return metadata
}

func buildDataURL(mimeType string, data string) string {
	if strings.TrimSpace(mimeType) == "" || strings.TrimSpace(data) == "" {
		return ""
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, data)
}

func partTypeFromMime(mimeType string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case mimeType == "application/pdf", strings.HasPrefix(mimeType, "text/"), strings.Contains(mimeType, "document"), strings.Contains(mimeType, "word"), strings.Contains(mimeType, "sheet"), strings.Contains(mimeType, "presentation"):
		return "document"
	default:
		return "file"
	}
}

func formatToMime(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "wav", "wave":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "pcm16":
		return "audio/pcm"
	case "opus":
		return "audio/opus"
	default:
		return ""
	}
}

func mimeToAudioFormat(mimeType string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch mimeType {
	case "audio/wav", "audio/x-wav":
		return "wav"
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/flac":
		return "flac"
	case "audio/pcm":
		return "pcm16"
	case "audio/opus":
		return "opus"
	default:
		return "wav"
	}
}

func extractMediaDescriptor(part domain.UnifiedPart) mediaDescriptor {
	desc := mediaDescriptor{}
	if part.Metadata == nil {
		return desc
	}
	if raw, ok := part.Metadata["mime_type"]; ok {
		desc.MimeType = decodeStringRaw(raw)
	}
	if raw, ok := part.Metadata["data"]; ok {
		desc.Data = decodeStringRaw(raw)
	}
	if raw, ok := part.Metadata["url"]; ok {
		desc.URL = decodeStringRaw(raw)
	}
	if raw, ok := part.Metadata["file_uri"]; ok {
		desc.FileURI = decodeStringRaw(raw)
	}
	if raw, ok := part.Metadata["file_id"]; ok {
		desc.FileID = decodeStringRaw(raw)
	}
	if raw, ok := part.Metadata["filename"]; ok {
		desc.Filename = decodeStringRaw(raw)
	}
	if raw, ok := part.Metadata["image_url"]; ok {
		var imageURL string
		if err := json.Unmarshal(raw, &imageURL); err == nil {
			desc.URL = imageURL
		}
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(raw, &payload); err == nil {
			if urlRaw, exists := payload["url"]; exists {
				desc.URL = decodeStringRaw(urlRaw)
			}
			if fileIDRaw, exists := payload["file_id"]; exists {
				desc.FileID = decodeStringRaw(fileIDRaw)
			}
		}
	}
	if raw, ok := part.Metadata["input_audio"]; ok {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(raw, &payload); err == nil {
			if dataRaw, exists := payload["data"]; exists {
				desc.Data = decodeStringRaw(dataRaw)
			}
			if formatRaw, exists := payload["format"]; exists {
				desc.Format = decodeStringRaw(formatRaw)
				if desc.MimeType == "" {
					desc.MimeType = formatToMime(desc.Format)
				}
			}
		}
	}
	if raw, ok := part.Metadata["source"]; ok {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(raw, &payload); err == nil {
			if mediaRaw, exists := payload["media_type"]; exists && desc.MimeType == "" {
				desc.MimeType = decodeStringRaw(mediaRaw)
			}
			if dataRaw, exists := payload["data"]; exists && desc.Data == "" {
				desc.Data = decodeStringRaw(dataRaw)
			}
			if urlRaw, exists := payload["url"]; exists && desc.URL == "" {
				desc.URL = decodeStringRaw(urlRaw)
			}
			if fileIDRaw, exists := payload["file_id"]; exists && desc.FileID == "" {
				desc.FileID = decodeStringRaw(fileIDRaw)
			}
		}
	}
	if raw, ok := part.Metadata["inlineData"]; ok {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(raw, &payload); err == nil {
			if mimeRaw, exists := payload["mimeType"]; exists && desc.MimeType == "" {
				desc.MimeType = decodeStringRaw(mimeRaw)
			}
			if dataRaw, exists := payload["data"]; exists && desc.Data == "" {
				desc.Data = decodeStringRaw(dataRaw)
			}
		}
	}
	if raw, ok := part.Metadata["fileData"]; ok {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(raw, &payload); err == nil {
			if mimeRaw, exists := payload["mimeType"]; exists && desc.MimeType == "" {
				desc.MimeType = decodeStringRaw(mimeRaw)
			}
			if uriRaw, exists := payload["fileUri"]; exists && desc.FileURI == "" {
				desc.FileURI = decodeStringRaw(uriRaw)
			}
		}
	}
	if raw, ok := part.Metadata["file_data"]; ok {
		desc.Data = decodeStringRaw(raw)
	}
	return desc
}

func enrichPartMetadata(partType string, metadata map[string]json.RawMessage, desc mediaDescriptor) map[string]json.RawMessage {
	metadata = cloneRawMap(metadata)
	metadata = setRawString(metadata, "mime_type", desc.MimeType)
	metadata = setRawString(metadata, "data", desc.Data)
	metadata = setRawString(metadata, "url", desc.URL)
	metadata = setRawString(metadata, "file_uri", desc.FileURI)
	metadata = setRawString(metadata, "file_id", desc.FileID)
	metadata = setRawString(metadata, "filename", desc.Filename)
	if partType == "audio" && strings.TrimSpace(desc.Format) != "" {
		metadata = setRawString(metadata, "format", desc.Format)
	}
	return metadata
}

func rawJSONToAny(raw json.RawMessage) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	return value
}
