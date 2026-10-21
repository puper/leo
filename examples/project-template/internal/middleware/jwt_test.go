package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"server/internal/config"
)

func TestGenerateToken_ShortSecret(t *testing.T) {
	shortSecret := make([]byte, 4)
	if _, err := rand.Read(shortSecret); err != nil {
		t.Fatalf("Failed to generate random secret: %v", err)
	}
	shortSecretB64 := base64.URLEncoding.EncodeToString(shortSecret)

	originalCfg := initTestConfig(shortSecretB64, 60)
	defer restoreTestConfig(originalCfg)

	_, err := GenerateTokenWithUserID(1, "testuser")
	if err == nil {
		t.Errorf("GenerateTokenWithUserID should reject short secret (< 32 bytes for HS256), but it succeeded")
	}
}

func TestGenerateToken_SecretMinimumLength(t *testing.T) {
	tests := []struct {
		name      string
		secretLen int
		wantErr   bool
	}{
		{"16 bytes should fail", 16, true},
		{"31 bytes should fail", 31, true},
		{"32 bytes should pass", 32, false},
		{"64 bytes should pass", 64, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := make([]byte, tt.secretLen)
			if _, err := rand.Read(secret); err != nil {
				t.Fatalf("Failed to generate random secret: %v", err)
			}
			secretB64 := base64.URLEncoding.EncodeToString(secret)

			originalCfg := initTestConfig(secretB64, 60)
			defer restoreTestConfig(originalCfg)

			_, err := GenerateTokenWithUserID(1, "testuser")

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTokenWithUserID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func initTestConfig(jwtSecret string, jwtExpire int) *config.Config {
	original := config.Instance()
	testCfg := &config.Config{
		App: &config.App{
			Env:      "dev",
			TimeZone: "Asia/Shanghai",
			Auth: &config.AuthConfig{
				JwtSecret: jwtSecret,
				JwtExpire: jwtExpire,
			},
		},
	}
	config.SetInstance(testCfg)
	return original
}

func restoreTestConfig(original *config.Config) {
	config.SetInstance(original)
}
