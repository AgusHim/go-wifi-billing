package seed

import (
	"log"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"gorm.io/gorm"
)

// SeedPackages menambahkan data awal ke tabel packages
func SeedPackages(db *gorm.DB) {
	var count int64
	db.Model(&models.Package{}).Count(&count)
	if count > 0 {
		log.Println("✅ Packages table already seeded, skipping...")
		return
	}

	packages := []models.Package{
		{
			Category:    "Basic",
			Name:        "Basic 10 Mbps",
			SpeedMbps:   10,
			QuotaGB:     "100",
			Price:       99000,
			Description: "Paket internet dasar dengan kecepatan hingga 10 Mbps dan kuota 100GB per bulan.",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Category:    "Standard",
			Name:        "Standard 30 Mbps",
			SpeedMbps:   30,
			QuotaGB:     "300",
			Price:       199000,
			Description: "Paket internet menengah dengan kecepatan hingga 30 Mbps dan kuota 300GB per bulan.",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Category:    "Premium",
			Name:        "Premium 100 Mbps",
			SpeedMbps:   100,
			QuotaGB:     "1000",
			Price:       399000,
			Description: "Paket internet premium untuk streaming dan gaming, kecepatan hingga 100 Mbps, kuota 1TB.",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	if err := db.Create(&packages).Error; err != nil {
		log.Fatalf("❌ Failed to seed packages: %v", err)
	}

	log.Println("🌱 Packages table seeded successfully!")
}
