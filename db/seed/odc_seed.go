package seed

import (
	"log"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedOdcs menambahkan data awal ke tabel odcs
func SeedOdcs(db *gorm.DB) {
	var count int64
	db.Model(&models.Odc{}).Count(&count)
	if count > 0 {
		log.Println("✅ ODC table already seeded, skipping...")
		return
	}

	odcs := []models.Odc{
		{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			CoverageID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"), // Ganti dengan ID coverage yang valid
			OdcKey:      "ODC-001",
			Code:        "ODC-001",
			PortOlt:     16,
			FoColor:     "Merah",
			PoleNumber:  "PN-01",
			CountPort:   64,
			Description: "ODC utama area A",
			ImageURL:    "https://example.com/odc1.jpg",
			Latitude:    -6.200000,
			Longitude:   106.816666,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			CoverageID:  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			OdcKey:      "ODC-002",
			Code:        "ODC-002",
			PortOlt:     8,
			FoColor:     "Kuning",
			PoleNumber:  "PN-02",
			CountPort:   32,
			Description: "ODC cabang area A",
			ImageURL:    "https://example.com/odc2.jpg",
			Latitude:    -6.201234,
			Longitude:   106.817890,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			CoverageID:  uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			OdcKey:      "ODC-003",
			Code:        "ODC-003",
			PortOlt:     12,
			FoColor:     "Biru",
			PoleNumber:  "PN-03",
			CountPort:   48,
			Description: "ODC area B",
			ImageURL:    "https://example.com/odc3.jpg",
			Latitude:    -6.205678,
			Longitude:   106.820123,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	if err := db.Create(&odcs).Error; err != nil {
		log.Fatalf("❌ Failed to seed ODCs: %v", err)
	}

	log.Println("🌱 ODC table seeded successfully!")
}
