package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PackageRepository interface {
	Create(pkg *models.Package) error
	FindAll() ([]models.Package, error)
	FindByID(id uuid.UUID) (*models.Package, error)
	Update(pkg *models.Package) error
	Delete(id uuid.UUID) error
}

type packageRepository struct {
	db *gorm.DB
}

func NewPackageRepository(db *gorm.DB) PackageRepository {
	return &packageRepository{db}
}

func (r *packageRepository) Create(pkg *models.Package) error {
	return r.db.Create(pkg).Error
}

func (r *packageRepository) FindAll() ([]models.Package, error) {
	var pkgs []models.Package
	err := r.db.Find(&pkgs).Error
	return pkgs, err
}

func (r *packageRepository) FindByID(id uuid.UUID) (*models.Package, error) {
	var pkg models.Package
	err := r.db.First(&pkg, "id = ?", id).Error
	return &pkg, err
}

func (r *packageRepository) Update(pkg *models.Package) error {
	return r.db.Save(pkg).Error
}

func (r *packageRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Package{}, "id = ?", id).Error
}
