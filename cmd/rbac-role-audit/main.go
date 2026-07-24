package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Agushim/go_wifi_billing/db"
	"github.com/Agushim/go_wifi_billing/db/seed"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	database, err := db.InitDB(os.Getenv("POSTGRES_URL"))
	if err != nil {
		log.Fatalf("role audit: connect database: %v", err)
	}

	report, err := seed.AuditLegacyUserRoles(database, os.Getenv("INITIAL_OWNER_EMAIL"))
	if err != nil {
		log.Fatalf("role audit: %v", err)
	}

	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("role audit: encode report: %v", err)
	}
	fmt.Println(string(output))

	if len(report.UnknownRoleUsers) > 0 {
		log.Fatalf("role audit blocked: %d user(s) have unknown roles", len(report.UnknownRoleUsers))
	}
	if len(report.OwnerCandidates) == 0 {
		log.Fatal("role audit blocked: no owner candidate; set INITIAL_OWNER_EMAIL to exactly one existing active user")
	}
}
