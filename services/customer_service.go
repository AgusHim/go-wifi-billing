package services

import (
	"errors"
	"time"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"

	"github.com/google/uuid"
)

type CustomerService interface {
	Create(customer *dto.CreateCustomerDTO) (*models.Customer, error)
	GetAll() ([]models.Customer, error)
	GetByID(id uuid.UUID) (*models.Customer, error)
	Update(id uuid.UUID, input *models.Customer) error
	Delete(id uuid.UUID) error
}

type customerService struct {
	repo                repositories.CustomerRepository
	userService         UserService
	subscriptionService SubscriptionService
}

func NewCustomerService(repo repositories.CustomerRepository, userService UserService, subscriptionService SubscriptionService) CustomerService {
	return &customerService{repo, userService, subscriptionService}
}

func (s *customerService) Create(body *dto.CreateCustomerDTO) (*models.Customer, error) {
	user, _ := s.userService.CheckIsRegistered(*body.Email, *body.Phone)
	if user != nil {
		return nil, errors.New("email or phone already registered")
	}

	user, err := s.userService.Register(dto.RegisterDTO{
		Name:       *body.Name,
		Email:      *body.Email,
		Phone:      *body.Phone,
		Password:   *body.Password,
		Role:       "customer",
		CoverageID: body.CoverageID,
	})
	if err != nil {
		return nil, err
	}

	customer := &models.Customer{
		ID:            uuid.New(),
		UserID:        user.ID,
		CoverageID:    uuid.MustParse(*body.CoverageID),
		OdcID:         uuid.MustParse(*body.OdcID),
		OdpID:         uuid.MustParse(*body.OdpID),
		PortOdp:       *body.PortOdp,
		ServiceNumber: *body.ServiceNumber,
		Card:          *body.Card,
		IDCard:        *body.IDCard,
		IsSendWa:      *body.IsSendWA,
		Status:        *body.Status,
		Address:       *body.Address,
		Latitude:      *body.Latitude,
		Longitude:     *body.Longitude,
		Mode:          *body.Mode,
		IDPPOE:        *body.IDPPOE,
		ProfilePPOE:   *body.ProfilePPOE,
	}
	err = s.repo.Create(customer)
	if err != nil {
		s.userService.Delete(user.ID.String())
		return nil, err
	}

	now := time.Now()
	err = s.subscriptionService.Create(&models.Subscription{
		CustomerID:   customer.ID,
		PackageID:    uuid.MustParse(*body.PackageID),
		IsIncludePPN: *body.IsIncludePPN,
		PaymentType:  *body.PaymentType,
		DueDay:       *body.DueDay,
		PeriodType:   *body.PeriodType,
		StartDate:    now,
		EndDate:      now.AddDate(0, 1, 0),
		Status:       *body.Status,
		AutoRenew:    true,
	})
	if err != nil {
		s.userService.Delete(user.ID.String())
		return nil, err
	}

	return customer, nil
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
