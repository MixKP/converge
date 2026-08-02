package token

import (
	"fmt"
	"strconv"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	secret []byte
	ttl    time.Duration
}

func NewJWT(secret []byte, ttl time.Duration) *JWT {
	if len(secret) == 0 {
		panic("token.NewJWT: empty secret")
	}
	if ttl <= 0 {
		panic("token.NewJWT: ttl must be positive")
	}
	return &JWT{secret: secret, ttl: ttl}
}

type claims struct {
	Name string      `json:"name"`
	Role shared.Role `json:"role"`
	jwt.RegisteredClaims
}

func (j *JWT) Issue(p shared.Principal) (string, error) {
	now := time.Now()
	c := claims{
		Name: p.Name,
		Role: p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(p.UserID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return tok, nil
}

func (j *JWT) Verify(raw string) (shared.Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return shared.Principal{}, fmt.Errorf("verify token: %w", err)
	}

	id, err := strconv.Atoi(c.Subject)
	if err != nil {
		return shared.Principal{}, fmt.Errorf("invalid subject %q: %w", c.Subject, err)
	}
	role, err := shared.ParseRole(string(c.Role))
	if err != nil {
		return shared.Principal{}, err
	}
	return shared.Principal{UserID: id, Name: c.Name, Role: role}, nil
}
