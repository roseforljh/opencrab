package relay

import (
	"github.com/roseforljh/opencrab/constant"
	"github.com/roseforljh/opencrab/relay/channel"
	"github.com/roseforljh/opencrab/relay/channel/ali"
	"github.com/roseforljh/opencrab/relay/channel/aws"
	"github.com/roseforljh/opencrab/relay/channel/baidu"
	"github.com/roseforljh/opencrab/relay/channel/baidu_v2"
	"github.com/roseforljh/opencrab/relay/channel/claude"
	"github.com/roseforljh/opencrab/relay/channel/cloudflare"
	"github.com/roseforljh/opencrab/relay/channel/codex"
	"github.com/roseforljh/opencrab/relay/channel/cohere"
	"github.com/roseforljh/opencrab/relay/channel/coze"
	"github.com/roseforljh/opencrab/relay/channel/deepseek"
	"github.com/roseforljh/opencrab/relay/channel/dify"
	"github.com/roseforljh/opencrab/relay/channel/gemini"
	"github.com/roseforljh/opencrab/relay/channel/jimeng"
	"github.com/roseforljh/opencrab/relay/channel/jina"
	"github.com/roseforljh/opencrab/relay/channel/minimax"
	"github.com/roseforljh/opencrab/relay/channel/mistral"
	"github.com/roseforljh/opencrab/relay/channel/mokaai"
	"github.com/roseforljh/opencrab/relay/channel/moonshot"
	"github.com/roseforljh/opencrab/relay/channel/ollama"
	"github.com/roseforljh/opencrab/relay/channel/openai"
	"github.com/roseforljh/opencrab/relay/channel/palm"
	"github.com/roseforljh/opencrab/relay/channel/perplexity"
	"github.com/roseforljh/opencrab/relay/channel/replicate"
	"github.com/roseforljh/opencrab/relay/channel/siliconflow"
	"github.com/roseforljh/opencrab/relay/channel/submodel"
	"github.com/roseforljh/opencrab/relay/channel/tencent"
	"github.com/roseforljh/opencrab/relay/channel/vertex"
	"github.com/roseforljh/opencrab/relay/channel/volcengine"
	"github.com/roseforljh/opencrab/relay/channel/xai"
	"github.com/roseforljh/opencrab/relay/channel/xunfei"
	"github.com/roseforljh/opencrab/relay/channel/zhipu"
	"github.com/roseforljh/opencrab/relay/channel/zhipu_4v"
	"github.com/gin-gonic/gin"
)

func GetAdaptor(apiType int) channel.Adaptor {
	switch apiType {
	case constant.APITypeAli:
		return &ali.Adaptor{}
	case constant.APITypeAnthropic:
		return &claude.Adaptor{}
	case constant.APITypeBaidu:
		return &baidu.Adaptor{}
	case constant.APITypeGemini:
		return &gemini.Adaptor{}
	case constant.APITypeOpenAI:
		return &openai.Adaptor{}
	case constant.APITypePaLM:
		return &palm.Adaptor{}
	case constant.APITypeTencent:
		return &tencent.Adaptor{}
	case constant.APITypeXunfei:
		return &xunfei.Adaptor{}
	case constant.APITypeZhipu:
		return &zhipu.Adaptor{}
	case constant.APITypeZhipuV4:
		return &zhipu_4v.Adaptor{}
	case constant.APITypeOllama:
		return &ollama.Adaptor{}
	case constant.APITypePerplexity:
		return &perplexity.Adaptor{}
	case constant.APITypeAws:
		return &aws.Adaptor{}
	case constant.APITypeCohere:
		return &cohere.Adaptor{}
	case constant.APITypeDify:
		return &dify.Adaptor{}
	case constant.APITypeJina:
		return &jina.Adaptor{}
	case constant.APITypeCloudflare:
		return &cloudflare.Adaptor{}
	case constant.APITypeSiliconFlow:
		return &siliconflow.Adaptor{}
	case constant.APITypeVertexAi:
		return &vertex.Adaptor{}
	case constant.APITypeMistral:
		return &mistral.Adaptor{}
	case constant.APITypeDeepSeek:
		return &deepseek.Adaptor{}
	case constant.APITypeMokaAI:
		return &mokaai.Adaptor{}
	case constant.APITypeVolcEngine:
		return &volcengine.Adaptor{}
	case constant.APITypeBaiduV2:
		return &baidu_v2.Adaptor{}
	case constant.APITypeOpenRouter:
		return &openai.Adaptor{}
	case constant.APITypeXinference:
		return &openai.Adaptor{}
	case constant.APITypeXai:
		return &xai.Adaptor{}
	case constant.APITypeCoze:
		return &coze.Adaptor{}
	case constant.APITypeJimeng:
		return &jimeng.Adaptor{}
	case constant.APITypeMoonshot:
		return &moonshot.Adaptor{}
	case constant.APITypeSubmodel:
		return &submodel.Adaptor{}
	case constant.APITypeMiniMax:
		return &minimax.Adaptor{}
	case constant.APITypeReplicate:
		return &replicate.Adaptor{}
	case constant.APITypeCodex:
		return &codex.Adaptor{}
	}
	return nil
}

func GetTaskPlatform(c *gin.Context) constant.TaskPlatform {
	_ = c
	return ""
}

func GetTaskAdaptor(platform constant.TaskPlatform) channel.TaskAdaptor {
	_ = platform
	return nil
}
