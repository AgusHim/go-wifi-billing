package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VoucherRepository interface {
	Create(voucher *models.Voucher) error
	FindAll(batchID string) ([]models.Voucher, error)
	FindByID(id uuid.UUID) (*models.Voucher, error)
	FindByCode(code string) (*models.Voucher, error)
	Update(voucher *models.Voucher) error
}

type voucherRepository struct {
	db *gorm.DB
}

func NewVoucherRepository(db *gorm.DB) VoucherRepository {
	return &voucherRepository{db: db}
}

func (r *voucherRepository) Create(voucher *models.Voucher) error {
	return r.db.Create(voucher).Error
}

func (r *voucherRepository) FindAll(batchID string) ([]models.Voucher, error) {
	var vouchers []models.Voucher
	query := r.db.
		Preload("Batch").
		Preload("Router").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router")
	if batchID != "" {
		query = query.Where("batch_id = ?", batchID)
	}
	err := query.Order("created_at desc").Find(&vouchers).Error
	return vouchers, err
}

func (r *voucherRepository) FindByID(id uuid.UUID) (*models.Voucher, error) {
	var voucher models.Voucher
	err := r.db.
		Preload("Batch").
		Preload("Router").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		First(&voucher, "id = ?", id).Error
	return &voucher, err
}

func (r *voucherRepository) FindByCode(code string) (*models.Voucher, error) {
	var voucher models.Voucher
	err := r.db.
		Preload("Batch").
		Preload("Router").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		First(&voucher, "LOWER(code) = LOWER(?)", code).Error
	return &voucher, err
}

func (r *voucherRepository) Update(voucher *models.Voucher) error {
	return r.db.Save(voucher).Error
}
