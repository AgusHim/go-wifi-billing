package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OdcRepository interface {
	Create(odc *models.Odc) error
	FindAll() ([]models.Odc, error)
	FindByID(id uuid.UUID) (*models.Odc, error)
	Update(odc *models.Odc) error
	Delete(id uuid.UUID) error
}

type odcRepository struct {
	db *gorm.DB
}

func NewOdcRepository(db *gorm.DB) OdcRepository {
	return &odcRepository{db}
}

func (r *odcRepository) Create(odc *models.Odc) error {
	return r.db.Create(odc).Error
}

func (r *odcRepository) FindAll() ([]models.Odc, error) {
	var odcs []models.Odc
	err := r.db.Preload("Coverage").Find(&odcs).Error
	return odcs, err
}

func (r *odcRepository) FindByID(id uuid.UUID) (*models.Odc, error) {
	var odc models.Odc
	err := r.db.First(&odc, "id = ?", id).Error
	return &odc, err
}

func (r *odcRepository) Update(odc *models.Odc) error {
	return r.db.Save(odc).Error
}

func (r *odcRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Odc{}, "id = ?", id).Error
}
