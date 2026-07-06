package services

import (
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Agushim/go_wifi_billing/repositories"
)

type ProvisioningWorkerService interface {
	StartScheduler()
	ProcessEligibleJobs(now time.Time) (int, error)
}

type provisioningWorkerService struct {
	jobRepo           repositories.ProvisioningJobRepository
	serviceAccountSvc ServiceAccountService
	voucherSvc        VoucherService
	running           int32
}

func NewProvisioningWorkerService(
	jobRepo repositories.ProvisioningJobRepository,
	serviceAccountSvc ServiceAccountService,
	voucherSvc VoucherService,
) ProvisioningWorkerService {
	return &provisioningWorkerService{
		jobRepo:           jobRepo,
		serviceAccountSvc: serviceAccountSvc,
		voucherSvc:        voucherSvc,
	}
}

func (s *provisioningWorkerService) StartScheduler() {
	if !provisioningEnabled() {
		return
	}

	interval := provisioningWorkerInterval()
	go func() {
		s.runOnce()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.runOnce()
		}
	}()
}

func (s *provisioningWorkerService) runOnce() {
	processed, err := s.ProcessEligibleJobs(time.Now())
	if err != nil {
		log.Printf("[provisioning-worker] failed to process retry jobs: %v", err)
		return
	}
	if processed > 0 {
		log.Printf("[provisioning-worker] processed %d eligible provisioning job(s)", processed)
	}
}

func (s *provisioningWorkerService) ProcessEligibleJobs(now time.Time) (int, error) {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return 0, nil
	}
	defer atomic.StoreInt32(&s.running, 0)

	jobs, err := s.jobRepo.FindEligibleForRetry(now, provisioningMaxAttempts(), provisioningRetryBatchSize())
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, job := range jobs {
		switch strings.TrimSpace(strings.ToLower(job.EntityType)) {
		case "service_account":
			if s.serviceAccountSvc != nil {
				s.serviceAccountSvc.ProcessProvisioningJob(job.ID)
				processed++
			}
		case "voucher":
			if s.voucherSvc != nil {
				s.voucherSvc.ProcessProvisioningJob(job.ID)
				processed++
			}
		default:
			log.Printf("[provisioning-worker] skipped unsupported entity_type=%q job=%s", job.EntityType, job.ID)
		}
	}

	return processed, nil
}
