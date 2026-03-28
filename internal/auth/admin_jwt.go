package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AdminClaims are the JWT claims used by the admin backend.
type AdminClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AdminJWTManager handles minting and validation for admin sessions.
type AdminJWTManager struct {
	secret     []byte
	issuer     string
	expiration time.Duration
}

func NewAdminJWTManager(secret, issuer string, expiration time.Duration) *AdminJWTManager {
	return &AdminJWTManager{
		secret:     []byte(secret),
		issuer:     issuer,
		expiration: expiration,
	}
}

func (m *AdminJWTManager) IssueToken(username string) (string, error) {
	now := time.Now().UTC()
	claims := AdminClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   "meshchat-admin-session",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *AdminJWTManager) ParseToken(raw string) (*AdminClaims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &AdminClaims{}, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*AdminClaims)
	if !ok || !parsed.Valid || claims.Subject != "meshchat-admin-session" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
