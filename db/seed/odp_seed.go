package seed

import (
	"log"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedOdps menambahkan data awal ke tabel odps
func SeedOdps(db *gorm.DB) {
	var count int64
	db.Model(&models.Odp{}).Count(&count)
	if count > 0 {
		log.Println("✅ ODP table already seeded, skipping...")
		return
	}

	odps := []models.Odp{
		{
			OdcID:         uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			CoverageID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			OdcPortNumber: 1,
			Code:          "ODP-001",
			FoTubeColor:   "Merah",
			PoleNumber:    "PN-ODP-01",
			CountPort:     8,
			Description:   "ODP utama area A - terhubung ke ODC-001",
			ImageURL:      "https://example.com/odp1.jpg",
			Latitude:      -6.2005,
			Longitude:     106.817,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			OdcID:         uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			CoverageID:    uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			OdcPortNumber: 2,
			Code:          "ODP-002",
			FoTubeColor:   "Kuning",
			PoleNumber:    "PN-ODP-02",
			CountPort:     16,
			Description:   "ODP cabang area A - jalur distribusi 2",
			ImageURL:      "https://example.com/odp2.jpg",
			Latitude:      -6.201,
			Longitude:     106.818,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			OdcID:         uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			CoverageID:    uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			OdcPortNumber: 3,
			Code:          "ODP-003",
			FoTubeColor:   "Biru",
			PoleNumber:    "PN-ODP-03",
			CountPort:     8,
			Description:   "ODP area B - terhubung ke ODC-003",
			ImageURL:      "https://example.com/odp3.jpg",
			Latitude:      -6.202,
			Longitude:     106.8195,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	if err := db.Omit("Odc", "Coverage").Create(&odps).Error; err != nil {
		log.Fatalf("❌ Failed to seed ODPs: %v", err)
	}

	log.Println("🌱 ODP table seeded successfully!")
}
