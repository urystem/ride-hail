package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"taxi-hailing/pkg"
)

type ctxKey string

const userCtxKey ctxKey = "user"

// for both services(driver and passenger)
func authMiddleware(next http.Handler, secret []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claim, err := getClaim(r, secret)
		if err != nil {
			errorWrite(w, http.StatusBadRequest, err)
			return
		}
		// кладем userID в контекст
		ctx := context.WithValue(r.Context(), userCtxKey, claim)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getClaim(r *http.Request, secret []byte) (*pkg.MyClaims, error) {
	// 1. Получаем заголовок Authorization
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	// 2. Проверяем что формат Bearer TOKEN
	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid Authorization header")
	}

	tokenStr := parts[1]

	// 3. Парсим токен
	return pkg.ParseTokenMyClaims(tokenStr, secret)
}
