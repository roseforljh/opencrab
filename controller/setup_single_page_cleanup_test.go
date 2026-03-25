package controller

import (
	"os"
	"regexp"
	"testing"
)

func TestSetupWizardIsSinglePagePinInit(t *testing.T) {
	content, err := os.ReadFile("../web/src/components/setup/SetupWizard.jsx")
	if err != nil {
		t.Fatalf("failed to read SetupWizard.jsx: %v", err)
	}

	disallowedPatterns := []string{
		`\bSteps\b`,
		`currentStep`,
		`DatabaseStep`,
		`UsageModeStep`,
		`CompleteStep`,
		`StepNavigation`,
		`usageMode`,
		`管理员账号`,
		`使用模式`,
		`数据库检查`,
		`完成初始化`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy setup wizard pattern still present in SetupWizard.jsx: %s", pattern)
		}
	}
}

func TestAdminStepOnlyCollectsPin(t *testing.T) {
	content, err := os.ReadFile("../web/src/components/setup/components/steps/AdminStep.jsx")
	if err != nil {
		t.Fatalf("failed to read AdminStep.jsx: %v", err)
	}

	disallowedPatterns := []string{
		`field='username'`,
		`管理员用户名`,
		`label=\{t\('用户名'\)\}`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy admin setup pattern still present in AdminStep.jsx: %s", pattern)
		}
	}
}

func TestSetupBackendOnlyAcceptsPinInit(t *testing.T) {
	content, err := os.ReadFile("setup.go")
	if err != nil {
		t.Fatalf("failed to read setup.go: %v", err)
	}

	disallowedPatterns := []string{
		`Username\s+string\s+` + "`json:\"username\"`",
		`Password\s+string\s+` + "`json:\"password\"`",
		`ConfirmPassword\s+string\s+` + "`json:\"confirmPassword\"`",
		`SelfUseModeEnabled\s+bool\s+` + "`json:\"SelfUseModeEnabled\"`",
		`DemoSiteEnabled\s+bool\s+` + "`json:\"DemoSiteEnabled\"`",
		`UpdateOption\(\"SelfUseModeEnabled\"`,
		`UpdateOption\(\"DemoSiteEnabled\"`,
		`保存自用模式设置失败`,
		`保存演示站点模式设置失败`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy setup backend pattern still present in setup.go: %s", pattern)
		}
	}
}
