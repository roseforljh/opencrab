package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/roseforljh/opencrab/common"
	"github.com/roseforljh/opencrab/constant"
	"github.com/roseforljh/opencrab/dto"
	"github.com/roseforljh/opencrab/model"
	"github.com/roseforljh/opencrab/relay"
	"github.com/roseforljh/opencrab/relay/channel/ai360"
	"github.com/roseforljh/opencrab/relay/channel/lingyiwanwu"
	"github.com/roseforljh/opencrab/relay/channel/minimax"
	"github.com/roseforljh/opencrab/relay/channel/moonshot"
	relaycommon "github.com/roseforljh/opencrab/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

var openAIModels []dto.OpenAIModels
var openAIModelsMap map[string]dto.OpenAIModels
var channelId2Models map[int][]string

func init() {
	for i := 0; i < constant.APITypeDummy; i++ {
		if i == constant.APITypeAIProxyLibrary {
			continue
		}
		adaptor := relay.GetAdaptor(i)
		channelName := adaptor.GetChannelName()
		modelNames := adaptor.GetModelList()
		for _, modelName := range modelNames {
			openAIModels = append(openAIModels, dto.OpenAIModels{
				Id:      modelName,
				Object:  "model",
				Created: 1626777600,
				OwnedBy: channelName,
			})
		}
	}
	for _, modelName := range ai360.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{Id: modelName, Object: "model", Created: 1626777600, OwnedBy: ai360.ChannelName})
	}
	for _, modelName := range moonshot.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{Id: modelName, Object: "model", Created: 1626777600, OwnedBy: moonshot.ChannelName})
	}
	for _, modelName := range lingyiwanwu.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{Id: modelName, Object: "model", Created: 1626777600, OwnedBy: lingyiwanwu.ChannelName})
	}
	for _, modelName := range minimax.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{Id: modelName, Object: "model", Created: 1626777600, OwnedBy: minimax.ChannelName})
	}
	openAIModelsMap = make(map[string]dto.OpenAIModels)
	for _, aiModel := range openAIModels {
		openAIModelsMap[aiModel.Id] = aiModel
	}
	channelId2Models = make(map[int][]string)
	for i := 1; i <= constant.ChannelTypeDummy; i++ {
		apiType, success := common.ChannelType2APIType(i)
		if !success || apiType == constant.APITypeAIProxyLibrary {
			continue
		}
		meta := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: i}}
		adaptor := relay.GetAdaptor(apiType)
		adaptor.Init(meta)
		channelId2Models[i] = adaptor.GetModelList()
	}
	openAIModels = lo.UniqBy(openAIModels, func(m dto.OpenAIModels) string { return m.Id })
}

func ListModels(c *gin.Context, modelType int) {
	userOpenAiModels := make([]dto.OpenAIModels, 0)

	modelLimitEnable := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
	if modelLimitEnable {
		s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
		var tokenModelLimit map[string]bool
		if ok {
			tokenModelLimit = s.(map[string]bool)
		} else {
			tokenModelLimit = map[string]bool{}
		}
		for allowModel := range tokenModelLimit {
			if oaiModel, ok := openAIModelsMap[allowModel]; ok {
				oaiModel.SupportedEndpointTypes = model.GetModelSupportEndpointTypes(allowModel)
				userOpenAiModels = append(userOpenAiModels, oaiModel)
			} else {
				userOpenAiModels = append(userOpenAiModels, dto.OpenAIModels{
					Id:                     allowModel,
					Object:                 "model",
					Created:                1626777600,
					OwnedBy:                "custom",
					SupportedEndpointTypes: model.GetModelSupportEndpointTypes(allowModel),
				})
			}
		}
	} else {
		userId := c.GetInt("id")
		userGroup, err := model.GetUserGroup(userId, false)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		if modelType == constant.ChannelTypeUnknown {
			modelNames := model.GetGroupModels(userGroup)
			for _, modelName := range modelNames {
				if oaiModel, ok := openAIModelsMap[modelName]; ok {
					oaiModel.SupportedEndpointTypes = model.GetModelSupportEndpointTypes(modelName)
					userOpenAiModels = append(userOpenAiModels, oaiModel)
				} else {
					userOpenAiModels = append(userOpenAiModels, dto.OpenAIModels{
						Id:                     modelName,
						Object:                 "model",
						Created:                1626777600,
						OwnedBy:                "custom",
						SupportedEndpointTypes: model.GetModelSupportEndpointTypes(modelName),
					})
				}
			}
		} else {
			for _, modelName := range channelId2Models[modelType] {
				if model.IsModelInGroup(modelName, userGroup) {
					if oaiModel, ok := openAIModelsMap[modelName]; ok {
						userOpenAiModels = append(userOpenAiModels, oaiModel)
					} else {
						userOpenAiModels = append(userOpenAiModels, dto.OpenAIModels{Id: modelName, Object: "model", Created: 1626777600, OwnedBy: "custom"})
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": userOpenAiModels, "success": true})
}

func DashboardListModels(c *gin.Context) {
	ListModels(c, constant.ChannelTypeUnknown)
}

func ChannelListModels(c *gin.Context) {
	ListModels(c, common.ChannelTypeId(c.Query("id")))
}

func EnabledListModels(c *gin.Context) {
	userGroup := c.Query("group")
	if userGroup == "" {
		userId := c.GetInt("id")
		if userId != 0 {
			group, _ := model.GetUserGroup(userId, false)
			userGroup = group
		}
	}
	data := model.GetGroupModels(userGroup)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func TestModel(c *gin.Context) {
	modelName := c.Param("model")
	channelType, _ := strconv.Atoi(c.Query("channel_type"))
	channel, err := model.GetRandomEnabledChannel(modelName, channelType)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": channel})
}

func GetModelStatus(c *gin.Context) {
	modelName := c.Param("model")
	channels, err := model.GetEnabledChannelsByModel(modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": channels})
}

func GetTestModels(c *gin.Context) {
	modelNames := model.GetEnabledModels()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": modelNames})
}

func TestModelEndpoint(c *gin.Context) {
	startTime := time.Now()
	modelName := c.Query("model")
	if modelName == "" {
		common.ApiErrorMsg(c, "model is required")
		return
	}
	channel, err := model.GetRandomEnabledChannel(modelName, 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	latency := time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"channel_id": channel.Id, "latency_ms": latency, "message": fmt.Sprintf("model %s available", modelName)}})
}
