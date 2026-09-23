package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey []byte
var tokenDuration time.Duration = 24 * time.Hour
var issuer string

func InitJWT(secret string, duration time.Duration, jwtIssuer string) {
	secretKey = []byte(secret)
	if duration > 0 {
		tokenDuration = duration
	}
	issuer = jwtIssuer
}

func GenerateToken(userID int, username string, rules string, tenant string) (string, error) {
	if len(secretKey) == 0 {
		return "", errors.New("JWT secret belum diinisialisasi")
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"rules":    rules,
		"tenant":   tenant,
		"iss":      issuer,
		"iat":      now.Unix(),
		"nbf":      now.Unix(),
		"exp":      now.Add(tokenDuration).Unix(),
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
	}, jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	if claimIss, exists := claims["iss"]; exists && issuer != "" {
		if iss, isStr := claimIss.(string); !isStr || iss != issuer {
			return nil, errors.New("penerbit token tidak sesuai")
		}
	}
	return claims, nil
}
