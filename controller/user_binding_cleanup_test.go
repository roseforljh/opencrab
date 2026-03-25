package controller

import (
	"os"
	"regexp"
	"testing"
)

func TestLegacyUserBindingFieldsRemoved(t *testing.T) {
	content, err := os.ReadFile("user.go")
	if err != nil {
		t.Fatalf("failed to read user.go: %v", err)
	}

	disallowedPatterns := []string{
		`json:"[^\"]*_id"`,
		`gorm:"column:[^\"]*_id`,
		`func AdminClearUserBinding\(c \*gin\.Context\)`,
	}

	for _, pattern := range disallowedPatterns {
		matched, matchErr := regexp.MatchString(pattern, string(content))
		if matchErr != nil {
			t.Fatalf("invalid regexp %q: %v", pattern, matchErr)
		}
		if matched {
			t.Fatalf("legacy user binding pattern still present in user.go: %s", pattern)
		}
	}
}
