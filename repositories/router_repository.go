package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterRepository interface {
	Create(router *models.Router) error
	FindAll() ([]models.Router, error)
	FindByID(id uuid.UUID) (*models.Router, error)
	Update(router *models.Router) error
	Delete(id uuid.UUID) error
}

type routerRepository struct {
	db *gorm.DB
}

func NewRouterRepository(db *gorm.DB) RouterRepository {
	return &routerRepository{db: db}
}

func (r *routerRepository) Create(router *models.Router) error {
	return r.db.Create(router).Error
}

func (r *routerRepository) FindAll() ([]models.Router, error) {
	var routers []models.Router
	err := r.db.Order("created_at desc").Find(&routers).Error
	return routers, err
}

func (r *routerRepository) FindByID(id uuid.UUID) (*models.Router, error) {
	var router models.Router
	err := r.db.First(&router, "id = ?", id).Error
	return &router, err
}

func (r *routerRepository) Update(router *models.Router) error {
	return r.db.Save(router).Error
}

func (r *routerRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Router{}, "id = ?", id).Error
}
