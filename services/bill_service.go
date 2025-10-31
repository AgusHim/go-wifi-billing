package services

import (
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type BillService interface {
	GetAll() ([]models.Bill, error)
	GetByID(id string) (models.Bill, error)
	Create(input models.Bill) (models.Bill, error)
	Update(id string, input models.Bill) (models.Bill, error)
	Delete(id string) error
}

type billService struct {
	repo repositories.BillRepository
}

func NewBillService(repo repositories.BillRepository) BillService {
	return &billService{repo}
}

func (s *billService) GetAll() ([]models.Bill, error) {
	return s.repo.FindAll()
}

func (s *billService) GetByID(id string) (models.Bill, error) {
	return s.repo.FindByID(id)
}

func (s *billService) Create(input models.Bill) (models.Bill, error) {
	input.ID = uuid.New()
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()
	err := s.repo.Create(&input)
	return input, err
}

func (s *billService) Update(id string, input models.Bill) (models.Bill, error) {
	bill, err := s.repo.FindByID(id)
	if err != nil {
		return bill, err
	}
	bill.Amount = input.Amount
	bill.Status = input.Status
	bill.BillDate = input.BillDate
	bill.DueDate = input.DueDate
	bill.TerminatedDate = input.TerminatedDate
	bill.UpdatedAt = time.Now()
	err = s.repo.Update(&bill)
	return bill, err
}

func (s *billService) Delete(id string) error {
	return s.repo.Delete(id)
}
