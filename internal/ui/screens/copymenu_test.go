package screens

import (
	"testing"

	"github.com/VeugDamien/aws-tui/internal/awsclient"
)

func TestASGDetailCopyMenu(t *testing.T) {
	g := awsclient.AutoScalingGroup{
		Name:       "web-asg",
		ARN:        "arn:aws:autoscaling:eu-west-3:123:autoScalingGroup:x:web-asg",
		LaunchName: "web-lt",
		VPCZoneID:  "subnet-1,subnet-2",
		AZs:        []string{"eu-west-3a", "eu-west-3b"},
	}
	d := NewASGDetail(g, 80, 24)

	if d.CopyMenuOpen() {
		t.Fatal("copy menu should start closed")
	}
	d.OpenCopyMenu()
	if !d.CopyMenuOpen() {
		t.Fatal("copy menu should be open after OpenCopyMenu")
	}

	// All 5 candidates have values, so all should be present.
	if got := len(d.copyItems); got != 5 {
		t.Fatalf("expected 5 copy items, got %d", got)
	}

	// First selected field is the name.
	f, ok := d.SelectedCopyField()
	if !ok || f.Label != "Name" || f.Value != "web-asg" {
		t.Fatalf("unexpected first field: %+v (ok=%v)", f, ok)
	}

	// Cursor cannot go above the first item.
	d.CopyMenuUp()
	if d.copyCursor != 0 {
		t.Fatalf("cursor should stay at 0, got %d", d.copyCursor)
	}

	d.CopyMenuDown()
	f, _ = d.SelectedCopyField()
	if f.Label != "ARN" {
		t.Fatalf("expected ARN after one down, got %q", f.Label)
	}

	d.CloseCopyMenu()
	if d.CopyMenuOpen() {
		t.Fatal("copy menu should be closed")
	}
}

func TestASGDetailCopyMenuSkipsEmpty(t *testing.T) {
	// Only Name is set; the other candidates are empty and must be filtered out.
	g := awsclient.AutoScalingGroup{Name: "solo"}
	d := NewASGDetail(g, 80, 24)
	d.OpenCopyMenu()
	if got := len(d.copyItems); got != 1 {
		t.Fatalf("expected 1 copy item, got %d", got)
	}
	f, ok := d.SelectedCopyField()
	if !ok || f.Value != "solo" {
		t.Fatalf("unexpected field: %+v", f)
	}
}

func TestELBDetailCopyMenu(t *testing.T) {
	lb := awsclient.LoadBalancer{
		Name:    "web-alb",
		DNSName: "web-alb-123.eu-west-3.elb.amazonaws.com",
		ARN:     "arn:aws:elasticloadbalancing:eu-west-3:123:loadbalancer/app/web-alb/abc",
		VPCID:   "vpc-1",
		AZs:     []string{"eu-west-3a"},
	}
	d := NewELBDetail(lb, 80, 24)

	d.OpenCopyMenu()
	if got := len(d.copyItems); got != 5 {
		t.Fatalf("expected 5 copy items, got %d", got)
	}
	f, ok := d.SelectedCopyField()
	if !ok || f.Label != "Name" || f.Value != "web-alb" {
		t.Fatalf("unexpected first field: %+v (ok=%v)", f, ok)
	}

	// Navigate to the DNS entry.
	d.CopyMenuDown()
	f, _ = d.SelectedCopyField()
	if f.Label != "DNS" || f.Value != lb.DNSName {
		t.Fatalf("expected DNS field, got %+v", f)
	}
}

func TestELBDetailCopyMenuSkipsEmpty(t *testing.T) {
	lb := awsclient.LoadBalancer{Name: "only-name"}
	d := NewELBDetail(lb, 80, 24)
	d.OpenCopyMenu()
	if got := len(d.copyItems); got != 1 {
		t.Fatalf("expected 1 copy item, got %d", got)
	}
}
