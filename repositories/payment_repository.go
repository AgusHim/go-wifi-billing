package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	FindAll() ([]models.Payment, error)
	FindByID(id string) (models.Payment, error)
	Create(payment *models.Payment) error
	Update(payment *models.Payment) error
	Delete(id string) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db}
}

func (r *paymentRepository) FindAll() ([]models.Payment, error) {
	var payments []models.Payment
	err := r.db.Preload("Bill").Find(&payments).Error
	return payments, err
}

func (r *paymentRepository) FindByID(id string) (models.Payment, error) {
	var payment models.Payment
	err := r.db.Preload("Bill").First(&payment, "id = ?", id).Error
	return payment, err
}

func (r *paymentRepository) Create(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

func (r *paymentRepository) Update(payment *models.Payment) error {
	return r.db.Save(payment).Error
}

func (r *paymentRepository) Delete(id string) error {
	return r.db.Delete(&models.Payment{}, "id = ?", id).Error
}
