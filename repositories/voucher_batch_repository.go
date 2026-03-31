package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VoucherBatchRepository interface {
	Create(batch *models.VoucherBatch) error
	FindAll() ([]models.VoucherBatch, error)
	FindByID(id uuid.UUID) (*models.VoucherBatch, error)
	Update(batch *models.VoucherBatch) error
}

type voucherBatchRepository struct {
	db *gorm.DB
}

func NewVoucherBatchRepository(db *gorm.DB) VoucherBatchRepository {
	return &voucherBatchRepository{db: db}
}

func (r *voucherBatchRepository) Create(batch *models.VoucherBatch) error {
	return r.db.Create(batch).Error
}

func (r *voucherBatchRepository) FindAll() ([]models.VoucherBatch, error) {
	var batches []models.VoucherBatch
	err := r.db.
		Preload("Router").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Preload("Vouchers").
		Order("created_at desc").
		Find(&batches).Error
	return batches, err
}

func (r *voucherBatchRepository) FindByID(id uuid.UUID) (*models.VoucherBatch, error) {
	var batch models.VoucherBatch
	err := r.db.
		Preload("Router").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Preload("Vouchers").
		First(&batch, "id = ?", id).Error
	return &batch, err
}

func (r *voucherBatchRepository) Update(batch *models.VoucherBatch) error {
	return r.db.Save(batch).Error
}
