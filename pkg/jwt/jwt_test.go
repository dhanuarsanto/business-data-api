package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateTokenIssuer(t *testing.T) {
	InitJWT("secret-uji", 1*time.Hour, "issuer-a")

	good, err := GenerateToken(1, "budi", "sa", "maxtop")
	if err != nil {
		t.Fatalf("generate token gagal: %v", err)
	}
	if _, err := ValidateToken(good); err != nil {
		t.Fatalf("token dengan issuer cocok harus lolos: %v", err)
	}

	bad := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  1,
		"username": "budi",
		"rules":    "sa",
		"tenant":   "maxtop",
		"iss":      "issuer-penipu",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	badStr, err := bad.SignedString(secretKey)
	if err != nil {
		t.Fatalf("sign token gagal: %v", err)
	}
	if _, err := ValidateToken(badStr); err == nil {
		t.Fatal("token dengan issuer berbeda harus ditolak")
	}
}