package controller

import (
	"os"
	"regexp"
	"testing"
)

func TestSettingPageDoesNotReferenceRatioFeatures(t *testing.T) {
	content, err := os.ReadFile("../web/src/pages/Setting/index.jsx")
	if err != nil {
		t.Fatalf("failed to read Setting index.jsx: %v", err)
	}

	disallowedPatterns := []string{
		`RatioSetting`,
		`模型倍率`,
		`分组相关设置`,
		`价格设置`,
		`上游倍率同步`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy ratio pattern still present in Setting index.jsx: %s", pattern)
		}
	}
}

func TestPersonalSettingsDoNotExposeUnsetRatioToggle(t *testing.T) {
	content, err := os.ReadFile("../web/src/components/settings/PersonalSetting.jsx")
	if err != nil {
		t.Fatalf("failed to read PersonalSetting.jsx: %v", err)
	}

	disallowedPatterns := []string{
		`accept_unset_model_ratio_model`,
		`acceptUnsetModelRatioModel`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy personal setting ratio pattern still present in PersonalSetting.jsx: %s", pattern)
		}
	}
}

func TestChannelSelectorModalDoesNotOfferRatioConfigEndpoint(t *testing.T) {
	content, err := os.ReadFile("../web/src/components/settings/ChannelSelectorModal.jsx")
	if err != nil {
		t.Fatalf("failed to read ChannelSelectorModal.jsx: %v", err)
	}

	disallowedPatterns := []string{
		`/api/ratio_config`,
		`ratio_config`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy ratio config endpoint pattern still present in ChannelSelectorModal.jsx: %s", pattern)
		}
	}
}

func TestApiRouterDoesNotRegisterRatioRoutes(t *testing.T) {
	content, err := os.ReadFile("../router/api-router.go")
	if err != nil {
		t.Fatalf("failed to read api-router.go: %v", err)
	}

	disallowedPatterns := []string{
		`/ratio_config`,
		`/ratio_sync`,
		`/rest_model_ratio`,
		`GetRatioConfig`,
		`FetchUpstreamRatios`,
		`GetSyncableChannels`,
		`ResetModelRatio`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy ratio route pattern still present in api-router.go: %s", pattern)
		}
	}
}

func TestControllerPackageDoesNotContainLegacyRatioHandlers(t *testing.T) {
	files := []string{
		"pricing.go",
		"ratio_config.go",
		"ratio_sync.go",
	}

	patternsByFile := map[string][]string{
		"pricing.go": {
			`func ResetModelRatio\(`,
		},
		"ratio_config.go": {
			`func GetRatioConfig\(`,
		},
		"ratio_sync.go": {
			`func FetchUpstreamRatios\(`,
			`func GetSyncableChannels\(`,
			`/api/ratio_config`,
		},
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		for _, pattern := range patternsByFile[file] {
			matched, matchErr := regexp.MatchString(pattern, string(content))
			if matchErr != nil {
				t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
			}
			if matched {
				t.Fatalf("legacy ratio handler pattern still present in %s: %s", file, pattern)
			}
		}
	}
}

func TestOptionMapDoesNotRegisterRatioOptions(t *testing.T) {
	content, err := os.ReadFile("../model/option.go")
	if err != nil {
		t.Fatalf("failed to read model/option.go: %v", err)
	}

	disallowedPatterns := []string{
		`common\.OptionMap\["ModelRatio"\]`,
		`common\.OptionMap\["ModelPrice"\]`,
		`common\.OptionMap\["CacheRatio"\]`,
		`common\.OptionMap\["CreateCacheRatio"\]`,
		`common\.OptionMap\["GroupRatio"\]`,
		`common\.OptionMap\["GroupGroupRatio"\]`,
		`common\.OptionMap\["CompletionRatio"\]`,
		`common\.OptionMap\["ImageRatio"\]`,
		`common\.OptionMap\["AudioRatio"\]`,
		`common\.OptionMap\["AudioCompletionRatio"\]`,
		`common\.OptionMap\["ExposeRatioEnabled"\]`,
		`case "ModelRatio":`,
		`case "ModelPrice":`,
		`case "CacheRatio":`,
		`case "CreateCacheRatio":`,
		`case "GroupRatio":`,
		`case "GroupGroupRatio":`,
		`case "CompletionRatio":`,
		`case "ImageRatio":`,
		`case "AudioRatio":`,
		`case "AudioCompletionRatio":`,
		`case "ExposeRatioEnabled":`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy ratio option pattern still present in model/option.go: %s", pattern)
		}
	}
}

