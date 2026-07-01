package repositories

import (
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AlertRepository interface {
	EnsureDefaultRules(rules []models.AlertRule) error
	FindEnabledRules() ([]models.AlertRule, error)
	UpsertOpenAlert(alert *models.AlertEvent) error
	FindAlerts(status string, limit int) ([]models.AlertEvent, error)
	UpdateAlertStatus(id uuid.UUID, status string) error
	CountOpenAlerts() (int64, error)
	FindRecentFailedProvisioningJobs(since time.Time) ([]models.ProvisioningJob, error)
	GetLatestTwoRouterSnapshots(routerID uuid.UUID) ([]models.RouterSnapshot, error)
}

type alertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) AlertRepository {
	return &alertRepository{db: db}
}

func (r *alertRepository) EnsureDefaultRules(rules []models.AlertRule) error {
	for _, rule := range rules {
		var existing models.AlertRule
		err := r.db.Where("rule_key = ?", rule.RuleKey).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := r.db.Create(&rule).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *alertRepository) FindEnabledRules() ([]models.AlertRule, error) {
	var rules []models.AlertRule
	err := r.db.Where("enabled = ?", true).Find(&rules).Error
	return rules, err
}

func (r *alertRepository) UpsertOpenAlert(alert *models.AlertEvent) error {
	if alert == nil {
		return nil
	}
	query := r.db.Where("rule_key = ? AND entity_type = ? AND status IN ?", alert.RuleKey, alert.EntityType, []string{"open", "acknowledged"})
	if alert.EntityID != nil {
		query = query.Where("entity_id = ?", *alert.EntityID)
	} else {
		query = query.Where("entity_id IS NULL")
	}

	var existing models.AlertEvent
	err := query.First(&existing).Error
	if err == nil {
		existing.Severity = alert.Severity
		existing.Title = alert.Title
		existing.Description = alert.Description
		existing.LastSeenAt = alert.LastSeenAt
		return r.db.Save(&existing).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if alert.FirstSeenAt.IsZero() {
		alert.FirstSeenAt = time.Now()
	}
	if alert.LastSeenAt.IsZero() {
		alert.LastSeenAt = alert.FirstSeenAt
	}
	if alert.Status == "" {
		alert.Status = "open"
	}
	return r.db.Create(alert).Error
}

func (r *alertRepository) FindAlerts(status string, limit int) ([]models.AlertEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var alerts []models.AlertEvent
	query := r.db.
		Preload("AlertRule").
		Preload("Router").
		Preload("ServiceAccount").
		Order("last_seen_at DESC").
		Limit(limit)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&alerts).Error
	return alerts, err
}

func (r *alertRepository) UpdateAlertStatus(id uuid.UUID, status string) error {
	now := time.Now()
	updates := map[string]interface{}{"status": status}
	if status == "acknowledged" {
		updates["acknowledged_at"] = &now
	}
	if status == "resolved" {
		updates["resolved_at"] = &now
	}
	return r.db.Model(&models.AlertEvent{}).Where("id = ?", id).Updates(updates).Error
}

func (r *alertRepository) CountOpenAlerts() (int64, error) {
	var total int64
	err := r.db.Model(&models.AlertEvent{}).
		Where("status IN ?", []string{"open", "acknowledged"}).
		Count(&total).Error
	return total, err
}

func (r *alertRepository) FindRecentFailedProvisioningJobs(since time.Time) ([]models.ProvisioningJob, error) {
	var jobs []models.ProvisioningJob
	err := r.db.
		Where("status = ? AND updated_at >= ?", "failed", since).
		Order("updated_at DESC").
		Find(&jobs).Error
	return jobs, err
}

func (r *alertRepository) GetLatestTwoRouterSnapshots(routerID uuid.UUID) ([]models.RouterSnapshot, error) {
	var snapshots []models.RouterSnapshot
	err := r.db.
		Where("router_id = ?", routerID).
		Order("collected_at DESC").
		Limit(2).
		Find(&snapshots).Error
	return snapshots, err
}
