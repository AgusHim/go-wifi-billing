package seed

import (
	"log"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedUsers(db *gorm.DB) {
	// Cek apakah sudah ada data user admin
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		log.Println("✅ Users table already seeded, skipping...")
		return
	}

	// Hash password
	password, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ Failed to hash password: %v", err)
	}

	users := []models.User{
		{
			Name:      "Administrator",
			Email:     "admin@example.com",
			Phone:     "081234567890",
			Password:  string(password),
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:      "User Biasa",
			Email:     "user@example.com",
			Phone:     "089876543210",
			Password:  string(password),
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	if err := db.Create(&users).Error; err != nil {
		log.Fatalf("❌ Failed to seed users: %v", err)
	}

	log.Println("🌱 User table seeded successfully!")
}
