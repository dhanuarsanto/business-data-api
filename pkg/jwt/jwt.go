package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey []byte
var tokenDuration time.Duration = 24 * time.Hour

func InitJWT(secret string, duration time.Duration) {
	secretKey = []byte(secret)
	if duration > 0 {
		tokenDuration = duration
	}
}

func GenerateToken(userID int, username string, rules string, tenant string) (string, error) {
	if len(secretKey) == 0 {
		return "", errors.New("JWT secret belum diinisialisasi")
	}
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"rules":    rules,
		"tenant":   tenant,
		"exp":      time.Now().Add(tokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	if len(secretKey) == 0 {
		return nil, errors.New("JWT secret belum diinisialisasi")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signature tidak valid")
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token tidak valid")
}
