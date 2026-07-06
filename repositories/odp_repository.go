package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OdpRepository interface {
	Create(odp *models.Odp) error
	FindAll(coverageIDs []uuid.UUID) ([]models.Odp, error)
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

func (r *odpRepository) FindAll(coverageIDs []uuid.UUID) ([]models.Odp, error) {
	var odps []models.Odp
	query := r.db.Preload("Odc").Preload("Coverage")
	if len(coverageIDs) > 0 {
		query = query.Where("coverage_id IN ?", coverageIDs)
	}
	err := query.Find(&odps).Error
	return odps, err
}

func (r *odpRepository) FindByID(id uuid.UUID) (*models.Odp, error) {
	var odp models.Odp
	err := r.db.First(&odp, "id = ?", id).Error
	return &odp, err
}

func (r *odpRepository) Update(odp *models.Odp) error {
	return r.db.Omit(clause.Associations).Save(odp).Error
}

func (r *odpRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Odp{}, "id = ?", id).Error
}
