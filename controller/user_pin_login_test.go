package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/roseforljh/opencrab/common"
	projecti18n "github.com/roseforljh/opencrab/i18n"
	"github.com/roseforljh/opencrab/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type pinLoginAPIResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func setupUserControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	if err := projecti18n.Init(); err != nil {
		t.Fatalf("failed to init i18n: %v", err)
	}
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db

	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate test tables: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func createRootUserWithPin(t *testing.T, db *gorm.DB, pin string) *model.User {
	t.Helper()

	hashedPassword, err := common.Password2Hash("temporary-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &model.User{
		Username:    "root",
		Password:    hashedPassword,
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Root User",
		Group:       "default",
	}
	if err := user.SetPin(pin); err != nil {
		t.Fatalf("failed to set pin: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create root user: %v", err)
	}
	return user
}

func TestPinLoginLogsInDirectly(t *testing.T) {
	db := setupUserControllerTestDB(t)
	user := createRootUserWithPin(t, db, "1234")

	payload, err := common.Marshal(map[string]string{"pin": "1234"})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	router := gin.New()
	store := cookie.NewStore([]byte("test-session-secret"))
	router.Use(sessions.Sessions("session", store))
	router.POST("/api/user/pin-login", PinLogin)
	req := httptest.NewRequest(http.MethodPost, "/api/user/pin-login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response pinLoginAPIResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}
	if _, exists := response.Data["require_2fa"]; exists {
		t.Fatalf("expected direct login response, got 2fa requirement: %s", recorder.Body.String())
	}
	if response.Data["username"] != user.Username {
		t.Fatalf("expected username %q, got %#v", user.Username, response.Data["username"])
	}
}
