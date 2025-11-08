package repositories

import (
	"errors"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ComplainRepository interface {
	Create(complain *models.Complain) error
	GetByID(id uuid.UUID) (*models.Complain, error)
	GetAll() ([]models.Complain, error)
	Update(complain *models.Complain) error
	Delete(id uuid.UUID) error
}

type complainRepository struct {
	db *gorm.DB
}

func NewComplainRepository(db *gorm.DB) ComplainRepository {
	return &complainRepository{db: db}
}

func (r *complainRepository) withRelations() *gorm.DB {
	return r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Technician")
}

func (r *complainRepository) Create(complain *models.Complain) error {
	return r.db.Create(complain).Error
}

func (r *complainRepository) GetByID(id uuid.UUID) (*models.Complain, error) {
	var complain models.Complain
	if err := r.withRelations().First(&complain, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &complain, nil
}

func (r *complainRepository) GetAll() ([]models.Complain, error) {
	var complains []models.Complain
	if err := r.withRelations().Find(&complains).Error; err != nil {
		return nil, err
	}
	return complains, nil
}

func (r *complainRepository) Update(complain *models.Complain) error {
	return r.db.Save(complain).Error
}

func (r *complainRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Complain{}, "id = ?", id).Error
}

