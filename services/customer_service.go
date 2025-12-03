package services

import (
	"errors"
	"fmt"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"

	"github.com/google/uuid"
)

type CustomerService interface {
	Create(customer *dto.CreateCustomerDTO) (*models.Customer, error)
	GetAll(page, limit int, search string) ([]models.Customer, int64, error)
	GetByID(id uuid.UUID) (*models.Customer, error)
	FindByUserID(userID uuid.UUID) (*models.Customer, error)
	Update(id uuid.UUID, input *dto.CreateCustomerDTO) (*models.Customer, error)
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
	// Cek apakah user sudah pernah terdaftar (termasuk yang soft delete)
	user, _ := s.userService.CheckIsRegistered(*body.Email, *body.Phone)

	if user != nil {
		if !user.DeletedAt.Valid {
			// User masih aktif → tidak boleh duplikat
			return nil, errors.New("email or phone already registered")
		}

		// Jika user soft deleted → restore user lama
		err := s.userService.Restore(user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to restore user: %w", err)
		}
	} else {
		// Jika belum ada sama sekali → register baru
		var err error
		user, err = s.userService.Register(dto.RegisterDTO{
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
	}

	adminID := uuid.MustParse(*body.AdminID)

	// Buat customer baru
	customer := &models.Customer{
		ID:            uuid.New(),
		UserID:        user.ID,
		User:          user,
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
		AdminID:       &adminID,
	}

	if err := s.repo.Create(customer); err != nil {
		// Kalau gagal bikin customer, rollback user jika baru dibuat
		if user != nil && !user.DeletedAt.Valid {
			s.userService.Delete(user.ID.String())
		}
		return nil, err
	}

	return customer, nil
}

func (s *customerService) GetAll(page, limit int, search string) ([]models.Customer, int64, error) {
	return s.repo.FindAll(page, limit, search)
}

func (s *customerService) GetByID(id uuid.UUID) (*models.Customer, error) {
	return s.repo.FindByID(id)
}

func (s *customerService) FindByUserID(userID uuid.UUID) (*models.Customer, error) {
	return s.repo.FindByUserID(userID)
}


func (s *customerService) Update(id uuid.UUID, input *dto.CreateCustomerDTO) (*models.Customer, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	existing.CoverageID = uuid.MustParse(*input.CoverageID)
	existing.OdcID = uuid.MustParse(*input.OdcID)
	existing.OdpID = uuid.MustParse(*input.OdpID)
	existing.PortOdp = *input.PortOdp
	existing.ServiceNumber = *input.ServiceNumber
	existing.Card = *input.Card
	existing.IDCard = *input.IDCard
	existing.IsSendWa = *input.IsSendWA
	existing.Status = *input.Status
	existing.Address = *input.Address
	existing.Description = *input.Description
	existing.Latitude = *input.Latitude
	existing.Longitude = *input.Longitude
	existing.Mode = *input.Mode
	existing.IDPPOE = *input.IDPPOE
	existing.ProfilePPOE = *input.ProfilePPOE
	adminID := uuid.MustParse(*input.AdminID)
	existing.AdminID = &adminID

	user, err := s.userService.GetByID(existing.UserID.String())
	if err != nil {
		return nil, err
	}

	user.Name = *input.Name
	user.Email = *input.Email
	user.Phone = *input.Phone

	_, err = s.userService.Update(user.ID.String(), user)
	if err != nil {
		return nil, err
	}

	err = s.repo.Update(existing)
	return existing, err
}

func (s *customerService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
