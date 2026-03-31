package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type ProvisioningService interface {
	ListJobs(limit int) ([]models.ProvisioningJob, error)
	ListLogs(limit int) ([]models.ProvisioningLog, error)
	GetJobsByEntity(entityType string, entityID uuid.UUID, limit int) ([]models.ProvisioningJob, error)
}

type provisioningService struct {
	jobRepo repositories.ProvisioningJobRepository
	logRepo repositories.ProvisioningLogRepository
}

func NewProvisioningService(
	jobRepo repositories.ProvisioningJobRepository,
	logRepo repositories.ProvisioningLogRepository,
) ProvisioningService {
	return &provisioningService{
		jobRepo: jobRepo,
		logRepo: logRepo,
	}
}

func (s *provisioningService) ListJobs(limit int) ([]models.ProvisioningJob, error) {
	return s.jobRepo.FindAll(limit)
}

func (s *provisioningService) ListLogs(limit int) ([]models.ProvisioningLog, error) {
	return s.logRepo.FindAll(limit)
}

func (s *provisioningService) GetJobsByEntity(entityType string, entityID uuid.UUID, limit int) ([]models.ProvisioningJob, error) {
	return s.jobRepo.FindByEntity(entityType, entityID, limit)
}
