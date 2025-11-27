package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// func main() {
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
// 		"sub":   "user-id-uuid", // UUID пользователя из базы
// 		"email": "admin@example.com",
// 		"role":  "ADMIN",
// 		"exp":   time.Now().Add(time.Hour * 24).Unix(),
// 	})
// 	tokenString, _ := token.SignedString([]byte("your-secret-key"))
// 	fmt.Println(tokenString)
// }

func HashPassword(password, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(password))
	//sum []byte қайтарады, әрі prefix қояды, бірақ парольға префикс керек емес
	return hex.EncodeToString(mac.Sum(nil))
}

func CheckPassword(password, secret, storedHash string) bool {
	hash := HashPassword(password, secret)
	// return hash == storedHash
	return hmac.Equal([]byte(hash), []byte(storedHash))
}

func main() {
	// token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
	// 	Subject: "",
	// })

	// sec := "it-is-my-secret-key-and-it-is-mine"
	// pass := "admin123"
	// has := HashPassword(pass, sec)
	// fmt.Println(has)
	// fmt.Println(CheckPassword(pass, sec, has))
}