func TestRuntimeFilesDoNotReferenceRatioHelpers(t *testing.T) {
	checks := map[string][]string{
		"group.go": {
			`GetGroupRatioCopy\(`,
			`GetUserGroupRatio\(`,
		},
		"model.go": {
			`GetModelRatioOrPrice\(`,
		},
		"../middleware/auth.go": {
			`ContainsGroupRatio\(`,
		},
	}

	for file, patterns := range checks {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		for _, pattern := range patterns {
			matched, matchErr := regexp.MatchString(pattern, string(content))
			if matchErr != nil {
				t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
			}
			if matched {
				t.Fatalf("legacy runtime ratio pattern still present in %s: %s", file, pattern)
			}
		}
	}
}

func TestBillingChainDoesNotReferenceRatioSetting(t *testing.T) {
	checks := map[string][]string{
		"../relay/helper/price.go": {
			`ratio_setting\.`,
			`ContainPriceOrRatio\(`,
		},
		"../service/group.go": {
			`ratio_setting\.`,
			`GetUserGroupRatio\(`,
		},
		"../service/task_billing.go": {
			`ratio_setting\.`,
		},
		"../service/quota.go": {
			`ratio_setting\.`,
		},
		"../model/pricing.go": {
			`ratio_setting\.`,
		},
	}

	for file, patterns := range checks {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		for _, pattern := range patterns {
			matched, matchErr := regexp.MatchString(pattern, string(content))
			if matchErr != nil {
				t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
			}
			if matched {
				t.Fatalf("legacy billing ratio pattern still present in %s: %s", file, pattern)
			}
		}
	}
}

func TestOptionControllerAndFrontendDoNotExposeRatioConfig(t *testing.T) {
	checks := map[string][]string{
		"option.go": {
			`CompletionRatioMeta`,
			`ratio_setting\.`,
			`CheckGroupRatio\(`,
			`UpdateImageRatioByJSONString\(`,
			`UpdateAudioRatioByJSONString\(`,
			`UpdateAudioCompletionRatioByJSONString\(`,
			`UpdateCreateCacheRatioByJSONString\(`,
		},
		"../web/src/constants/common.constant.js": {
			`/api/ratio_config`,
		},
	}

	for file, patterns := range checks {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		for _, pattern := range patterns {
			matched, matchErr := regexp.MatchString(pattern, string(content))
			if matchErr != nil {
				t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
			}
			if matched {
				t.Fatalf("legacy ratio config pattern still present in %s: %s", file, pattern)
			}
		}
	}
}

func TestMainAndFrontendDoNotLoadRatioModules(t *testing.T) {
	checks := map[string][]string{
		"../main.go": {
			`ratio_setting`,
			`InitRatioSettings\(`,
		},
		"../web/src/components/settings/RatioSetting.jsx": {
			`pages/Setting/Ratio/`,
			`ModelRatioSettings`,
			`GroupRatioSettings`,
			`UpstreamRatioSync`,
		},
	}

	for file, patterns := range checks {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		for _, pattern := range patterns {
			matched, matchErr := regexp.MatchString(pattern, string(content))
			if matchErr != nil {
				t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
			}
			if matched {
				t.Fatalf("legacy ratio module pattern still present in %s: %s", file, pattern)
			}
		}
	}
}

func TestRemainingBackendFilesDoNotReferenceRatioSetting(t *testing.T) {
	checks := map[string][]string{
		"pricing.go": {
			`ratio_setting\.`,
		},
		"subscription.go": {
			`GetGroupRatioCopy\(`,
		},
		"../relay/compatible_handler.go": {
			`ContainsAudioRatio\(`,
			`ContainsAudioCompletionRatio\(`,
		},
	}

	for file, patterns := range checks {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		for _, pattern := range patterns {
			matched, matchErr := regexp.MatchString(pattern, string(content))
			if matchErr != nil {
				t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
			}
			if matched {
				t.Fatalf("legacy backend ratio pattern still present in %s: %s", file, pattern)
			}
		}
	}
}

func TestLastRatioReferencesAndPagesAreRemoved(t *testing.T) {
	checks := map[string][]string{
		"../middleware/distributor.go": {
			`FormatMatchingModelName\(`,
			`WithCompactModelSuffix\(`,
		},
		"../model/channel_cache.go": {
			`FormatMatchingModelName\(`,
		},
		"channel-test.go": {
			`CompactModelSuffix`,
			`WithCompactModelSuffix\(`,
		},
		"../web/src/components/settings/RatioSetting.jsx": {
			`pages/Setting/Ratio/`,
		},
	}

	for file, patterns := range checks {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		for _, pattern := range patterns {
			matched, matchErr := regexp.MatchString(pattern, string(content))
			if matchErr != nil {
				t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
			}
			if matched {
				t.Fatalf("legacy final ratio pattern still present in %s: %s", file, pattern)
			}
		}
	}
}
