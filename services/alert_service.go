package services

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type AlertEvaluationSummary struct {
	EvaluatedAt      time.Time `json:"evaluated_at"`
	CreatedOrUpdated int       `json:"created_or_updated"`
}

type AlertService interface {
	Evaluate() (*AlertEvaluationSummary, error)
	GetAlerts(status string, limit int) ([]models.AlertEvent, error)
	Acknowledge(id uuid.UUID) error
	Resolve(id uuid.UUID) error
	CountOpenAlerts() (int64, error)
	StartAlertScheduler()
}

type alertService struct {
	alertRepo          repositories.AlertRepository
	routerRepo         repositories.RouterRepository
	nocRepo            repositories.NOCRepository
	reconciliationRepo repositories.NOCRepository
}

func NewAlertService(alertRepo repositories.AlertRepository, routerRepo repositories.RouterRepository, nocRepo repositories.NOCRepository) AlertService {
	return &alertService{
		alertRepo:          alertRepo,
		routerRepo:         routerRepo,
		nocRepo:            nocRepo,
		reconciliationRepo: nocRepo,
	}
}

func (s *alertService) Evaluate() (*AlertEvaluationSummary, error) {
	if err := s.alertRepo.EnsureDefaultRules(defaultAlertRules()); err != nil {
		return nil, err
	}
	rules, err := s.alertRepo.FindEnabledRules()
	if err != nil {
		return nil, err
	}
	ruleByKey := make(map[string]models.AlertRule, len(rules))
	for _, rule := range rules {
		ruleByKey[rule.RuleKey] = rule
	}

	summary := &AlertEvaluationSummary{EvaluatedAt: time.Now()}
	if err := s.evaluateRouterAlerts(ruleByKey, summary); err != nil {
		return nil, err
	}
	if err := s.evaluateInterfaceAlerts(ruleByKey, summary); err != nil {
		return nil, err
	}
	if err := s.evaluateProvisioningAlerts(ruleByKey, summary); err != nil {
		return nil, err
	}
	if err := s.evaluateReconciliationAlerts(ruleByKey, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func (s *alertService) GetAlerts(status string, limit int) ([]models.AlertEvent, error) {
	if strings.TrimSpace(status) == "" {
		status = "open"
	}
	return s.alertRepo.FindAlerts(status, limit)
}

func (s *alertService) Acknowledge(id uuid.UUID) error {
	return s.alertRepo.UpdateAlertStatus(id, "acknowledged")
}

func (s *alertService) Resolve(id uuid.UUID) error {
	return s.alertRepo.UpdateAlertStatus(id, "resolved")
}

func (s *alertService) CountOpenAlerts() (int64, error) {
	return s.alertRepo.CountOpenAlerts()
}

func (s *alertService) StartAlertScheduler() {
	if !nocAlertsEnabled() {
		return
	}
	interval := resolveAlertEvaluationInterval()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := s.Evaluate(); err != nil {
				log.Printf("[alerts] evaluation failed: %v", err)
			}
		}
	}()
}

func (s *alertService) evaluateRouterAlerts(ruleByKey map[string]models.AlertRule, summary *AlertEvaluationSummary) error {
	routers, err := s.routerRepo.FindAll()
	if err != nil {
		return err
	}
	snapshots, err := s.nocRepo.GetLatestRouterSnapshots()
	if err != nil {
		return err
	}
	snapshotByRouter := make(map[uuid.UUID]models.RouterSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotByRouter[snapshot.RouterID] = snapshot
	}

	for _, router := range routers {
		if rule, ok := ruleByKey["router_down"]; ok && !strings.EqualFold(router.Status, "connected") {
			id := router.ID
			if err := s.upsert(rule, "router", &id, &router.ID, nil, fmt.Sprintf("Router %s down", router.Name), router.LastError); err != nil {
				return err
			}
			summary.CreatedOrUpdated++
		}
		snapshot, ok := snapshotByRouter[router.ID]
		if !ok {
			continue
		}
		if rule, ok := ruleByKey["high_cpu"]; ok && snapshot.CPULoad >= rule.Threshold {
			id := router.ID
			if err := s.upsert(rule, "router", &id, &router.ID, nil, fmt.Sprintf("CPU router %s tinggi", router.Name), fmt.Sprintf("CPU load %d%% >= threshold %d%%", snapshot.CPULoad, rule.Threshold)); err != nil {
				return err
			}
			summary.CreatedOrUpdated++
		}
		if rule, ok := ruleByKey["low_memory"]; ok && snapshot.TotalMemory > 0 {
			freePercent := int((snapshot.FreeMemory * 100) / snapshot.TotalMemory)
			if freePercent <= rule.Threshold {
				id := router.ID
				if err := s.upsert(rule, "router", &id, &router.ID, nil, fmt.Sprintf("Memory router %s rendah", router.Name), fmt.Sprintf("Free memory %d%% <= threshold %d%%", freePercent, rule.Threshold)); err != nil {
					return err
				}
				summary.CreatedOrUpdated++
			}
		}
		if rule, ok := ruleByKey["active_session_drop"]; ok {
			latest, err := s.alertRepo.GetLatestTwoRouterSnapshots(router.ID)
			if err != nil {
				return err
			}
			if len(latest) == 2 {
				current := latest[0].ActivePPPoE + latest[0].ActiveHotspot
				previous := latest[1].ActivePPPoE + latest[1].ActiveHotspot
				if previous > 0 && current*100/previous <= 100-rule.Threshold {
					id := router.ID
					if err := s.upsert(rule, "router", &id, &router.ID, nil, fmt.Sprintf("Active session router %s turun drastis", router.Name), fmt.Sprintf("Sessions turun dari %d ke %d", previous, current)); err != nil {
						return err
					}
					summary.CreatedOrUpdated++
				}
			}
		}
	}
	return nil
}

