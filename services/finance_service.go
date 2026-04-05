package services

import (
	"errors"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type FinanceSummaryResult struct {
	TotalIncome  int64  `json:"total_income"`
	TotalExpense int64  `json:"total_expense"`
	NetProfit    int64  `json:"net_profit"`
	PeriodStart  string `json:"period_start"`
	PeriodEnd    string `json:"period_end"`
}

type FinanceBySubscriptionResult struct {
	ByMethod  []repositories.IncomeByMethodRow  `json:"by_method"`
	ByPackage []repositories.IncomeByPackageRow `json:"by_package"`
}

type FinanceService interface {
	GetSummary(adminID string, startAt string, endAt string) (FinanceSummaryResult, error)
	GetMonthly(months int) ([]repositories.FinanceMonthlyRow, error)
	GetBySubscription(adminID string, startAt string, endAt string) (FinanceBySubscriptionResult, error)
}

type financeService struct {
	repo repositories.FinanceRepository
}

func NewFinanceService(repo repositories.FinanceRepository) FinanceService {
	return &financeService{repo: repo}
}

func (s *financeService) GetSummary(adminID string, startAt string, endAt string) (FinanceSummaryResult, error) {
	adminID = strings.TrimSpace(adminID)
	startAt = strings.TrimSpace(startAt)
	endAt = strings.TrimSpace(endAt)

	var parsedAdminID *uuid.UUID
	if adminID != "" {
		uid, err := uuid.Parse(adminID)
		if err != nil {
			return FinanceSummaryResult{}, errors.New("invalid admin_id")
		}
		parsedAdminID = &uid
	}

	startDate, endDate, err := parseStartEndRange(startAt, endAt)
	if err != nil {
		return FinanceSummaryResult{}, err
	}

	data, err := s.repo.GetSummary(startDate, endDate, parsedAdminID)
	if err != nil {
		return FinanceSummaryResult{}, err
	}

	periodStart := startAt
	periodEnd := endAt
	if periodStart == "" {
		periodStart = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if periodEnd == "" {
		periodEnd = time.Now().Format("2006-01-02")
	}

	return FinanceSummaryResult{
		TotalIncome:  data.TotalIncome,
		TotalExpense: data.TotalExpense,
		NetProfit:    data.NetProfit,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
	}, nil
}

func (s *financeService) GetMonthly(months int) ([]repositories.FinanceMonthlyRow, error) {
	if months <= 0 || months > 24 {
		months = 12
	}
	return s.repo.GetMonthly(months)
}

func (s *financeService) GetBySubscription(adminID string, startAt string, endAt string) (FinanceBySubscriptionResult, error) {
	adminID = strings.TrimSpace(adminID)
	startAt = strings.TrimSpace(startAt)
	endAt = strings.TrimSpace(endAt)

	var parsedAdminID *uuid.UUID
	if adminID != "" {
		uid, err := uuid.Parse(adminID)
		if err != nil {
			return FinanceBySubscriptionResult{}, errors.New("invalid admin_id")
		}
		parsedAdminID = &uid
	}

	startDate, endDate, err := parseStartEndRange(startAt, endAt)
	if err != nil {
		return FinanceBySubscriptionResult{}, err
	}

	byMethod, err := s.repo.GetIncomeByMethod(startDate, endDate, parsedAdminID)
	if err != nil {
		return FinanceBySubscriptionResult{}, err
	}

	byPackage, err := s.repo.GetIncomeByPackage(startDate, endDate)
	if err != nil {
		return FinanceBySubscriptionResult{}, err
	}

	return FinanceBySubscriptionResult{
		ByMethod:  byMethod,
		ByPackage: byPackage,
	}, nil
}
