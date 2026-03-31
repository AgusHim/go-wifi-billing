package services

import (
	"errors"
	"strings"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type NetworkPlanService interface {
	Create(plan *models.NetworkPlan) (*models.NetworkPlan, error)
	GetAll() ([]models.NetworkPlan, error)
	GetByID(id uuid.UUID) (*models.NetworkPlan, error)
	Update(id uuid.UUID, input *models.NetworkPlan) (*models.NetworkPlan, error)
	Delete(id uuid.UUID) error
}

type networkPlanService struct {
	repo repositories.NetworkPlanRepository
}

func NewNetworkPlanService(repo repositories.NetworkPlanRepository) NetworkPlanService {
	return &networkPlanService{repo: repo}
}

func (s *networkPlanService) Create(plan *models.NetworkPlan) (*models.NetworkPlan, error) {
	if err := validateNetworkPlan(plan); err != nil {
		return nil, err
	}
	if err := s.repo.Create(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *networkPlanService) GetAll() ([]models.NetworkPlan, error) {
	return s.repo.FindAll()
}

func (s *networkPlanService) GetByID(id uuid.UUID) (*models.NetworkPlan, error) {
	return s.repo.FindByID(id)
}

func (s *networkPlanService) Update(id uuid.UUID, input *models.NetworkPlan) (*models.NetworkPlan, error) {
	if err := validateNetworkPlan(input); err != nil {
		return nil, err
	}
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	existing.Name = input.Name
	existing.ServiceType = input.ServiceType
	existing.RouterID = input.RouterID
	existing.MikrotikProfileName = input.MikrotikProfileName
	existing.AddressPool = input.AddressPool
	existing.DownloadKbps = input.DownloadKbps
	existing.UploadKbps = input.UploadKbps
	existing.Description = input.Description
	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *networkPlanService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func validateNetworkPlan(plan *models.NetworkPlan) error {
	if plan == nil {
		return errors.New("invalid network plan payload")
	}
	if strings.TrimSpace(plan.Name) == "" {
		return errors.New("network plan name is required")
	}
	if strings.TrimSpace(plan.ServiceType) == "" {
		plan.ServiceType = "pppoe"
	}
	return nil
}
