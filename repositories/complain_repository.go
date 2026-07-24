package repositories

import (
	"errors"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ComplainRepository interface {
	Create(complain *models.Complain) error
	GetByID(id uuid.UUID) (*models.Complain, error)
	GetAll() ([]models.Complain, error)
	GetByIDForUser(id, userID uuid.UUID) (*models.Complain, error)
	GetAllByUserID(userID uuid.UUID) ([]models.Complain, error)
	CustomerBelongsToUser(customerID, userID uuid.UUID) (bool, error)
	SubscriptionBelongsToCustomer(subscriptionID, customerID uuid.UUID) (bool, error)
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

func (r *complainRepository) GetByIDForUser(id, userID uuid.UUID) (*models.Complain, error) {
	var complain models.Complain
	if err := r.withRelations().
		Joins("JOIN customers AS owner_customers ON owner_customers.id = complains.customer_id").
		Where("complains.id = ? AND owner_customers.user_id = ?", id, userID).
		First(&complain).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &complain, nil
}

func (r *complainRepository) GetAllByUserID(userID uuid.UUID) ([]models.Complain, error) {
	var complains []models.Complain
	if err := r.withRelations().
		Joins("JOIN customers AS owner_customers ON owner_customers.id = complains.customer_id").
		Where("owner_customers.user_id = ?", userID).
		Find(&complains).Error; err != nil {
		return nil, err
	}
	return complains, nil
}

func (r *complainRepository) CustomerBelongsToUser(customerID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Customer{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", customerID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 1, nil
}

func (r *complainRepository) SubscriptionBelongsToCustomer(subscriptionID, customerID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Subscription{}).
		Where("id = ? AND customer_id = ? AND deleted_at IS NULL", subscriptionID, customerID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 1, nil
}

func (r *complainRepository) Update(complain *models.Complain) error {
	return r.db.Omit(clause.Associations).Save(complain).Error
}

func (r *complainRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Complain{}, "id = ?", id).Error
}
