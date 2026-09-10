package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	if os.Getenv("APP_ENV") == "production" {
		fmt.Println("AKSES DITOLAK: Script generator ini dilarang keras dieksekusi di server Production!")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Println("Gunakan perintah: go run cmd/keygen/main.go \"Nama Developer\"")
		return
	}
	developerName := os.Args[1]

	bytes := make([]byte, 32)
	rand.Read(bytes)
	newKey := "key_" + hex.EncodeToString(bytes)

	fileData, _ := os.ReadFile("api_keys.json")
	keys := make(map[string]string)
	json.Unmarshal(fileData, &keys)

	keys[newKey] = developerName

	newData, _ := json.MarshalIndent(keys, "", "  ")
	os.WriteFile("api_keys.json", newData, 0644)

	fmt.Printf("API Key berhasil dibuat untuk: %s\n", developerName)
	fmt.Printf("API Key: %s\n", newKey)
}
