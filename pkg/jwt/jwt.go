package jwt

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const MinSecretLength = 32

const tokenLeeway = 30 * time.Second

var errNotInitialized = errors.New("JWT secret belum diinisialisasi")

type manager struct {
	secretKey     []byte
	tokenDuration time.Duration
	issuer        string
}

var active atomic.Pointer[manager]

func newManager(secret string, duration time.Duration, jwtIssuer string) *manager {
	if duration <= 0 {
		duration = 24 * time.Hour
	}
	return &manager{
		secretKey:     []byte(secret),
		tokenDuration: duration,
		issuer:        jwtIssuer,
	}
}

func InitJWT(secret string, duration time.Duration, jwtIssuer string) {
	active.Store(newManager(secret, duration, jwtIssuer))
}

func current() (*manager, error) {
	m := active.Load()
	if m == nil || len(m.secretKey) == 0 {
		return nil, errNotInitialized
	}
	return m, nil
}

func (m *manager) generateToken(userID int, username string, rules string, tenant string) (string, error) {
	if len(m.secretKey) == 0 {
		return "", errNotInitialized
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"rules":    rules,
		"tenant":   tenant,
		"iss":      m.issuer,
		"iat":      now.Unix(),
		"nbf":      now.Unix(),
		"exp":      now.Add(m.tokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *manager) validateToken(tokenString string) (jwt.MapClaims, error) {
	if len(m.secretKey) == 0 {
		return nil, errNotInitialized
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signature tidak valid")
		}
		return m.secretKey, nil
	}, jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(tokenLeeway))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	if m.issuer != "" {
		claimIss, exists := claims["iss"]
		if !exists {
			return nil, errors.New("penerbit token tidak sesuai")
		}
		if iss, isStr := claimIss.(string); !isStr || iss != m.issuer {
			return nil, errors.New("penerbit token tidak sesuai")
		}
	}
	return claims, nil
}

func GenerateToken(userID int, username string, rules string, tenant string) (string, error) {
	m, err := current()
	if err != nil {
		return "", err
	}
	return m.generateToken(userID, username, rules, tenant)
}

func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	m, err := current()
	if err != nil {
		return nil, err
	}
	return m.validateToken(tokenString)
}
