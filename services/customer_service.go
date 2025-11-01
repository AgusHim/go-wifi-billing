package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"

	"github.com/google/uuid"
)

type CustomerService interface {
	Create(customer *models.Customer) error
	GetAll() ([]models.Customer, error)
	GetByID(id uuid.UUID) (*models.Customer, error)
	Update(id uuid.UUID, input *models.Customer) error
	Delete(id uuid.UUID) error
}

type customerService struct {
	repo repositories.CustomerRepository
}

func NewCustomerService(repo repositories.CustomerRepository) CustomerService {
	return &customerService{repo}
}

func (s *customerService) Create(customer *models.Customer) error {
	return s.repo.Create(customer)
}

func (s *customerService) GetAll() ([]models.Customer, error) {
	return s.repo.FindAll()
}

func (s *customerService) GetByID(id uuid.UUID) (*models.Customer, error) {
	return s.repo.FindByID(id)
}

func (s *customerService) Update(id uuid.UUID, input *models.Customer) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	existing.CoverageID = input.CoverageID
	existing.OdcID = input.OdcID
	existing.OdpID = input.OdpID
	existing.ServiceNumber = input.ServiceNumber
	existing.Card = input.Card
	existing.IDCard = input.IDCard
	existing.IsIncludePPN = input.IsIncludePPN
	existing.PaymentType = input.PaymentType
	existing.DueDay = input.DueDay
	existing.IsSendWa = input.IsSendWa
	existing.Status = input.Status
	existing.Address = input.Address
	existing.Latitude = input.Latitude
	existing.Longitude = input.Longitude
	existing.Mode = input.Mode
	existing.IDPPOE = input.IDPPOE
	existing.ProfilePPOE = input.ProfilePPOE

	return s.repo.Update(existing)
}

func (s *customerService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
