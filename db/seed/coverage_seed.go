package seed

import (
	"log"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedCoverages menambahkan data awal ke tabel coverages
func SeedCoverages(db *gorm.DB) {
	var count int64
	db.Model(&models.Coverage{}).Count(&count)
	if count > 0 {
		log.Println("✅ Coverages table already seeded, skipping...")
		return
	}

	coverages := []models.Coverage{
		{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			CodeArea:    "CGK-001",
			Name:        "Jakarta Pusat",
			Address:     "Jl. MH Thamrin No. 10, Jakarta Pusat",
			Description: "Area jangkauan utama untuk wilayah Jakarta Pusat.",
			RangeArea:   15,
			Latitude:    -6.1834,
			Longitude:   106.8325,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			CodeArea:    "BDG-002",
			Name:        "Bandung Kota",
			Address:     "Jl. Asia Afrika No. 20, Bandung",
			Description: "Area jangkauan untuk pelanggan di wilayah Bandung kota dan sekitarnya.",
			RangeArea:   20,
			Latitude:    -6.9175,
			Longitude:   107.6191,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			CodeArea:    "SBY-003",
			Name:        "Surabaya Barat",
			Address:     "Jl. Mayjend Sungkono No. 50, Surabaya",
			Description: "Cakupan area layanan untuk wilayah Surabaya Barat.",
			RangeArea:   25,
			Latitude:    -7.2768,
			Longitude:   112.7193,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	if err := db.Create(&coverages).Error; err != nil {
		log.Fatalf("❌ Failed to seed coverages: %v", err)
	}

	log.Println("🌱 Coverages table seeded successfully!")
}
