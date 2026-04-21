package repositories

import (
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	FindAll(adminID *uuid.UUID, search string, status string, startDate *time.Time, endDate *time.Time) ([]models.Payment, error)
	FindByID(id string) (models.Payment, error)
	Create(payment *models.Payment) error
	Update(payment *models.Payment) error
	Delete(id string) error
	FindByUserID(userID string) ([]models.Payment, error)
	FindActiveByBillID(billID uuid.UUID) (*models.Payment, error)
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db}
}

func (r *paymentRepository) FindAll(adminID *uuid.UUID, search string, status string, startDate *time.Time, endDate *time.Time) ([]models.Payment, error) {
	var payments []models.Payment
	query := r.db.
		Preload("Bill").
		Preload("Bill.Customer.User").
		Preload("Bill.Subscription").
		Preload("Bill.Subscription.Package").
		Preload("Admin")

	if search != "" {
		query = query.
			Joins("JOIN bills ON payments.bill_id = bills.id").
			Joins("JOIN customers ON bills.customer_id = customers.id").
			Joins("JOIN users ON customers.user_id = users.id").
			Where("LOWER(users.name) LIKE LOWER(?)", "%"+search+"%")
	}

	if adminID != nil {
		query = query.Where("payments.admin_id = ?", *adminID)
	}
	if status != "" {
		query = query.Where("LOWER(payments.status) = LOWER(?)", status)
	}
	if startDate != nil {
		query = query.Where("payments.payment_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("payments.payment_date < ?", *endDate)
	}

	err := query.Order("payments.payment_date DESC").Find(&payments).Error
	return payments, err
}

func (r *paymentRepository) FindByID(id string) (models.Payment, error) {
	var payment models.Payment
	err := r.db.
		Preload("Bill").
		Preload("Bill.Customer.User").
		Preload("Bill.Subscription").
		Preload("Bill.Subscription.Package").
		Preload("Admin").
		First(&payment, "id = ?", id).Error
	return payment, err
}

func (r *paymentRepository) Create(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

func (r *paymentRepository) Update(payment *models.Payment) error {
	return r.db.Omit("Bill").Save(payment).Error
}

func (r *paymentRepository) Delete(id string) error {
	return r.db.Delete(&models.Payment{}, "id = ?", id).Error
}

func (r *paymentRepository) FindActiveByBillID(billID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.
		Where("bill_id = ? AND LOWER(status) IN ?", billID, []string{"confirmed", "pending"}).
		First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *paymentRepository) FindByUserID(userID string) ([]models.Payment, error) {
	var payments []models.Payment
	err := r.db.
		Joins("JOIN bills ON payments.bill_id = bills.id").
		Joins("JOIN customers ON bills.customer_id = customers.id").
		Joins("JOIN users ON customers.user_id = users.id").
		Where("users.id = ?", userID).
		Preload("Bill").
		Preload("Bill.Customer.User").
		Preload("Bill.Subscription").
		Preload("Bill.Subscription.Package").
		Preload("Admin").
		Find(&payments).Error
	return payments, err
}
