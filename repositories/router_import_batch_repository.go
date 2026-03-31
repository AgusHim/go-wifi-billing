package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterImportBatchRepository interface {
	Create(batch *models.RouterImportBatch) error
	FindByRouterID(routerID uuid.UUID) ([]models.RouterImportBatch, error)
	FindByID(id uuid.UUID) (*models.RouterImportBatch, error)
}

type routerImportBatchRepository struct {
	db *gorm.DB
}

func NewRouterImportBatchRepository(db *gorm.DB) RouterImportBatchRepository {
	return &routerImportBatchRepository{db: db}
}

func (r *routerImportBatchRepository) Create(batch *models.RouterImportBatch) error {
	return r.db.Create(batch).Error
}

func (r *routerImportBatchRepository) FindByRouterID(routerID uuid.UUID) ([]models.RouterImportBatch, error) {
	var batches []models.RouterImportBatch
	err := r.db.
		Preload("Router").
		Order("created_at desc").
		Where("router_id = ?", routerID).
		Find(&batches).Error
	return batches, err
}

func (r *routerImportBatchRepository) FindByID(id uuid.UUID) (*models.RouterImportBatch, error) {
	var batch models.RouterImportBatch
	err := r.db.
		Preload("Router").
		Preload("Items").
		Preload("Items.ExistingNetworkPlan").
		Preload("Items.ExistingServiceAccount").
		Preload("Items.MatchedNetworkPlan").
		First(&batch, "id = ?", id).Error
	return &batch, err
}
