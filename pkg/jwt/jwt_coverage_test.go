package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func withActive(t *testing.T, m *manager) {
	t.Helper()
	sebelum := active.Load()
	t.Cleanup(func() { active.Store(sebelum) })
	active.Store(m)
}

func TestNewManagerDurasiTidakPositifMemakaiBawaan(t *testing.T) {
	for _, durasi := range []time.Duration{0, -1 * time.Hour, -1} {
		if got := newManager("rahasia", durasi, "issuer").tokenDuration; got != 24*time.Hour {
			t.Fatalf("durasi %v harus jadi bawaan 24 jam, dapat %v", durasi, got)
		}
	}
	if got := newManager("rahasia", 5*time.Minute, "issuer").tokenDuration; got != 5*time.Minute {
		t.Fatalf("durasi positif harus dipertahankan, dapat %v", got)
	}
}

func TestManagerTanpaRahasiaMenolakSemuaOperasi(t *testing.T) {
	tanpaKunci := &manager{tokenDuration: time.Hour}

	if _, err := tanpaKunci.generateToken(1, "budi", "sa", "maxtop"); !errors.Is(err, errNotInitialized) {
		t.Fatalf("GenerateToken tanpa rahasia harus ditolak, dapat %v", err)
	}
	if _, err := tanpaKunci.validateToken("apa-saja"); !errors.Is(err, errNotInitialized) {
		t.Fatalf("ValidateToken tanpa rahasia harus ditolak, dapat %v", err)
	}
}

func TestFungsiPaketMenolakSaatBelumDiinisialisasi(t *testing.T) {
	withActive(t, nil)
	if _, err := GenerateToken(1, "budi", "sa", "maxtop"); !errors.Is(err, errNotInitialized) {
		t.Fatalf("GenerateToken harus menolak, dapat %v", err)
	}
	if _, err := ValidateToken("apa-saja"); !errors.Is(err, errNotInitialized) {
		t.Fatalf("ValidateToken harus menolak, dapat %v", err)
	}

	withActive(t, &manager{tokenDuration: time.Hour})
	if _, err := GenerateToken(1, "budi", "sa", "maxtop"); !errors.Is(err, errNotInitialized) {
		t.Fatalf("rahasia tanpaKunci harus ditolak, dapat %v", err)
	}
	if _, err := ValidateToken("apa-saja"); !errors.Is(err, errNotInitialized) {
		t.Fatalf("rahasia tanpaKunci harus ditolak, dapat %v", err)
	}
}

func TestValidateTokenMenolakMetodeSignatureLain(t *testing.T) {
	InitJWT("secret-uji-signature", time.Hour, "issuer-a")
	rahasia := testSecretKey(t)

	asli, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "issuer-a",
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(rahasia)
	if err != nil {
		t.Fatalf("sign HS256 gagal: %v", err)
	}
	if _, err := ValidateToken(asli); err != nil {
		t.Fatalf("HS256 harus lolos: %v", err)
	}

	bagian := strings.Split(asli, ".")
	if len(bagian) != 3 {
		t.Fatalf("token harus punya 3 bagian, dapat %d", len(bagian))
	}
	tandaTangan := []byte(bagian[2])
	if tandaTangan[0] == 'A' {
		tandaTangan[0] = 'B'
	} else {
		tandaTangan[0] = 'A'
	}
	rusak := bagian[0] + "." + bagian[1] + "." + string(tandaTangan)
	if _, err := ValidateToken(rusak); err == nil {
		t.Fatal("token dengan tanda tangan dirusak harus ditolak")
	}

	kunciECDSA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("buat kunci ECDSA gagal: %v", err)
	}
	asimetris, err := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": "issuer-a",
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(kunciECDSA)
	if err != nil {
		t.Fatalf("sign ES256 gagal: %v", err)
	}
	if _, err := ValidateToken(asimetris); err == nil {
		t.Fatal("token dengan metode bukan HMAC harus ditolak")
	}
}

