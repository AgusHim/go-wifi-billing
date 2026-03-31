package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NetworkPlanRepository interface {
	Create(plan *models.NetworkPlan) error
	FindAll() ([]models.NetworkPlan, error)
	FindByID(id uuid.UUID) (*models.NetworkPlan, error)
	Update(plan *models.NetworkPlan) error
	Delete(id uuid.UUID) error
}

type networkPlanRepository struct {
	db *gorm.DB
}

func NewNetworkPlanRepository(db *gorm.DB) NetworkPlanRepository {
	return &networkPlanRepository{db: db}
}

func (r *networkPlanRepository) Create(plan *models.NetworkPlan) error {
	return r.db.Create(plan).Error
}

func (r *networkPlanRepository) FindAll() ([]models.NetworkPlan, error) {
	var plans []models.NetworkPlan
	err := r.db.Preload("Router").Order("created_at desc").Find(&plans).Error
	return plans, err
}

func (r *networkPlanRepository) FindByID(id uuid.UUID) (*models.NetworkPlan, error) {
	var plan models.NetworkPlan
	err := r.db.Preload("Router").First(&plan, "id = ?", id).Error
	return &plan, err
}

func (r *networkPlanRepository) Update(plan *models.NetworkPlan) error {
	return r.db.Save(plan).Error
}

func (r *networkPlanRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.NetworkPlan{}, "id = ?", id).Error
}
