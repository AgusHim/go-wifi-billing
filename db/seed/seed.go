package seed

import (
	"gorm.io/gorm"
)

// Seed memanggil semua fungsi seeder
func Seed(db *gorm.DB) {
	SeedPackages(db)
	SeedCoverages(db)
	SeedOdcs(db)
	SeedOdps(db)
	SeedUsers(db)
}
