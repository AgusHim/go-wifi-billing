package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"

	"github.com/google/uuid"
)

type CustomerService interface {
	Create(customer *dto.CreateCustomerDTO) (*models.Customer, error)
	GetAll(page, limit int, search string, adminID string, coverageID string) ([]models.Customer, int64, error)
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

		// Cegah double-insert customer saat user lama direstore.
		// Why: soft-delete hanya menyentuh user; customer lama bisa sudah ada atau race dengan request kembar.
		if existing, _ := s.repo.FindByUserID(user.ID); existing != nil && existing.ID != uuid.Nil {
			return nil, errors.New("customer already exists for this user")
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

	coverageID, err := parseRequiredUUID(body.CoverageID, "coverage_id")
	if err != nil {
		return nil, err
	}
	adminID, err := parseRequiredUUID(body.AdminID, "admin_id")
	if err != nil {
		return nil, err
	}
	odcID, err := parseOptionalUUID(body.OdcID, "odc_id")
	if err != nil {
		return nil, err
	}
	odpID, err := parseOptionalUUID(body.OdpID, "odp_id")
	if err != nil {
		return nil, err
	}
	portOdp := ""
	if body.PortOdp != nil {
		portOdp = *body.PortOdp
	}
	description := ""
	if body.Description != nil {
		description = *body.Description
	}

	// Auto-generate service number unik per coverage; input dari client diabaikan.
	serviceNumber, err := s.repo.NextServiceNumber(coverageID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service_number: %w", err)
	}

	// Buat customer baru
	customer := &models.Customer{
		ID:            uuid.New(),
		UserID:        user.ID,
		User:          user,
		CoverageID:    coverageID,
		OdcID:         odcID,
		OdpID:         odpID,
		PortOdp:       portOdp,
		ServiceNumber: serviceNumber,
		Card:          *body.Card,
		IDCard:        *body.IDCard,
		IsSendWa:      *body.IsSendWA,
		Status:        *body.Status,
		Address:       *body.Address,
		Description:   description,
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

func (s *customerService) GetAll(page, limit int, search string, adminID string, coverageID string) ([]models.Customer, int64, error) {
	adminID = strings.TrimSpace(adminID)
	coverageID = strings.TrimSpace(coverageID)

	var parsedAdminID *uuid.UUID
	if adminID != "" {
		uid, err := uuid.Parse(adminID)
		if err != nil {
			return nil, 0, errors.New("invalid admin_id")
		}
		parsedAdminID = &uid
	}

	var parsedCoverageID *uuid.UUID
	if coverageID != "" {
		cid, err := uuid.Parse(coverageID)
		if err != nil {
			return nil, 0, errors.New("invalid coverage_id")
		}
		parsedCoverageID = &cid
	}

	return s.repo.FindAll(page, limit, search, parsedAdminID, parsedCoverageID)
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

	coverageID, err := parseRequiredUUID(input.CoverageID, "coverage_id")
	if err != nil {
		return nil, err
	}
	existing.CoverageID = coverageID
	if input.OdcID != nil {
		odcID, err := parseOptionalUUID(input.OdcID, "odc_id")
		if err != nil {
			return nil, err
		}
		existing.OdcID = odcID
	}
	if input.OdpID != nil {
		odpID, err := parseOptionalUUID(input.OdpID, "odp_id")
		if err != nil {
			return nil, err
		}
		existing.OdpID = odpID
	}
	if input.PortOdp != nil {
		existing.PortOdp = *input.PortOdp
	}
	// service_number dikunci; selalu auto-generated dan tidak boleh ditimpa via API.
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
	adminID, err := parseRequiredUUID(input.AdminID, "admin_id")
	if err != nil {
		return nil, err
	}
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

func parseRequiredUUID(value *string, field string) (uuid.UUID, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return uuid.Nil, fmt.Errorf("%s is required", field)
	}
	parsed, err := uuid.Parse(*value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", field)
	}
	return parsed, nil
}

func parseOptionalUUID(value *string, field string) (*uuid.UUID, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s", field)
	}
	return &parsed, nil
}
