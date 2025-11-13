package lib

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
)

func GetMidtransAuth() string {
	// Misalnya ini adalah value dari env.My_Server_Key
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	
	// Gabungkan dengan titik dua di akhir sesuai string aslinya
	raw := fmt.Sprintf("%s:", serverKey)

	// Encode ke base64
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))

	return encoded
}

func GenerateSignature(orderID string, statusCode string, grossAmount string) string {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	raw := orderID + statusCode + grossAmount + serverKey
	hash := sha512.Sum512([]byte(raw))
	return hex.EncodeToString(hash[:])
}
