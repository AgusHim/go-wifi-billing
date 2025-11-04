package seed

import (
	"gorm.io/gorm"
)

// Seed memanggil semua fungsi seeder
func Seed(db *gorm.DB) {
	SeedCoverages(db)
	SeedOdcs(db)
	//SeedOdps(db)
	SeedPackages(db)
	SeedUsers(db)
}
