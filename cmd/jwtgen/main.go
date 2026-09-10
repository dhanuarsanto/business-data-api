package main

import (
	"crypto/rand"
	"encoding/hex"
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

	bytes := make([]byte, 32)
	rand.Read(bytes)
	fmt.Printf("Silakan gunakan string berikut sebagai JWT_SECRET di .env Anda:\n\n%s\n", hex.EncodeToString(bytes))
}