func TestValidateTokenTanpaIssuerMenghapusPemeriksaanPenerbit(t *testing.T) {
	InitJWT("secret-uji-tanpa-iss", time.Hour, "")

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1,
		"iat":     time.Now().Unix(),
		"nbf":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}).SignedString(testSecretKey(t))
	if err != nil {
		t.Fatalf("sign gagal: %v", err)
	}
	claims, err := ValidateToken(tok)
	if err != nil {
		t.Fatalf("issuer tanpaKunci tak boleh diperiksa: %v", err)
	}
	if claims["user_id"] == nil {
		t.Fatalf("claims harus dikembalikan: %v", claims)
	}
}

func TestValidateTokenMenolakIssuerBukanTeks(t *testing.T) {
	InitJWT("secret-uji-iss-angka", time.Hour, "issuer-a")

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": 12345,
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(testSecretKey(t))
	if err != nil {
		t.Fatalf("sign gagal: %v", err)
	}
	if _, err := ValidateToken(tok); err == nil {
		t.Fatal("issuer numerik harus ditolak")
	}
}

func TestValidateTokenMenolakSecretLainDanKadaluarsa(t *testing.T) {
	InitJWT("secret-asli-untuk-validasi-token", time.Hour, "issuer-a")

	asal, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "issuer-a",
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("secret-pemalsu-yang-panjang-sekali"))
	if err != nil {
		t.Fatalf("sign gagal: %v", err)
	}
	if _, err := ValidateToken(asal); err == nil {
		t.Fatal("token yang ditandatangani secret lain harus ditolak")
	}

	kadaluarsa, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "issuer-a",
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"nbf": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-time.Hour).Unix(),
	}).SignedString(testSecretKey(t))
	if err != nil {
		t.Fatalf("sign gagal: %v", err)
	}
	if _, err := ValidateToken(kadaluarsa); err == nil {
		t.Fatal("token kedaluwarsa harus ditolak")
	}
}

func TestGenerateTokenMemuatSeluruhKlaimDenganBawaan(t *testing.T) {
	InitJWT("secret-uji-klaim", 2*time.Hour, "issuer-klaim")

	raw, err := GenerateToken(42, "siti", "op", "pandora")
	if err != nil {
		t.Fatalf("generate gagal: %v", err)
	}
	claims, err := ValidateToken(raw)
	if err != nil {
		t.Fatalf("validate gagal: %v", err)
	}
	if claims["user_id"] != float64(42) || claims["username"] != "siti" ||
		claims["rules"] != "op" || claims["tenant"] != "pandora" || claims["iss"] != "issuer-klaim" {
		t.Fatalf("klaim tak lengkap: %v", claims)
	}
	for _, wajib := range []string{"iat", "nbf", "exp"} {
		if claims[wajib] == nil {
			t.Fatalf("klaim %q wajib ada: %v", wajib, claims)
		}
	}
}

func TestInitJWTMenggantiManagerSecaraAtomik(t *testing.T) {
	sebelum := active.Load()
	t.Cleanup(func() { active.Store(sebelum) })

	InitJWT("secret-pertama-abcdefghijklmnopqrstuvwxyz", time.Hour, "issuer-satu")
	pertama, err := GenerateToken(1, "a", "sa", "maxtop")
	if err != nil {
		t.Fatalf("generate gagal: %v", err)
	}

	InitJWT("secret-kedua-abcdefghijklmnopqrstuvwxyz", time.Hour, "issuer-dua")
	kedua, err := GenerateToken(1, "a", "sa", "maxtop")
	if err != nil {
		t.Fatalf("generate gagal: %v", err)
	}
	if pertama == kedua {
		t.Fatal("pergantian secret harus menghasilkan token berbeda")
	}
	if _, err := ValidateToken(pertama); err == nil {
		t.Fatal("token dari secret lama harus ditolak")
	}
	if _, err := ValidateToken(kedua); err != nil {
		t.Fatalf("token dari secret baru harus lolos: %v", err)
	}
}
