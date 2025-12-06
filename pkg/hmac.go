package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func HashPassword(password string, secret []byte) (string, error) {
	mac := hmac.New(sha256.New, secret)
	_, err := mac.Write([]byte(password))
	if err != nil {
		return "", err
	}
	//sum []byte қайтарады, әрі prefix қояды, бірақ парольға префикс керек емес
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func CheckPassword(password, storedHash string, secret []byte) (bool, error) {
	hash, err := HashPassword(password, secret)
	if err != nil {
		return false, err
	}
	// return hash == storedHash
	return hmac.Equal([]byte(hash), []byte(storedHash)), nil
}
