package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"server/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestSecret(t *testing.T) {
	original := config.Instance()
	t.Cleanup(func() { config.SetInstance(original) })
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}
	secretB64 := base64.URLEncoding.EncodeToString(secret)
	testCfg := &config.Config{
		App: &config.App{
			Env:      "dev",
			TimeZone: "Asia/Shanghai",
			Auth: &config.AuthConfig{
				JwtSecret: secretB64,
				JwtExpire: 60,
			},
		},
	}
	config.SetInstance(testCfg)
}

func TestAuthMiddleware_TokenSources(t *testing.T) {
	setupTestSecret(t)

	token, err := GenerateTokenWithUserID(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name        string
		url         string
		wantStatus  int
		description string
	}{
		{
			name:        "Token in Authorization header (SECURE)",
			url:         "/test",
			wantStatus:  http.StatusOK,
			description: "Should accept token in Authorization header",
		},
		{
			name:        "Token in query parameter (rejected)",
			url:         "/test?token=" + token,
			wantStatus:  http.StatusUnauthorized,
			description: " Should NOT accept token in query parameter due to security risks (logged URLs, referrer leakage)",
		},
		{
			name:        "No token (should fail)",
			url:         "/test",
			wantStatus:  http.StatusUnauthorized,
			description: "Should reject request without token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", tt.url, nil)

			if tt.name == "Token in Authorization header (SECURE)" {
				c.Request.Header.Set("Authorization", "Bearer "+token)
			}

			handler := AuthMiddleware()
			handler(c)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: status = %d, want %d", tt.description, w.Code, tt.wantStatus)
			}
		})
	}
}

func TestAuthMiddleware_CookieToken(t *testing.T) {
	setupTestSecret(t)

	token, err := GenerateTokenWithUserID(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.AddCookie(&http.Cookie{Name: "token", Value: token})

	handler := AuthMiddleware()
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Cookie token must be rejected: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareRejectsInvalidClaims(t *testing.T) {
	setupTestSecret(t)
	key, err := config.Instance().App.Auth.SigningKey()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		method jwt.SigningMethod
		claims Claims
	}{
		{"wrong algorithm", jwt.SigningMethodHS384, Claims{UserId: 1, Username: "user", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}},
		{"expired", jwt.SigningMethodHS256, Claims{UserId: 1, Username: "user", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}},
		{"missing expiry", jwt.SigningMethodHS256, Claims{UserId: 1, Username: "user"}},
		{"missing identity", jwt.SigningMethodHS256, Claims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(tc.method, tc.claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			r := gin.New()
			r.GET("/test", AuthMiddleware(), func(c *gin.Context) { t.Error("invalid token reached handler") })
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			if res.Code != http.StatusUnauthorized {
				t.Errorf("got %d, want 401", res.Code)
			}
		})
	}
}
