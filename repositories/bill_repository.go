package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"gorm.io/gorm"
)

type BillRepository interface {
	FindAll() ([]models.Bill, error)
	FindByID(id string) (models.Bill, error)
	FindByPublicID(publicID string) (*models.Bill, error)
	FindByUserID(userID string) ([]models.Bill, error)
	Create(bill *models.Bill) error
	Update(bill *models.Bill) error
	Delete(id string) error
	FindBillByCustomerAndMonth(customerID string, month int, year int) (*models.Bill, error)
}

type billRepository struct {
	db *gorm.DB
}

func NewBillRepository(db *gorm.DB) BillRepository {
	return &billRepository{db}
}

func (r *billRepository) FindAll() ([]models.Bill, error) {
	var bills []models.Bill
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		Find(&bills).Error
	return bills, err
}

func (r *billRepository) FindByID(id string) (models.Bill, error) {
	var bill models.Bill
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		First(&bill, "id = ?", id).Error
	return bill, err
}
func (r *billRepository) FindByUserID(userID string) ([]models.Bill, error) {
	var bills []models.Bill
	err := r.db.
		Joins("JOIN customers ON customers.id = bills.customer_id").
		Joins("JOIN users ON users.id = customers.user_id").
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		Where("users.id = ?", userID).
		Find(&bills).Error

	return bills, err
}

func (r *billRepository) FindByPublicID(publicID string) (*models.Bill, error) {
	var bill models.Bill
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Subscription").
		Preload("Subscription.Package").
		First(&bill, "public_id = ?", publicID).Error

	if err != nil {
		return nil, err
	}
	return &bill, nil
}


func (r *billRepository) Create(bill *models.Bill) error {
	return r.db.Create(bill).Error
}

func (r *billRepository) Update(bill *models.Bill) error {
	return r.db.Omit("Subscription").Save(bill).Error
}

func (r *billRepository) Delete(id string) error {
	return r.db.Delete(&models.Bill{}, "id = ?", id).Error
}

func (r *billRepository) FindBillByCustomerAndMonth(customerID string, month int, year int) (*models.Bill, error) {
	var bill models.Bill
	err := r.db.
		Where("customer_id = ? AND EXTRACT(MONTH FROM bill_date) = ? AND EXTRACT(YEAR FROM bill_date) = ?",
			customerID, month, year).
		First(&bill).Error

	if err != nil {
		return nil, err
	}
	return &bill, nil
}
