package pkg

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type RegistrationResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type MyClaims struct {
	UserID string
	Name   string
	Email  string
	Role   string
	jwt.RegisteredClaims
}

func ParseTokenMyClaims(tokenStr string, secret []byte) (*MyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &MyClaims{}, func(t *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*MyClaims)
	if !ok {
		return nil, fmt.Errorf("invalid struture")
	}
	return claims, nil
}

func GenerateTokenMyClaims(claims *MyClaims, secret []byte) (string, error) {
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
