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
	Create(input models.Payment) (*models.Payment, error)
	Update(id string, input models.Payment) (*models.Payment, error)
	Delete(id string) error
}

type paymentService struct {
	repo     repositories.PaymentRepository
	subcRepo repositories.SubscriptionRepository
	billRepo repositories.BillRepository
}

func NewPaymentService(
	repo repositories.PaymentRepository,
	subcRepo repositories.SubscriptionRepository,
	billRepo repositories.BillRepository,
) PaymentService {
	return &paymentService{
		repo,
		subcRepo,
		billRepo,
	}
}

func (s *paymentService) GetAll() ([]models.Payment, error) {
	return s.repo.FindAll()
}

func (s *paymentService) GetByID(id string) (models.Payment, error) {
	return s.repo.FindByID(id)
}

func (s *paymentService) Create(input models.Payment) (*models.Payment, error) {
	bill, err := s.billRepo.FindByID(input.BillID.String())
	if err != nil {
		return nil, err
	}

	input.ID = uuid.New()
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()
	err = s.repo.Create(&input)

	if input.Status == "confirmed" {
		nbill, nsubs, nerr := s.UpdateBillAndSubs(input, bill)
		if nerr != nil {
			return nil, nerr
		}
		input.Bill = *nbill
		input.Bill.Subscription = *nsubs
	}
	return &input, err
}

func (s *paymentService) Update(id string, input models.Payment) (*models.Payment, error) {
	payment, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	bill, err := s.billRepo.FindByID(input.BillID.String())
	if err != nil {
		return nil, err
	}

	payment.RefID = input.RefID
	payment.PaymentDate = input.PaymentDate
	payment.DueDate = input.DueDate
	payment.Method = input.Method
	payment.Amount = input.Amount
	payment.Status = input.Status
	payment.UpdatedAt = time.Now()

	err = s.repo.Update(&payment)
	if err != nil {
		return nil, err
	}

	// Update Bill And Subscription
	if payment.Status == "confirmed" {
		nbill, nsubs, nerr := s.UpdateBillAndSubs(payment, bill)
		if nerr != nil {
			return nil, nerr
		}
		payment.Bill = *nbill
		payment.Bill.Subscription = *nsubs
	}

	return &payment, err
}

func (s *paymentService) Delete(id string) error {
	payment, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// Rollback bill and subscribe
	if payment.Status == "confirmed" {
		_, _, err := s.RollbackBillAndSubs(payment.Bill)
		if err != nil {
			return err
		}
	}
	return s.repo.Delete(id)
}

func (s *paymentService) UpdateBillAndSubs(input models.Payment, bill models.Bill) (*models.Bill, *models.Subscription, error) {

	// Update bill status to paid
	bill.Status = "paid"
	err := s.billRepo.Update(&bill)
	if err != nil {
		return nil, nil, err
	}

	// Update subscription duration
	subs, err := s.subcRepo.FindByID(bill.SubscriptionID)
	if err != nil {
		return nil, nil, err
	}
	endSubs := subs.EndDate
	subs.StartDate = endSubs
	subs.EndDate = endSubs.AddDate(0, 1, 0)

	err = s.subcRepo.Update(subs)
	if err != nil {
		return nil, nil, err
	}
	return &bill, subs, nil
}

func (s *paymentService) RollbackBillAndSubs(bill models.Bill) (*models.Bill, *models.Subscription, error) {
	// Update bill status to paid
	bill.Status = "unpaid"
	err := s.billRepo.Update(&bill)
	if err != nil {
		return nil, nil, err
	}

	// Update subscription duration
	subs, err := s.subcRepo.FindByID(bill.SubscriptionID)
	if err != nil {
		return nil, nil, err
	}

	// Mundurkan 1 bulan masa berlangganan
	subs.EndDate = subs.EndDate.AddDate(0, -1, 0)
	subs.StartDate = subs.StartDate.AddDate(0, -1, 0)

	err = s.subcRepo.Update(subs)
	if err != nil {
		return nil, nil, err
	}

	return &bill, subs, nil
}
