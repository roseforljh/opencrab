package i18n

import (
	"os"
	"regexp"
	"testing"
)

func TestLegacyUserBindingI18nKeysRemoved(t *testing.T) {
	content, err := os.ReadFile("keys.go")
	if err != nil {
		t.Fatalf("failed to read keys.go: %v", err)
	}

	disallowedPattern := `MsgUser[A-Za-z]+IdEmpty`
	matched, matchErr := regexp.MatchString(disallowedPattern, string(content))
	if matchErr != nil {
		t.Fatalf("invalid regexp %q: %v", disallowedPattern, matchErr)
	}
	if matched {
		t.Fatalf("legacy binding i18n pattern still present in keys.go: %s", disallowedPattern)
	}
}
