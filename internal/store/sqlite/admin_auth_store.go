package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"opencrab/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

const (
	adminPasswordHashSettingKey   = "admin.password_hash"
	adminPasswordInitAtSettingKey = "admin.password_initialized_at"
	adminSessionSecretSettingKey  = "admin.session_secret"
	adminSecondaryEnabledKey      = "admin.secondary_enabled"
	adminSecondaryHashKey         = "admin.secondary_password_hash"
	adminSecondaryInitAtKey       = "admin.secondary_password_initialized_at"
)

var (
	ErrAdminPasswordAlreadyInitialized = errors.New("管理员密码已初始化")
	ErrAdminPasswordNotInitialized     = errors.New("管理员密码尚未初始化")
	ErrInvalidAdminPassword            = errors.New("密码错误")
	ErrAdminSessionSecretMissing       = errors.New("管理员会话密钥缺失")
	ErrSecondaryPasswordRequired       = errors.New("二级密码未通过校验")
	ErrSecondaryPasswordNotConfigured  = errors.New("二级密码尚未设置")
)

type adminSecondaryState struct {
	Enabled    bool
	Configured bool
	Hash       string
}

func GetAdminAuthState(ctx context.Context, db *sql.DB) (domain.AdminAuthState, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT key, value FROM system_settings WHERE key IN (?, ?, ?)`,
		adminPasswordHashSettingKey,
		adminPasswordInitAtSettingKey,
		adminSessionSecretSettingKey,
	)
	if err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("查询管理员认证配置失败: %w", err)
	}
	defer rows.Close()

	state := domain.AdminAuthState{}
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return domain.AdminAuthState{}, fmt.Errorf("读取管理员认证配置失败: %w", err)
		}
		switch key {
		case adminPasswordHashSettingKey:
			state.PasswordHash = value
		case adminPasswordInitAtSettingKey:
			state.InitializedAt = value
		case adminSessionSecretSettingKey:
			state.SessionSecret = value
		}
	}
	if err := rows.Err(); err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("遍历管理员认证配置失败: %w", err)
	}

	state.Initialized = strings.TrimSpace(state.PasswordHash) != ""
	return state, nil
}

func SetupAdminPassword(ctx context.Context, db *sql.DB, password string) (domain.AdminAuthState, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("开启管理员密码初始化事务失败: %w", err)
	}
	defer tx.Rollback()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("生成管理员密码哈希失败: %w", err)
	}
	secret, err := generateAdminSessionSecret()
	if err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("生成管理员会话密钥失败: %w", err)
	}
	now := time.Now().Format(time.RFC3339)
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO system_settings(key, value, updated_at) SELECT ?, ?, ? WHERE NOT EXISTS (SELECT 1 FROM system_settings WHERE key = ? LIMIT 1)`,
		adminPasswordHashSettingKey,
		string(hash),
		now,
		adminPasswordHashSettingKey,
	)
	if err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("写入管理员密码配置失败: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("读取管理员密码初始化状态失败: %w", err)
	}
	if rowsAffected == 0 {
		return domain.AdminAuthState{}, ErrAdminPasswordAlreadyInitialized
	}

	settings := []domain.UpdateSystemSettingInput{
		{Key: adminPasswordInitAtSettingKey, Value: now},
		{Key: adminSessionSecretSettingKey, Value: secret},
	}
	for _, item := range settings {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO system_settings(key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			item.Key,
			item.Value,
			now,
		); err != nil {
			return domain.AdminAuthState{}, fmt.Errorf("写入管理员认证配置失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("提交管理员密码初始化事务失败: %w", err)
	}

	return domain.AdminAuthState{
		Initialized:   true,
		PasswordHash:  string(hash),
		SessionSecret: secret,
		InitializedAt: now,
	}, nil
}

func VerifyAdminPassword(ctx context.Context, db *sql.DB, password string) (domain.AdminAuthState, error) {
	state, err := GetAdminAuthState(ctx, db)
	if err != nil {
		return domain.AdminAuthState{}, err
	}
	if !state.Initialized {
		return domain.AdminAuthState{}, ErrAdminPasswordNotInitialized
	}
	if strings.TrimSpace(state.SessionSecret) == "" {
		return domain.AdminAuthState{}, ErrAdminSessionSecretMissing
	}
	if err := bcrypt.CompareHashAndPassword([]byte(state.PasswordHash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return domain.AdminAuthState{}, ErrInvalidAdminPassword
		}
		return domain.AdminAuthState{}, fmt.Errorf("校验管理员密码失败: %w", err)
	}
	return state, nil
}

func ChangeAdminPassword(ctx context.Context, db *sql.DB, input domain.AdminPasswordChangeInput) (domain.AdminAuthState, error) {
	state, err := VerifyAdminPassword(ctx, db, input.CurrentPassword)
	if err != nil {
		return domain.AdminAuthState{}, err
	}
	if strings.TrimSpace(input.NewPassword) == "" || len(strings.TrimSpace(input.NewPassword)) < 8 {
		return domain.AdminAuthState{}, fmt.Errorf("新密码至少需要 8 个字符")
	}
	if input.NewPassword != input.ConfirmPassword {
		return domain.AdminAuthState{}, fmt.Errorf("两次输入的新密码不一致")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), 12)
	if err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("生成新密码哈希失败: %w", err)
	}
	newSecret, err := generateAdminSessionSecret()
	if err != nil {
		return domain.AdminAuthState{}, fmt.Errorf("生成新会话密钥失败: %w", err)
	}
	now := time.Now().Format(time.RFC3339)
	settings := []domain.UpdateSystemSettingInput{
		{Key: adminPasswordHashSettingKey, Value: string(hash)},
		{Key: adminSessionSecretSettingKey, Value: newSecret},
		{Key: adminPasswordInitAtSettingKey, Value: now},
	}
	for _, item := range settings {
		if _, err := db.ExecContext(ctx, `INSERT INTO system_settings(key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, item.Key, item.Value, now); err != nil {
			return domain.AdminAuthState{}, fmt.Errorf("更新管理员密码失败: %w", err)
		}
	}
	state.PasswordHash = string(hash)
	state.SessionSecret = newSecret
	state.InitializedAt = now
	return state, nil
}

func GetAdminSecondarySecurityState(ctx context.Context, db *sql.DB) (domain.AdminSecondarySecurityState, error) {
	state, err := getAdminSecondaryState(ctx, db)
	if err != nil {
		return domain.AdminSecondarySecurityState{}, err
	}
	return domain.AdminSecondarySecurityState{Enabled: state.Enabled, Configured: state.Configured}, nil
}

func UpdateAdminSecondaryPassword(ctx context.Context, db *sql.DB, input domain.AdminSecondaryPasswordUpdateInput) (domain.AdminSecondarySecurityState, error) {
	if _, err := VerifyAdminPassword(ctx, db, input.CurrentAdminPassword); err != nil {
		return domain.AdminSecondarySecurityState{}, err
	}
	state, err := getAdminSecondaryState(ctx, db)
	if err != nil {
		return domain.AdminSecondarySecurityState{}, err
	}
	now := time.Now().Format(time.RFC3339)
	if !input.Enabled {
		if _, err := db.ExecContext(ctx, `INSERT INTO system_settings(key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, adminSecondaryEnabledKey, "false", now); err != nil {
			return domain.AdminSecondarySecurityState{}, fmt.Errorf("关闭二级密码失败: %w", err)
		}
		return domain.AdminSecondarySecurityState{Enabled: false, Configured: state.Configured}, nil
	}

	shouldChangePassword := strings.TrimSpace(input.NewPassword) != "" || strings.TrimSpace(input.ConfirmPassword) != "" || !state.Configured
	if shouldChangePassword {
		if state.Configured {
			if err := verifySecondaryPasswordHash(state.Hash, input.CurrentSecondaryPassword); err != nil {
				return domain.AdminSecondarySecurityState{}, err
			}
		}
		if strings.TrimSpace(input.NewPassword) == "" || len(strings.TrimSpace(input.NewPassword)) < 8 {
			return domain.AdminSecondarySecurityState{}, fmt.Errorf("新密码至少需要 8 个字符")
		}
		if input.NewPassword != input.ConfirmPassword {
			return domain.AdminSecondarySecurityState{}, fmt.Errorf("两次输入的新密码不一致")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), 12)
		if err != nil {
			return domain.AdminSecondarySecurityState{}, fmt.Errorf("生成二级密码哈希失败: %w", err)
		}
		for _, item := range []domain.UpdateSystemSettingInput{{Key: adminSecondaryHashKey, Value: string(hash)}, {Key: adminSecondaryInitAtKey, Value: now}} {
			if _, err := db.ExecContext(ctx, `INSERT INTO system_settings(key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, item.Key, item.Value, now); err != nil {
				return domain.AdminSecondarySecurityState{}, fmt.Errorf("保存二级密码失败: %w", err)
			}
		}
		state.Configured = true
		state.Hash = string(hash)
	}

	if !state.Configured {
		return domain.AdminSecondarySecurityState{}, ErrSecondaryPasswordNotConfigured
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO system_settings(key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, adminSecondaryEnabledKey, "true", now); err != nil {
		return domain.AdminSecondarySecurityState{}, fmt.Errorf("开启二级密码失败: %w", err)
	}
	return domain.AdminSecondarySecurityState{Enabled: true, Configured: true}, nil
}

func VerifySecondaryPassword(ctx context.Context, db *sql.DB, password string) error {
	state, err := getAdminSecondaryState(ctx, db)
	if err != nil {
		return err
	}
	if !state.Enabled {
		return nil
	}
	if !state.Configured {
		return ErrSecondaryPasswordNotConfigured
	}
	return verifySecondaryPasswordHash(state.Hash, password)
}

func getAdminSecondaryState(ctx context.Context, db *sql.DB) (adminSecondaryState, error) {
	rows, err := db.QueryContext(ctx, `SELECT key, value FROM system_settings WHERE key IN (?, ?, ?)`, adminSecondaryEnabledKey, adminSecondaryHashKey, adminSecondaryInitAtKey)
	if err != nil {
		return adminSecondaryState{}, fmt.Errorf("查询二级密码状态失败: %w", err)
	}
	defer rows.Close()
	state := adminSecondaryState{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return adminSecondaryState{}, fmt.Errorf("读取二级密码状态失败: %w", err)
		}
		switch key {
		case adminSecondaryEnabledKey:
			state.Enabled = strings.EqualFold(strings.TrimSpace(value), "true")
		case adminSecondaryHashKey:
			state.Hash = value
			state.Configured = strings.TrimSpace(value) != ""
		}
	}
	if err := rows.Err(); err != nil {
		return adminSecondaryState{}, fmt.Errorf("遍历二级密码状态失败: %w", err)
	}
	return state, nil
}

func verifySecondaryPasswordHash(hash string, password string) error {
	if strings.TrimSpace(password) == "" {
		return ErrSecondaryPasswordRequired
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrSecondaryPasswordRequired
		}
		return fmt.Errorf("校验二级密码失败: %w", err)
	}
	return nil
}

func generateAdminSessionSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
