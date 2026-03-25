package service

import (
	"fmt"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/roseforljh/opencrab/dto"
	"github.com/roseforljh/opencrab/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var channelAffinityUsageCacheTestSeq uint64

func buildChannelAffinityStatsContextForTest(ruleName, usingGroup, keyFP string) *gin.Context {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	setChannelAffinityContext(ctx, channelAffinityMeta{
		CacheKey:       fmt.Sprintf("test:%s:%s:%s", ruleName, usingGroup, keyFP),
		TTLSeconds:     600,
		RuleName:       ruleName,
		UsingGroup:     usingGroup,
		KeyFingerprint: keyFP,
	})
	return ctx
}

func uniqueChannelAffinityTestID(prefix string) string {
	seq := atomic.AddUint64(&channelAffinityUsageCacheTestSeq, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), seq)
}

func TestObserveChannelAffinityUsageCacheByRelayFormat_ClaudeMode(t *testing.T) {
	resetChannelAffinityUsageCacheStatsForTest()
	ruleName := uniqueChannelAffinityTestID("rule")
	usingGroup := "default"
	keyFP := uniqueChannelAffinityTestID("fp")
	ctx := buildChannelAffinityStatsContextForTest(ruleName, usingGroup, keyFP)

	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 40,
		TotalTokens:      140,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 30,
		},
	}

	ObserveChannelAffinityUsageCacheByRelayFormat(ctx, usage, types.RelayFormatClaude)
	stats := GetChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP)

	require.EqualValues(t, 1, stats.Total)
	require.EqualValues(t, 1, stats.Hit)
	require.EqualValues(t, 100, stats.PromptTokens)
	require.EqualValues(t, 40, stats.CompletionTokens)
	require.EqualValues(t, 140, stats.TotalTokens)
	require.EqualValues(t, 30, stats.CachedTokens)
	require.Equal(t, cacheTokenRateModeCachedOverPromptPlusCached, stats.CachedTokenRateMode)
}

func TestObserveChannelAffinityUsageCacheByRelayFormat_MixedMode(t *testing.T) {
	resetChannelAffinityUsageCacheStatsForTest()
	ruleName := uniqueChannelAffinityTestID("rule")
	usingGroup := "default"
	keyFP := uniqueChannelAffinityTestID("fp")
	ctx := buildChannelAffinityStatsContextForTest(ruleName, usingGroup, keyFP)

	openAIUsage := &dto.Usage{
		PromptTokens: 100,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 10,
		},
	}
	claudeUsage := &dto.Usage{
		PromptTokens: 80,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 20,
		},
	}

	ObserveChannelAffinityUsageCacheByRelayFormat(ctx, openAIUsage, types.RelayFormatOpenAI)
	ObserveChannelAffinityUsageCacheByRelayFormat(ctx, claudeUsage, types.RelayFormatClaude)
	stats := GetChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP)

	require.EqualValues(t, 2, stats.Total)
	require.EqualValues(t, 2, stats.Hit)
	require.EqualValues(t, 180, stats.PromptTokens)
	require.EqualValues(t, 30, stats.CachedTokens)
	require.Equal(t, cacheTokenRateModeMixed, stats.CachedTokenRateMode)
}

func TestObserveChannelAffinityUsageCacheByRelayFormat_UnsupportedModeKeepsEmpty(t *testing.T) {
	resetChannelAffinityUsageCacheStatsForTest()
	ruleName := uniqueChannelAffinityTestID("rule")
	usingGroup := "default"
	keyFP := uniqueChannelAffinityTestID("fp")
	ctx := buildChannelAffinityStatsContextForTest(ruleName, usingGroup, keyFP)

	usage := &dto.Usage{
		PromptTokens: 100,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 25,
		},
	}

	ObserveChannelAffinityUsageCacheByRelayFormat(ctx, usage, types.RelayFormatGemini)
	stats := GetChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP)

	require.EqualValues(t, 1, stats.Total)
	require.EqualValues(t, 1, stats.Hit)
	require.EqualValues(t, 25, stats.CachedTokens)
	require.Equal(t, "", stats.CachedTokenRateMode)
}

func TestObserveChannelAffinityUsageCacheByRelayFormat_SeparateRunsUseDistinctKeys(t *testing.T) {
	resetChannelAffinityUsageCacheStatsForTest()
	ruleNameA := uniqueChannelAffinityTestID("rule")
	keyFPA := uniqueChannelAffinityTestID("fp")
	ctxA := buildChannelAffinityStatsContextForTest(ruleNameA, "default", keyFPA)
	ObserveChannelAffinityUsageCacheByRelayFormat(ctxA, &dto.Usage{PromptTokens: 1}, types.RelayFormatOpenAI)
	statsA := GetChannelAffinityUsageCacheStats(ruleNameA, "default", keyFPA)
	require.EqualValues(t, 1, statsA.Total)

	ruleNameB := uniqueChannelAffinityTestID("rule")
	keyFPB := uniqueChannelAffinityTestID("fp")
	ctxB := buildChannelAffinityStatsContextForTest(ruleNameB, "default", keyFPB)
	statsBBefore := GetChannelAffinityUsageCacheStats(ruleNameB, "default", keyFPB)
	require.EqualValues(t, 0, statsBBefore.Total)

	ObserveChannelAffinityUsageCacheByRelayFormat(ctxB, &dto.Usage{PromptTokens: 2}, types.RelayFormatOpenAI)
	statsBAfter := GetChannelAffinityUsageCacheStats(ruleNameB, "default", keyFPB)
	require.EqualValues(t, 1, statsBAfter.Total)
}

func TestObserveChannelAffinityUsageCacheStatsCanBeResetBetweenRuns(t *testing.T) {
	resetChannelAffinityUsageCacheStatsForTest()
	ruleName := "shared-rule"
	usingGroup := "default"
	keyFP := "shared-fp"

	ctx := buildChannelAffinityStatsContextForTest(ruleName, usingGroup, keyFP)
	ObserveChannelAffinityUsageCacheByRelayFormat(ctx, &dto.Usage{PromptTokens: 1}, types.RelayFormatOpenAI)
	stats := GetChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP)
	require.EqualValues(t, 1, stats.Total)

	resetChannelAffinityUsageCacheStatsForTest()

	statsAfterReset := GetChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP)
	require.EqualValues(t, 0, statsAfterReset.Total)

	ctx2 := buildChannelAffinityStatsContextForTest(ruleName, usingGroup, keyFP)
	ObserveChannelAffinityUsageCacheByRelayFormat(ctx2, &dto.Usage{PromptTokens: 2}, types.RelayFormatOpenAI)
	statsSecondRun := GetChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP)
	require.EqualValues(t, 1, statsSecondRun.Total)
}
