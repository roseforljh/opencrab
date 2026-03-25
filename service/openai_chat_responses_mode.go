package service

import (
	"github.com/roseforljh/opencrab/service/openaicompat"
	"github.com/roseforljh/opencrab/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	return openaicompat.ShouldChatCompletionsUseResponsesPolicy(policy, channelID, channelType, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return openaicompat.ShouldChatCompletionsUseResponsesGlobal(channelID, channelType, model)
}
