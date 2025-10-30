package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"

	"github.com/google/uuid"
)

type PackageService interface {
	Create(pkg *models.Package) error
	GetAll() ([]models.Package, error)
	GetByID(id uuid.UUID) (*models.Package, error)
	Update(id uuid.UUID, input *models.Package) error
	Delete(id uuid.UUID) error
}

type packageService struct {
	repo repositories.PackageRepository
}

func NewPackageService(repo repositories.PackageRepository) PackageService {
	return &packageService{repo}
}

func (s *packageService) Create(pkg *models.Package) error {
	return s.repo.Create(pkg)
}

func (s *packageService) GetAll() ([]models.Package, error) {
	return s.repo.FindAll()
}

func (s *packageService) GetByID(id uuid.UUID) (*models.Package, error) {
	return s.repo.FindByID(id)
}

func (s *packageService) Update(id uuid.UUID, input *models.Package) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	existing.Category = input.Category
	existing.Name = input.Name
	existing.SpeedMbps = input.SpeedMbps
	existing.QuotaGB = input.QuotaGB
	existing.Price = input.Price
	existing.Description = input.Description
	return s.repo.Update(existing)
}

func (s *packageService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
