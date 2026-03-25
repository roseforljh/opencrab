package relay

import (
	relaycommon "github.com/roseforljh/opencrab/relay/common"
	"github.com/roseforljh/opencrab/types"
)

func openCrabErrorFromParamOverride(err error) *types.OpenCrabError {
	if fixedErr, ok := relaycommon.AsParamOverrideReturnError(err); ok {
		return relaycommon.OpenCrabErrorFromParamOverride(fixedErr)
	}
	return types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
}
