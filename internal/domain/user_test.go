package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUserPasswordNeverSerialized(t *testing.T) {
	u := User{UserID: 1, Username: "budi", Password: "rahasia", Rules: "op"}
	raw, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal gagal: %v", err)
	}
	if strings.Contains(string(raw), "rahasia") {
		t.Fatalf("password bocor ke JSON: %s", raw)
	}
}