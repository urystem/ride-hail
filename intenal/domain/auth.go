package domain

import "github.com/golang-jwt/jwt/v5"

type MyClaims struct {
	PassengerID string
	Name        string
	Email       string
	Role        string
	jwt.RegisteredClaims
}
