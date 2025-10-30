package repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Agushim/go_wifi_billing/models"
)

type CoverageRepository interface {
	Create(coverage *models.Coverage) error
	GetByID(id uuid.UUID) (*models.Coverage, error)
	GetAll() ([]models.Coverage, error)
	Update(coverage *models.Coverage) error
	Delete(id uuid.UUID) error
}

type coverageRepository struct {
	db *gorm.DB
}

func NewCoverageRepository(db *gorm.DB) CoverageRepository {
	return &coverageRepository{db: db}
}

func (r *coverageRepository) Create(coverage *models.Coverage) error {
	return r.db.Create(coverage).Error
}

func (r *coverageRepository) GetByID(id uuid.UUID) (*models.Coverage, error) {
	var c models.Coverage
	if err := r.db.First(&c, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *coverageRepository) GetAll() ([]models.Coverage, error) {
	var list []models.Coverage
	if err := r.db.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *coverageRepository) Update(coverage *models.Coverage) error {
	return r.db.Save(coverage).Error
}

func (r *coverageRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Coverage{}, "id = ?", id).Error
}
