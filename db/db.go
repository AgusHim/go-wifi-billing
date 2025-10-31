package db

import (
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Agushim/go_wifi_billing/models"
)

func InitDB(postgresDsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if postgresDsn != "" {
		dialector = postgres.Open(postgresDsn)
	} else {
		// fallback to sqlite
		dialector = sqlite.Open("test.db")
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}
	return db.AutoMigrate(
		&models.Coverage{},
		&models.Package{},
		&models.User{},
		&models.Odc{},
		&models.Odp{},
		&models.Customer{},
		&models.Subscription{},
		&models.Bill{},
		&models.Payment{},
	)
}
