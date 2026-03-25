package minimax

import (
	"fmt"

	channelconstant "github.com/QuantumNous/opencrab/constant"
	relaycommon "github.com/QuantumNous/opencrab/relay/common"
	"github.com/QuantumNous/opencrab/relay/constant"
	"github.com/QuantumNous/opencrab/types"
)

func GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseUrl := info.ChannelBaseUrl
	if baseUrl == "" {
		baseUrl = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeMiniMax]
	}
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		return fmt.Sprintf("%s/anthropic/v1/messages", info.ChannelBaseUrl), nil
	default:
		switch info.RelayMode {
		case constant.RelayModeChatCompletions:
			return fmt.Sprintf("%s/v1/text/chatcompletion_v2", baseUrl), nil
		case constant.RelayModeAudioSpeech:
			return fmt.Sprintf("%s/v1/t2a_v2", baseUrl), nil
		default:
			return "", fmt.Errorf("unsupported relay mode: %d", info.RelayMode)
		}
	}
}
