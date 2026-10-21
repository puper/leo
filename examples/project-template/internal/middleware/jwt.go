package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	"server/internal/config"
	"server/internal/derrors"
	dto "server/internal/model/dto"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserId   int64  `json:"userId,string"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateTokenWithUserID(userID int64, username string) (string, error) {
	cfg := config.Instance()
	if cfg == nil || cfg.App == nil || cfg.App.Auth == nil {
		return "", errors.New("auth config not found")
	}
	appCfg := cfg.App
	authCfg := appCfg.Auth
	claims := Claims{
		UserId:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(authCfg.JwtExpire) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := authCfg.SigningKey()
	if err != nil {
		return "", err
	}
	if userID <= 0 || strings.TrimSpace(username) == "" {
		return "", errors.New("valid user identity is required")
	}
	return token.SignedString(secret)
}

func abortWithDetailedError(c *gin.Context, err *derrors.DetailedError) {
	dto.NewResp(c).Error(err)
	c.Abort()
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if !strings.HasPrefix(tokenString, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer ")) == "" {
			abortWithDetailedError(c, derrors.NewUnAuthError(errors.New("missing bearer token")))
			return
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		cfg := config.Instance()
		if cfg == nil || cfg.App == nil || cfg.App.Auth == nil {
			abortWithDetailedError(c, derrors.NewInternalError(errors.New("auth config not found")).SetMessage("配置错误: 鉴权配置缺失"))
			return
		}

		appCfg := cfg.App
		authCfg := appCfg.Auth

		secret, err := authCfg.SigningKey()
		if err != nil {
			abortWithDetailedError(c, derrors.NewInternalError(err))
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())

		if err != nil || !token.Valid {
			abortWithDetailedError(c, derrors.NewUnAuthError(errors.New("invalid token")).SetMessage("无效的 token"))
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && claims.UserId > 0 && strings.TrimSpace(claims.Username) != "" {
			c.Set("userId", claims.UserId)
			c.Set("username", claims.Username)
			ctx := context.WithValue(c.Request.Context(), "userId", claims.UserId)
			ctx = context.WithValue(ctx, "username", claims.Username)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		} else {
			abortWithDetailedError(c, derrors.NewUnAuthError(errors.New("invalid token claims")).SetMessage("无效的 token claims"))
		}
	}
}
