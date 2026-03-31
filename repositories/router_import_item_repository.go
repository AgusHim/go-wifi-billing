package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"gorm.io/gorm"
)

type RouterImportItemRepository interface {
	CreateMany(items []models.RouterImportItem) error
}

type routerImportItemRepository struct {
	db *gorm.DB
}

func NewRouterImportItemRepository(db *gorm.DB) RouterImportItemRepository {
	return &routerImportItemRepository{db: db}
}

func (r *routerImportItemRepository) CreateMany(items []models.RouterImportItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}