func (s *alertService) evaluateInterfaceAlerts(ruleByKey map[string]models.AlertRule, summary *AlertEvaluationSummary) error {
	rule, ok := ruleByKey["interface_down"]
	if !ok {
		return nil
	}
	issues, err := s.nocRepo.GetRecentInterfaceIssues(time.Now().Add(-time.Duration(rule.WindowMinutes)*time.Minute), 50)
	if err != nil {
		return err
	}
	for _, issue := range issues {
		id := issue.ID
		routerID := issue.RouterID
		title := fmt.Sprintf("Interface %s down/disabled", issue.Name)
		description := fmt.Sprintf("Interface %s router %s running=%t disabled=%t", issue.Name, issue.RouterID, issue.Running, issue.Disabled)
		if err := s.upsert(rule, "interface", &id, &routerID, nil, title, description); err != nil {
			return err
		}
		summary.CreatedOrUpdated++
	}
	return nil
}

func (s *alertService) evaluateProvisioningAlerts(ruleByKey map[string]models.AlertRule, summary *AlertEvaluationSummary) error {
	rule, ok := ruleByKey["provisioning_failed"]
	if !ok {
		return nil
	}
	jobs, err := s.alertRepo.FindRecentFailedProvisioningJobs(time.Now().Add(-time.Duration(rule.WindowMinutes) * time.Minute))
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.AttemptCount < rule.Threshold {
			continue
		}
		id := job.ID
		if err := s.upsert(rule, "provisioning_job", &id, nil, nil, "Provisioning gagal berulang", fmt.Sprintf("Job %s gagal %d kali: %s", job.ID, job.AttemptCount, job.ErrorMessage)); err != nil {
			return err
		}
		summary.CreatedOrUpdated++
	}
	return nil
}

func (s *alertService) evaluateReconciliationAlerts(ruleByKey map[string]models.AlertRule, summary *AlertEvaluationSummary) error {
	rule, ok := ruleByKey["billing_mikrotik_mismatch"]
	if !ok {
		return nil
	}
	findings, err := s.reconciliationRepo.FindReconciliationFindings("open")
	if err != nil {
		return err
	}
	for _, finding := range findings {
		id := finding.ID
		routerID := finding.RouterID
		if err := s.upsert(rule, "reconciliation_finding", &id, &routerID, finding.ServiceAccountID, "Mismatch billing vs MikroTik", finding.Description); err != nil {
			return err
		}
		summary.CreatedOrUpdated++
	}
	return nil
}

func (s *alertService) upsert(rule models.AlertRule, entityType string, entityID *uuid.UUID, routerID *uuid.UUID, serviceAccountID *uuid.UUID, title string, description string) error {
	ruleID := rule.ID
	now := time.Now()
	return s.alertRepo.UpsertOpenAlert(&models.AlertEvent{
		AlertRuleID:      &ruleID,
		RuleKey:          rule.RuleKey,
		EntityType:       entityType,
		EntityID:         entityID,
		RouterID:         routerID,
		ServiceAccountID: serviceAccountID,
		Severity:         rule.Severity,
		Status:           "open",
		Title:            title,
		Description:      description,
		FirstSeenAt:      now,
		LastSeenAt:       now,
	})
}

func defaultAlertRules() []models.AlertRule {
	return []models.AlertRule{
		{RuleKey: "router_down", Name: "Router Down", Severity: "critical", Enabled: true, Threshold: 5, WindowMinutes: 5},
		{RuleKey: "high_cpu", Name: "High CPU", Severity: "warning", Enabled: true, Threshold: 85, WindowMinutes: 5},
		{RuleKey: "low_memory", Name: "Low Memory", Severity: "warning", Enabled: true, Threshold: 10, WindowMinutes: 5},
		{RuleKey: "interface_down", Name: "Interface Down", Severity: "warning", Enabled: true, Threshold: 1, WindowMinutes: 15},
		{RuleKey: "provisioning_failed", Name: "Provisioning Failed", Severity: "high", Enabled: true, Threshold: 3, WindowMinutes: 60},
		{RuleKey: "billing_mikrotik_mismatch", Name: "Billing MikroTik Mismatch", Severity: "high", Enabled: true, Threshold: 1, WindowMinutes: 60},
		{RuleKey: "active_session_drop", Name: "Active Session Drop", Severity: "warning", Enabled: true, Threshold: 50, WindowMinutes: 15},
	}
}

func nocAlertsEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_NOC_ALERTS_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func resolveAlertEvaluationInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv("NOC_ALERT_EVALUATION_INTERVAL"))
	if value == "" {
		return 5 * time.Minute
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 5 * time.Minute
	}
	return duration
}
