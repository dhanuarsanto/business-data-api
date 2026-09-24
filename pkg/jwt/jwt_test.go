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

func TestValidateTokenRejectsMissingIssuer(t *testing.T) {
	InitJWT("secret-uji", 1*time.Hour, "issuer-a")

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  1,
		"username": "budi",
		"rules":    "sa",
		"tenant":   "maxtop",
		"exp":      time.Now().Add(time.Hour).Unix(),
	}).SignedString(secretKey)
	if err != nil {
		t.Fatalf("sign token gagal: %v", err)
	}
	if _, err := ValidateToken(tok); err == nil {
		t.Fatal("token tanpa issuer harus ditolak saat JWT_ISSUER diset")
	}
}

func TestValidateTokenRejectsWeakClaims(t *testing.T) {
	InitJWT("secret-uji", 1*time.Hour, "issuer-a")

	sign := func(claims jwt.MapClaims) string {
		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secretKey)
		if err != nil {
			t.Fatalf("sign token gagal: %v", err)
		}
		return tok
	}

	futureIat := sign(jwt.MapClaims{
		"iss": "issuer-a",
		"iat": time.Now().Add(2 * time.Hour).Unix(),
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(3 * time.Hour).Unix(),
	})
	if _, err := ValidateToken(futureIat); err == nil {
		t.Fatal("token dengan iat di masa depan harus ditolak")
	}

	noExp := sign(jwt.MapClaims{
		"iss": "issuer-a",
		"iat": time.Now().Unix(),
	})
	if _, err := ValidateToken(noExp); err == nil {
		t.Fatal("token tanpa exp harus ditolak")
	}
}
