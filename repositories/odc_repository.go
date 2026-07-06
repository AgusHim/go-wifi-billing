package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OdcRepository interface {
	Create(odc *models.Odc) error
	FindAll(coverageIDs []uuid.UUID) ([]models.Odc, error)
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

func (r *odcRepository) FindAll(coverageIDs []uuid.UUID) ([]models.Odc, error) {
	var odcs []models.Odc
	query := r.db.Preload("Coverage")
	if len(coverageIDs) > 0 {
		query = query.Where("coverage_id IN ?", coverageIDs)
	}
	err := query.Find(&odcs).Error
	return odcs, err
}

func (r *odcRepository) FindByID(id uuid.UUID) (*models.Odc, error) {
	var odc models.Odc
	err := r.db.First(&odc, "id = ?", id).Error
	return &odc, err
}

func (r *odcRepository) Update(odc *models.Odc) error {
	return r.db.Omit(clause.Associations).Save(odc).Error
}

func (r *odcRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Odc{}, "id = ?", id).Error
}
