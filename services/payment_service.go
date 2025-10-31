package services

import (
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type PaymentService interface {
	GetAll() ([]models.Payment, error)
	GetByID(id string) (models.Payment, error)
	Create(input models.Payment) (models.Payment, error)
	Update(id string, input models.Payment) (models.Payment, error)
	Delete(id string) error
}

type paymentService struct {
	repo repositories.PaymentRepository
}

func NewPaymentService(repo repositories.PaymentRepository) PaymentService {
	return &paymentService{repo}
}

func (s *paymentService) GetAll() ([]models.Payment, error) {
	return s.repo.FindAll()
}

func (s *paymentService) GetByID(id string) (models.Payment, error) {
	return s.repo.FindByID(id)
}

func (s *paymentService) Create(input models.Payment) (models.Payment, error) {
	input.ID = uuid.New()
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()
	err := s.repo.Create(&input)
	return input, err
}

func (s *paymentService) Update(id string, input models.Payment) (models.Payment, error) {
	payment, err := s.repo.FindByID(id)
	if err != nil {
		return payment, err
	}

	payment.RefID = input.RefID
	payment.PaymentDate = input.PaymentDate
	payment.DueDate = input.DueDate
	payment.Method = input.Method
	payment.Amount = input.Amount
	payment.Status = input.Status
	payment.UpdatedAt = time.Now()

	err = s.repo.Update(&payment)
	return payment, err
}

func (s *paymentService) Delete(id string) error {
	return s.repo.Delete(id)
}
