package token

import (
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueVerifyRoundTrip(t *testing.T) {
	j := NewJWT([]byte("test-secret"), time.Hour)
	want := shared.Principal{UserID: 42, Name: "alice", Role: shared.RoleTeacher}

	tok, err := j.Issue(want)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	got, err := j.Verify(tok)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestVerifyRejectsExpired(t *testing.T) {
	secret := []byte("test-secret")
	j := NewJWT(secret, time.Hour)

	// Craft a token that expired an hour ago, signed with the same secret.
	past := time.Now().Add(-time.Hour)
	expired := claims{
		Name: "bob",
		Role: shared.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			IssuedAt:  jwt.NewNumericDate(past.Add(-time.Hour)),
			ExpiresAt: jwt.NewNumericDate(past),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expired).SignedString(secret)
	require.NoError(t, err)

	_, err = j.Verify(raw)
	assert.Error(t, err)
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	signer := NewJWT([]byte("real-secret"), time.Hour)
	tok, err := signer.Issue(shared.Principal{UserID: 1, Name: "bob", Role: shared.RoleAdmin})
	require.NoError(t, err)

	attacker := NewJWT([]byte("other-secret"), time.Hour)
	_, err = attacker.Verify(tok)
	assert.Error(t, err)
}

func TestVerifyRejectsGarbage(t *testing.T) {
	j := NewJWT([]byte("test-secret"), time.Hour)
	_, err := j.Verify("not-a-jwt")
	assert.Error(t, err)
}
