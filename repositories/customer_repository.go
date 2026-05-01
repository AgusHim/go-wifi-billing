package repositories

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Agushim/go_wifi_billing/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerRepository interface {
	Create(customer *models.Customer) error
	FindAll(page, limit int, search string, adminID *uuid.UUID, coverageID *uuid.UUID) ([]models.Customer, int64, error)
	FindByID(id uuid.UUID) (*models.Customer, error)
	FindByUserID(userID uuid.UUID) (*models.Customer, error)
	Update(customer *models.Customer) error
	Delete(id uuid.UUID) error
	NextServiceNumber(coverageID uuid.UUID) (string, error)
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

func (r *customerRepository) FindAll(page, limit int, search string, adminID *uuid.UUID, coverageID *uuid.UUID) ([]models.Customer, int64, error) {
	var customers []models.Customer
	var total int64

	query := r.db.Model(&models.Customer{}).
		Preload("User", func(tx *gorm.DB) *gorm.DB { return tx.Unscoped() }).
		Preload("Admin").
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		Preload("Subscriptions", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("subscriptions.created_at DESC")
		}).
		Preload("Subscriptions.Package").
		Joins("LEFT JOIN users ON users.id = customers.user_id")

	// Pencarian di User.Name atau User.Email
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ?", searchPattern, searchPattern)
	}
	if adminID != nil {
		query = query.Where("customers.admin_id = ?", *adminID)
	}
	if coverageID != nil {
		query = query.Where("customers.coverage_id = ?", *coverageID)
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
		Preload("User", func(tx *gorm.DB) *gorm.DB { return tx.Unscoped() }).
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		First(&customer, "id = ?", id).Error
	return &customer, err
}

func (r *customerRepository) FindByUserID(userID uuid.UUID) (*models.Customer, error) {
	var customer models.Customer

	err := r.db.
		Preload("User", func(tx *gorm.DB) *gorm.DB { return tx.Unscoped() }).
		Preload("Admin").
		Preload("Coverage").
		Preload("Odc").
		Preload("Odp").
		First(&customer, "user_id = ?", userID).Error

	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func (r *customerRepository) Update(customer *models.Customer) error {
	return r.db.Omit("User", "Coverage", "Odc", "Odp", "Admin").Save(customer).Error
}

func (r *customerRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Customer{}, "id = ?", id).Error
}

// NextServiceNumber menghasilkan service number unik per coverage
// dengan format "{CODE_AREA}-{seq:06d}". Sequence dihitung dari max
// existing service_number yang punya prefix sama (termasuk yang soft-deleted).
func (r *customerRepository) NextServiceNumber(coverageID uuid.UUID) (string, error) {
	var coverage models.Coverage
	if err := r.db.Select("code_area").First(&coverage, "id = ?", coverageID).Error; err != nil {
		return "", err
	}
	code := strings.TrimSpace(coverage.CodeArea)
	if code == "" {
		return "", errors.New("coverage code_area kosong")
	}
	prefix := code + "-"

	var existing []string
	if err := r.db.
		Unscoped().
		Model(&models.Customer{}).
		Where("service_number LIKE ?", prefix+"%").
		Pluck("service_number", &existing).Error; err != nil {
		return "", err
	}

	maxSeq := 0
	for _, sn := range existing {
		suffix := strings.TrimPrefix(sn, prefix)
		n, err := strconv.Atoi(suffix)
		if err == nil && n > maxSeq {
			maxSeq = n
		}
	}

	return fmt.Sprintf("%s%06d", prefix, maxSeq+1), nil
}
