package web

import (
	"net/http"
	"strings"

	"github.com/RinZ5/converge/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

const principalContextKey = "auth.principal"

type tokenVerifier interface {
	Verify(raw string) (shared.Principal, error)
}

func AuthMiddleware(v tokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := bearerToken(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed Authorization header"})
			return
		}
		p, err := v.Verify(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(principalContextKey, p)
		c.Next()
	}
}

func RequireRole(roles ...shared.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := PrincipalFrom(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		for _, r := range roles {
			if p.Role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

func PrincipalFrom(c *gin.Context) (shared.Principal, bool) {
	v, exists := c.Get(principalContextKey)
	if !exists {
		return shared.Principal{}, false
	}
	p, ok := v.(shared.Principal)
	return p, ok
}

func bearerToken(c *gin.Context) (string, bool) {
	h := c.GetHeader("Authorization")
	if h == "" {
		return "", false
	}
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(h[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}
