package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT payload for DevConnect tokens.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// getSecret returns the JWT signing key from environment.
func getSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "devconnect-default-secret-change-me"
	}
	return []byte(secret)
}

// GenerateToken creates a signed JWT for the given user.
// The token expires after 24 hours.
func GenerateToken(userID, username string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "devconnect",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecret())
}

// ValidateToken parses and validates a JWT string, returning the claims.
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return getSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// AuthRequired is a Gin middleware that enforces JWT authentication.
// It reads the Authorization header (Bearer <token>), validates the JWT,
// and sets "user_id" and "username" in the Gin context.
// Returns 401 if the token is missing or invalid.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header must be: Bearer <token>",
			})
			return
		}

		claims, err := ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}

		// Set user info in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// OptionalAuth is a Gin middleware that attempts JWT validation but does
// NOT block the request if the token is missing. Useful for WebSocket
// endpoints where auth is optional but preferred.
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check header first
		authHeader := c.GetHeader("Authorization")
		tokenStr := ""

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
				tokenStr = parts[1]
			}
		}

		// Fallback: check query param (for WebSocket connections)
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}

		if tokenStr != "" {
			claims, err := ValidateToken(tokenStr)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("username", claims.Username)
			}
		}

		c.Next()
	}
}
