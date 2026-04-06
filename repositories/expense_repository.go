package repositories

import (
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExpenseRepository interface {
	FindAll(adminID *uuid.UUID, search string, category string, startDate *time.Time, endDate *time.Time) ([]models.Expense, error)
	FindByID(id string) (models.Expense, error)
	Create(expense *models.Expense) error
	Update(expense *models.Expense) error
	Delete(id string) error
}

type expenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) ExpenseRepository {
	return &expenseRepository{db}
}

func (r *expenseRepository) FindAll(adminID *uuid.UUID, search string, category string, startDate *time.Time, endDate *time.Time) ([]models.Expense, error) {
	var expenses []models.Expense
	query := r.db.Preload("Admin")

	if adminID != nil {
		query = query.Where("admin_id = ?", *adminID)
	}
	if search != "" {
		query = query.Where("LOWER(title) LIKE LOWER(?)", "%"+search+"%")
	}
	if category != "" {
		query = query.Where("LOWER(category) = LOWER(?)", category)
	}
	if startDate != nil {
		query = query.Where("expense_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("expense_date < ?", *endDate)
	}

	err := query.Order("expense_date DESC").Find(&expenses).Error
	return expenses, err
}

func (r *expenseRepository) FindByID(id string) (models.Expense, error) {
	var expense models.Expense
	err := r.db.Preload("Admin").First(&expense, "id = ?", id).Error
	return expense, err
}

func (r *expenseRepository) Create(expense *models.Expense) error {
	return r.db.Create(expense).Error
}

func (r *expenseRepository) Update(expense *models.Expense) error {
	return r.db.Save(expense).Error
}

func (r *expenseRepository) Delete(id string) error {
	return r.db.Delete(&models.Expense{}, "id = ?", id).Error
}
