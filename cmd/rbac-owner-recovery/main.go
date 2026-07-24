package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/Agushim/go_wifi_billing/db"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	if os.Getenv("RECOVERY_CONFIRM") != "RECOVER_OWNER" {
		log.Fatal("owner recovery: set RECOVERY_CONFIRM=RECOVER_OWNER after completing the recovery checklist")
	}
	dsn := strings.TrimSpace(os.Getenv("POSTGRES_URL"))
	if dsn == "" {
		log.Fatal("owner recovery: POSTGRES_URL is required; SQLite fallback is disabled")
	}
	database, err := db.InitDB(dsn)
	if err != nil {
		log.Fatalf("owner recovery: connect database: %v", err)
	}
	result, err := services.RecoverOwner(context.Background(), database, services.OwnerRecoveryRequest{
		TargetEmail:   os.Getenv("RECOVERY_OWNER_EMAIL"),
		OperatorEmail: os.Getenv("RECOVERY_OPERATOR_EMAIL"),
		Reason:        os.Getenv("RECOVERY_REASON"),
	})
	if err != nil {
		log.Fatalf("owner recovery: %v", err)
	}
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("owner recovery: encode result: %v", err)
	}
	log.Printf("owner recovery completed: %s", output)
}
