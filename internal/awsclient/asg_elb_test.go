package awsclient

import "testing"

func TestAutoScalingGroupCounts(t *testing.T) {
	g := AutoScalingGroup{
		Instances: []ASGInstance{
			{InstanceID: "i-1", HealthStatus: "Healthy"},
			{InstanceID: "i-2", HealthStatus: "Healthy"},
			{InstanceID: "i-3", HealthStatus: "Unhealthy"},
			{InstanceID: "i-4", HealthStatus: "Pending"},
		},
	}
	if got := g.InstanceCount(); got != 4 {
		t.Errorf("InstanceCount() = %d, want 4", got)
	}
	if got := g.HealthyCount(); got != 2 {
		t.Errorf("HealthyCount() = %d, want 2", got)
	}
}

func TestTargetGroupCounts(t *testing.T) {
	tg := TargetGroup{
		Targets: []TargetHealth{
			{ID: "i-1", State: "healthy"},
			{ID: "i-2", State: "healthy"},
			{ID: "i-3", State: "unhealthy"},
		},
	}
	if got := tg.TotalCount(); got != 3 {
		t.Errorf("TotalCount() = %d, want 3", got)
	}
	if got := tg.HealthyCount(); got != 2 {
		t.Errorf("HealthyCount() = %d, want 2", got)
	}
}

func TestTargetGroupCountsEmpty(t *testing.T) {
	tg := TargetGroup{}
	if got := tg.TotalCount(); got != 0 {
		t.Errorf("TotalCount() = %d, want 0", got)
	}
	if got := tg.HealthyCount(); got != 0 {
		t.Errorf("HealthyCount() = %d, want 0", got)
	}
}
