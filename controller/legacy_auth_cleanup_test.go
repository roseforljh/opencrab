package controller

import (
	"os"
	"strings"
	"testing"
)

func TestLegacyAuthHandlersRemoved(t *testing.T) {
	testCases := []struct {
		filePath string
		symbols  []string
	}{
		{
			filePath: "user.go",
			symbols: []string{
				"type LoginRequest struct",
				"func Login(c *gin.Context)",
				"func Register(c *gin.Context)",
				"func EmailBind(c *gin.Context)",
			},
		},
		{
			filePath: "misc.go",
			symbols: []string{
				"func SendEmailVerification(c *gin.Context)",
				"func SendPasswordResetEmail(c *gin.Context)",
				"type PasswordResetRequest struct",
				"func ResetPassword(c *gin.Context)",
			},
		},
	}

	for _, tc := range testCases {
		content, err := os.ReadFile(tc.filePath)
		if err != nil {
			t.Fatalf("failed to read %s: %v", tc.filePath, err)
		}
		for _, symbol := range tc.symbols {
			if strings.Contains(string(content), symbol) {
				t.Fatalf("legacy auth symbol still present in %s: %s", tc.filePath, symbol)
			}
		}
	}
}
