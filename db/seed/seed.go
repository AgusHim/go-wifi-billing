package seed

import (
	"gorm.io/gorm"
)

// Seed memanggil semua fungsi seeder
func Seed(db *gorm.DB) {
	SeedUsers(db)
	SeedPackages(db)
	SeedCoverages(db)
	SeedOdcs(db)
	SeedOdps(db)
}
