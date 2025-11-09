package services

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BillService interface {
	GetAll() ([]models.Bill, error)
	GetByID(id string) (models.Bill, error)
	Create(input models.Bill) (models.Bill, error)
	Update(id string, input models.Bill) (models.Bill, error)
	Delete(id string) error
	GenerateMonthlyBills() error
}

type billService struct {
	repo    repositories.BillRepository
	subRepo repositories.SubscriptionRepository
}

func NewBillService(repo repositories.BillRepository, subRepo repositories.SubscriptionRepository) BillService {
	return &billService{repo, subRepo}
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

func (s *billService) GenerateMonthlyBills() error {
	status := "active"
	subs, err := s.subRepo.FindForBill(nil, &status, true)
	log.Printf("Found %d active subscriptions", len(subs))
	if err != nil {
		return fmt.Errorf("failed to fetch subscriptions: %w", err)
	}

	currentMonth := int(time.Now().Month())
	currentYear := time.Now().Year()

	log.Printf("Generating monthly bills for %d-%02d", currentYear, currentMonth)

	for _, sub := range subs {
		// Cek apakah sudah ada bill bulan ini
		existing, err := s.repo.FindBillByCustomerAndMonth(sub.CustomerID.String(), currentMonth, currentYear)

		if err == nil && existing != nil {
			continue // sudah ada bill bulan ini
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		billDate := time.Now()
		dueDate := time.Date(currentYear, time.Month(currentMonth), sub.DueDay, 23, 59, 59, 0, time.Local)
		if dueDate.Month() != time.Month(currentMonth) {
			dueDate = time.Date(currentYear, time.Month(currentMonth)+1, 1, 23, 59, 59, 0, time.Local).AddDate(0, 0, -1)
		}

		amount := sub.Package.Price
		ppn := int(float64(sub.Package.Price) * 0.11)
		if sub.IsIncludePPN {
			amount = int(float64(amount) * 1.11) // tambahkan 11% PPN
		}

		// ✅ Generate unique code (001–500)
		uniqueCode := 0
		if sub.IsActiveUniqueCode {
			uniqueCode = rand.Intn(799) + 1 // hasil 1–500
			amount += uniqueCode            // tambahkan ke total
		}

		bill := &models.Bill{
			ID:             uuid.New(),
			PublicID:       fmt.Sprintf("%d%02d-%s", currentYear, currentMonth, uuid.NewString()[:6]),
			SubscriptionID: sub.ID,
			CustomerID:     sub.CustomerID,
			BillDate:       billDate,
			DueDate:        dueDate,
			Amount:         amount,
			PPN:            ppn,
			UniqueCode:     uniqueCode,
			Status:         "unpaid",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		log.Printf("Creating bill %s for customer %s: %v", bill.PublicID, sub.CustomerID.String(), bill)

		if err := s.repo.Create(bill); err != nil {
			return fmt.Errorf("failed to create bill: %w", err)
		}
	}

	return nil
}
