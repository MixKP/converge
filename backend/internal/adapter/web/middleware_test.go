package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// stubVerifier verifies exactly one token string, returning the given principal.
type stubVerifier struct {
	valid     string
	principal shared.Principal
}

func (s stubVerifier) Verify(raw string) (shared.Principal, error) {
	if raw == s.valid {
		return s.principal, nil
	}
	return shared.Principal{}, errors.New("invalid token")
}

func init() { gin.SetMode(gin.TestMode) }

func newRouter(v tokenVerifier, roles ...shared.Role) *gin.Engine {
	r := gin.New()
	r.GET("/protected", AuthMiddleware(v), RequireRole(roles...), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	v := stubVerifier{valid: "good", principal: shared.Principal{UserID: 1, Role: shared.RoleAdmin}}
	r := newRouter(v, shared.RoleAdmin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	v := stubVerifier{valid: "good", principal: shared.Principal{UserID: 1, Role: shared.RoleAdmin}}
	r := newRouter(v, shared.RoleAdmin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer bad")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireRole_WrongRole(t *testing.T) {
	v := stubVerifier{valid: "good", principal: shared.Principal{UserID: 1, Role: shared.RoleTeacher}}
	r := newRouter(v, shared.RoleAdmin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer good")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireRole_Allowed(t *testing.T) {
	v := stubVerifier{valid: "good", principal: shared.Principal{UserID: 1, Role: shared.RoleAdmin}}
	r := newRouter(v, shared.RoleAdmin, shared.RoleTeacher)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer good")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
