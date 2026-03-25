package service

import (
	"fmt"
	"strings"

	"github.com/roseforljh/opencrab/constant"
	"github.com/roseforljh/opencrab/dto"
)

type MidjourneyRequestError struct {
	Description string
}

func GetMjRequestModel(relayMode int, req *dto.MidjourneyRequest) (string, *MidjourneyRequestError, bool) {
	if req == nil {
		return "", &MidjourneyRequestError{Description: "midjourney request is nil"}, false
	}
	return "midjourney", nil, true
}

func CoverTaskActionToModelName(platform constant.TaskPlatform, action string) string {
	action = strings.TrimSpace(action)
	switch platform {
	case constant.TaskPlatformSuno:
		switch strings.ToUpper(action) {
		case constant.SunoActionMusic:
			return "suno_music"
		case constant.SunoActionLyrics:
			return "suno_lyrics"
		default:
			if action == "" {
				return "suno_music"
			}
			return fmt.Sprintf("suno_%s", strings.ToLower(action))
		}
	default:
		if action == "" {
			return string(platform)
		}
		return fmt.Sprintf("%s_%s", platform, strings.ToLower(action))
	}
}
