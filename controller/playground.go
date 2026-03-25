package controller

import (
	"errors"
	"fmt"

	"github.com/roseforljh/opencrab/middleware"
	"github.com/roseforljh/opencrab/model"
	relaycommon "github.com/roseforljh/opencrab/relay/common"
	"github.com/roseforljh/opencrab/types"

	"github.com/gin-gonic/gin"
)

func Playground(c *gin.Context) {
	var openCrabError *types.OpenCrabError

	defer func() {
		if openCrabError != nil {
			c.JSON(openCrabError.StatusCode, gin.H{
				"error": openCrabError.ToOpenAIError(),
			})
		}
	}()

	useAccessToken := c.GetBool("use_access_token")
	if useAccessToken {
		openCrabError = types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
		return
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAI, nil, nil)
	if err != nil {
		openCrabError = types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
		return
	}

	userId := c.GetInt("id")

	// Write user context to ensure acceptUnsetRatio is available
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		openCrabError = types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
		return
	}
	userCache.WriteContext(c)

	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("playground-%s", relayInfo.UsingGroup),
		Group:  relayInfo.UsingGroup,
	}
	_ = middleware.SetupContextForToken(c, tempToken)

	Relay(c, types.RelayFormatOpenAI)
}
