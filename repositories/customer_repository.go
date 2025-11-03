package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerRepository interface {
	Create(customer *models.Customer) error
	FindAll() ([]models.Customer, error)
	FindByID(id uuid.UUID) (*models.Customer, error)
	Update(customer *models.Customer) error
	Delete(id uuid.UUID) error
}

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) CustomerRepository {
	return &customerRepository{db}
}

func (r *customerRepository) Create(customer *models.Customer) error {
	return r.db.Create(customer).Error
}

func (r *customerRepository) FindAll() ([]models.Customer, error) {
	var customers []models.Customer
	err := r.db.
		Preload("User").
		Preload("Admin").
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		Find(&customers).Error
	return customers, err
}

func (r *customerRepository) FindByID(id uuid.UUID) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.
		Preload("User").
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		First(&customer, "id = ?", id).Error
	return &customer, err
}

func (r *customerRepository) Update(customer *models.Customer) error {
	return r.db.
		Omit("User").
		Omit("Coverage").
		Omit("Odc").
		Omit("Odp").
		Save(customer).Error
}

func (r *customerRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Customer{}, "id = ?", id).Error
}
