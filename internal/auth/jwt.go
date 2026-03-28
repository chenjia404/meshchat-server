package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the stable JWT claims used by HTTP and WebSocket auth.
type Claims struct {
	UserID uint64 `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTManager handles token minting and validation.
type JWTManager struct {
	secret     []byte
	issuer     string
	expiration time.Duration
}

func NewJWTManager(secret, issuer string, expiration time.Duration) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		issuer:     issuer,
		expiration: expiration,
	}
}

func (m *JWTManager) IssueToken(userID uint64) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   "meshchat-session",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) ParseToken(raw string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
