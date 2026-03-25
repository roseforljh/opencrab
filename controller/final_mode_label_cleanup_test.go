package controller

import (
	"os"
	"regexp"
	"testing"
)

func TestFinalModeLabelsRemovedFromFrontend(t *testing.T) {
	files := []string{
		"../web/src/components/settings/SystemSetting.jsx",
		"../web/src/components/layout/headerbar/HeaderLogo.jsx",
		"../web/src/components/layout/headerbar/index.jsx",
		"../web/src/components/layout/headerbar/UserArea.jsx",
		"../web/src/components/layout/headerbar/ActionButtons.jsx",
		"../web/src/components/layout/Footer.jsx",
		"../web/src/pages/Home/index.jsx",
		"../web/src/hooks/common/useHeaderBar.js",
	}

	patterns := []string{
		`自用模式`,
		`演示站点`,
		`个人自用模式`,
		`isSelfUseMode`,
		`isDemoSiteMode`,
		`self_use_mode_enabled`,
		`demo_site_enabled`,
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
				t.Fatalf("legacy final mode pattern still present in %s: %s", file, pattern)
			}
		}
	}
}
