package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OdpRepository interface {
	Create(odp *models.Odp) error
	FindAll() ([]models.Odp, error)
	FindByID(id uuid.UUID) (*models.Odp, error)
	Update(odp *models.Odp) error
	Delete(id uuid.UUID) error
}

type odpRepository struct {
	db *gorm.DB
}

func NewOdpRepository(db *gorm.DB) OdpRepository {
	return &odpRepository{db}
}

func (r *odpRepository) Create(odp *models.Odp) error {
	return r.db.Create(odp).Error
}

func (r *odpRepository) FindAll() ([]models.Odp, error) {
	var odps []models.Odp
	err := r.db.Preload("Odc").Preload("Coverage").Find(&odps).Error
	return odps, err
}

func (r *odpRepository) FindByID(id uuid.UUID) (*models.Odp, error) {
	var odp models.Odp
	err := r.db.First(&odp, "id = ?", id).Error
	return &odp, err
}

func (r *odpRepository) Update(odp *models.Odp) error {
	return r.db.Save(odp).Error
}

func (r *odpRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Odp{}, "id = ?", id).Error
}
