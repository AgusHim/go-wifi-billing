package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BillingAutomationRunRepository interface {
	Create(run *models.BillingAutomationRun) error
	Update(run *models.BillingAutomationRun) error
}

type billingAutomationRunRepository struct {
	db *gorm.DB
}

func NewBillingAutomationRunRepository(db *gorm.DB) BillingAutomationRunRepository {
	return &billingAutomationRunRepository{db: db}
}

func (r *billingAutomationRunRepository) Create(run *models.BillingAutomationRun) error {
	return r.db.Create(run).Error
}

func (r *billingAutomationRunRepository) Update(run *models.BillingAutomationRun) error {
	return r.db.Omit(clause.Associations).Save(run).Error
}
