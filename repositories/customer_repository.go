package repositories

import (
	"strings"

	"github.com/Agushim/go_wifi_billing/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerRepository interface {
	Create(customer *models.Customer) error
	FindAll(page, limit int, search string) ([]models.Customer, int64, error)
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

func (r *customerRepository) FindAll(page, limit int, search string) ([]models.Customer, int64, error) {
	var customers []models.Customer
	var total int64

	query := r.db.Model(&models.Customer{}).
		Preload("User").
		Preload("Admin").
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		Joins("LEFT JOIN users ON users.id = customers.user_id")

	// Pencarian di User.Name atau User.Email
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?", searchPattern, searchPattern)
	}

	// Hitung total data
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Pagination
	offset := (page - 1) * limit
	if err := query.
		Order("customers.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&customers).Error; err != nil {
		return nil, 0, err
	}

	return customers, total, nil
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
