package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterMetricAggregate struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID          uuid.UUID      `json:"router_id" gorm:"type:uuid;not null;uniqueIndex:idx_router_metric_aggregate"`
	Granularity       string         `json:"granularity" gorm:"not null;uniqueIndex:idx_router_metric_aggregate"`
	BucketAt          time.Time      `json:"bucket_at" gorm:"not null;uniqueIndex:idx_router_metric_aggregate;index"`
	AvgCPULoad        float64        `json:"avg_cpu_load" gorm:"column:avg_cpu_load"`
	MaxCPULoad        int            `json:"max_cpu_load" gorm:"column:max_cpu_load"`
	AvgFreeMemory     float64        `json:"avg_free_memory"`
	AvgActivePPPoE    float64        `json:"avg_active_pppoe" gorm:"column:avg_active_pppoe"`
	AvgActiveHotspot  float64        `json:"avg_active_hotspot"`
	MaxActiveSessions int            `json:"max_active_sessions"`
	SampleCount       int            `json:"sample_count"`
	FirstSampleAt     time.Time      `json:"first_sample_at"`
	LastSampleAt      time.Time      `json:"last_sample_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router *Router `json:"router,omitempty" gorm:"foreignKey:RouterID"`
}

func (r *RouterMetricAggregate) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
