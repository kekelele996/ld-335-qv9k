package util

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明：sub=client_id，role=client.role。
type Claims struct {
	ClientID uint   `json:"client_id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	TokenType string `json:"token_type"` // admin / service
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT。
func GenerateToken(secret string, expireHours int, clientID uint, name, role, tokenType string) (string, error) {
	now := time.Now()
	claims := Claims{
		ClientID:  clientID,
		Name:      name,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "gbinsureapi",
			Subject:   fmtUint(clientID),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken 解析 JWT。
func ParseToken(secret, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func fmtUint(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
