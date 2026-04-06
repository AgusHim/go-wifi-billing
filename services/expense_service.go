package services

import (
	"errors"
	"mime/multipart"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type ExpenseService interface {
	GetAll(adminID string, search string, category string, startAt string, endAt string) ([]models.Expense, error)
	GetByID(id string) (models.Expense, error)
	Create(input models.Expense, imageFile *multipart.FileHeader) (*models.Expense, error)
	Update(id string, input models.Expense, imageFile *multipart.FileHeader) (*models.Expense, error)
	Delete(id string) error
}

type expenseService struct {
	repo repositories.ExpenseRepository
}

func NewExpenseService(repo repositories.ExpenseRepository) ExpenseService {
	return &expenseService{repo: repo}
}

func (s *expenseService) GetAll(adminID string, search string, category string, startAt string, endAt string) ([]models.Expense, error) {
	adminID = strings.TrimSpace(adminID)
	search = strings.TrimSpace(search)
	category = strings.TrimSpace(category)
	startAt = strings.TrimSpace(startAt)
	endAt = strings.TrimSpace(endAt)

	var parsedAdminID *uuid.UUID
	if adminID != "" {
		uid, err := uuid.Parse(adminID)
		if err != nil {
			return nil, errors.New("invalid admin_id")
		}
		parsedAdminID = &uid
	}

	startDate, endDate, err := parseStartEndRange(startAt, endAt)
	if err != nil {
		return nil, err
	}

	return s.repo.FindAll(parsedAdminID, search, category, startDate, endDate)
}

func (s *expenseService) GetByID(id string) (models.Expense, error) {
	return s.repo.FindByID(id)
}

func (s *expenseService) Create(input models.Expense, imageFile *multipart.FileHeader) (*models.Expense, error) {
	input.ID = uuid.New()
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()

	if imageFile != nil {
		url, publicID, err := lib.UploadExpenseImage(imageFile)
		if err != nil {
			return nil, err
		}
		input.ProofImageUrl = url
		input.ProofImagePublicId = publicID
	}

	if err := s.repo.Create(&input); err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *expenseService) Update(id string, input models.Expense, imageFile *multipart.FileHeader) (*models.Expense, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	existing.Title = input.Title
	existing.Category = input.Category
	existing.Amount = input.Amount
	existing.ExpenseDate = input.ExpenseDate
	existing.Description = input.Description
	existing.AdminID = input.AdminID
	existing.UpdatedAt = time.Now()

	if imageFile != nil {
		_ = lib.DeleteExpenseImage(existing.ProofImagePublicId)
		url, publicID, err := lib.UploadExpenseImage(imageFile)
		if err != nil {
			return nil, err
		}
		existing.ProofImageUrl = url
		existing.ProofImagePublicId = publicID
	}

	if err := s.repo.Update(&existing); err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *expenseService) Delete(id string) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	_ = lib.DeleteExpenseImage(existing.ProofImagePublicId)
	return s.repo.Delete(id)
}
