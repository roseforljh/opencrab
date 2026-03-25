package controller

import (
	"os"
	"regexp"
	"testing"
)

func TestModeFlagsRemovedFromOperationSettings(t *testing.T) {
	files := []string{
		"../web/src/pages/Setting/Operation/SettingsGeneral.jsx",
		"../web/src/components/settings/OperationSetting.jsx",
	}

	patterns := []string{
		`SelfUseModeEnabled`,
		`DemoSiteEnabled`,
		`自用模式`,
		`演示站点模式`,
	}

	for _, file := range files {
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
				t.Fatalf("legacy mode pattern still present in %s: %s", file, pattern)
			}
		}
	}
}
