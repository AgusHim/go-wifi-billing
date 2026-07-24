package observability

import (
	"testing"

	"github.com/google/uuid"
)

func TestAccessControlObserverAggregatesMetricsAndAlerts(t *testing.T) {
	t.Setenv("AUTHZ_FORBIDDEN_ALERT_THRESHOLD", "2")
	t.Setenv("AUTHZ_FORBIDDEN_ALERT_WINDOW", "5m")
	observer := NewAccessControlObserver()
	observer.RecordAuthorization("get", "/admin_api/customers/:id", "customers.read", 401)
	observer.RecordAuthorization("get", "/admin_api/customers/:id", "customers.read", 403)
	observer.RecordAuthorization("get", "/admin_api/customers/:id", "customers.read", 403)
	observer.RecordAuthorization("get", "/admin_api/customers/:id", "customers.read", 200)

	actorID := uuid.New()
	targetID := uuid.New()
	observer.RecordOwnerChange(actorID, targetID, "user_role_changed")
	snapshot := observer.Snapshot()
	if len(snapshot.Metrics) != 2 {
		t.Fatalf("metrics = %d, want 2 status series", len(snapshot.Metrics))
	}
	if len(snapshot.Alerts) != 2 || snapshot.Alerts[0].Type != "forbidden_spike" || snapshot.Alerts[1].Type != "owner_change" {
		t.Fatalf("unexpected alerts: %#v", snapshot.Alerts)
	}
	if snapshot.Alerts[1].ActorID != actorID || snapshot.Alerts[1].TargetID != targetID {
		t.Fatalf("owner alert identifiers were not retained: %#v", snapshot.Alerts[1])
	}
}
