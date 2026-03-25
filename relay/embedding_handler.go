package relay

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/roseforljh/opencrab/common"
	"github.com/roseforljh/opencrab/dto"
	"github.com/roseforljh/opencrab/logger"
	relaycommon "github.com/roseforljh/opencrab/relay/common"
	"github.com/roseforljh/opencrab/relay/helper"
	"github.com/roseforljh/opencrab/service"
	"github.com/roseforljh/opencrab/types"

	"github.com/gin-gonic/gin"
)

func EmbeddingHelper(c *gin.Context, info *relaycommon.RelayInfo) (openCrabError *types.OpenCrabError) {
	info.InitChannelMeta(c)

	embeddingReq, ok := info.Request.(*dto.EmbeddingRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected *dto.EmbeddingRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(embeddingReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to EmbeddingRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	convertedRequest, err := adaptor.ConvertEmbeddingRequest(c, info, *request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			return openCrabErrorFromParamOverride(err)
		}
	}

	logger.LogDebug(c, fmt.Sprintf("converted embedding request body: %s", string(jsonData)))
	requestBody := bytes.NewBuffer(jsonData)
	statusCodeMappingStr := c.GetString("status_code_mapping")
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK {
			openCrabError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			// reset status code 重置状态码
			service.ResetStatusCode(openCrabError, statusCodeMappingStr)
			return openCrabError
		}
	}

	usage, openCrabError := adaptor.DoResponse(c, httpResp, info)
	if openCrabError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(openCrabError, statusCodeMappingStr)
		return openCrabError
	}
	postConsumeQuota(c, info, usage.(*dto.Usage))
	return nil
}
