package awsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// LoadBalancer is a flattened view of an ELBv2 load balancer (ALB/NLB/GWLB).
type LoadBalancer struct {
	Name        string
	ARN         string
	DNSName     string
	Type        string // "application", "network", "gateway"
	Scheme      string // "internet-facing", "internal"
	State       string // "active", "provisioning", "failed"…
	VPCID       string
	CreatedTime string
	AZs         []string
}

// ListLoadBalancers returns all ELBv2 load balancers reachable with the given
// config, handling pagination transparently.
func ListLoadBalancers(ctx context.Context, cfg aws.Config) ([]LoadBalancer, error) {
	client := elbv2.NewFromConfig(cfg)
	paginator := elbv2.NewDescribeLoadBalancersPaginator(client, &elbv2.DescribeLoadBalancersInput{})

	var lbs []LoadBalancer
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe load balancers: %w", err)
		}
		for _, lb := range page.LoadBalancers {
			lbs = append(lbs, flattenLB(lb))
		}
	}
	return lbs, nil
}

func flattenLB(lb types.LoadBalancer) LoadBalancer {
	out := LoadBalancer{
		Name:    aws.ToString(lb.LoadBalancerName),
		ARN:     aws.ToString(lb.LoadBalancerArn),
		DNSName: aws.ToString(lb.DNSName),
		Type:    string(lb.Type),
		Scheme:  string(lb.Scheme),
		VPCID:   aws.ToString(lb.VpcId),
	}
	if lb.State != nil {
		out.State = string(lb.State.Code)
	}
	if lb.CreatedTime != nil {
		out.CreatedTime = lb.CreatedTime.Format(time.RFC3339)
	}
	for _, az := range lb.AvailabilityZones {
		out.AZs = append(out.AZs, aws.ToString(az.ZoneName))
	}
	return out
}

// Listener is a flattened view of a load balancer listener.
type Listener struct {
	ARN      string
	Protocol string
	Port     int32
}

// DescribeListeners returns the listeners of a load balancer.
func DescribeListeners(ctx context.Context, cfg aws.Config, lbARN string) ([]Listener, error) {
	client := elbv2.NewFromConfig(cfg)
	paginator := elbv2.NewDescribeListenersPaginator(client, &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(lbARN),
	})

	var listeners []Listener
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe listeners: %w", err)
		}
		for _, l := range page.Listeners {
			listeners = append(listeners, Listener{
				ARN:      aws.ToString(l.ListenerArn),
				Protocol: string(l.Protocol),
				Port:     aws.ToInt32(l.Port),
			})
		}
	}
	return listeners, nil
}

// TargetHealth is a flattened view of a single target's health.
type TargetHealth struct {
	ID     string
	Port   int32
	AZ     string
	State  string // "healthy", "unhealthy", "initial", "draining"…
	Reason string
}

// TargetGroup is a flattened view of a target group plus the health of its targets.
type TargetGroup struct {
	Name       string
	ARN        string
	Protocol   string
	Port       int32
	TargetType string // "instance", "ip", "lambda", "alb"
	Targets    []TargetHealth
}

// HealthyCount / TotalCount summarise target health for compact display.
func (g TargetGroup) HealthyCount() int {
	n := 0
	for _, t := range g.Targets {
		if t.State == "healthy" {
			n++
		}
	}
	return n
}

func (g TargetGroup) TotalCount() int { return len(g.Targets) }

// DescribeTargetGroupsForLB returns the target groups associated with a load
// balancer, each enriched with the current health of its targets.
func DescribeTargetGroupsForLB(ctx context.Context, cfg aws.Config, lbARN string) ([]TargetGroup, error) {
	client := elbv2.NewFromConfig(cfg)

	var groups []TargetGroup
	paginator := elbv2.NewDescribeTargetGroupsPaginator(client, &elbv2.DescribeTargetGroupsInput{
		LoadBalancerArn: aws.String(lbARN),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe target groups: %w", err)
		}
		for _, tg := range page.TargetGroups {
			g := TargetGroup{
				Name:       aws.ToString(tg.TargetGroupName),
				ARN:        aws.ToString(tg.TargetGroupArn),
				Protocol:   string(tg.Protocol),
				Port:       aws.ToInt32(tg.Port),
				TargetType: string(tg.TargetType),
			}
			health, err := targetHealth(ctx, client, g.ARN)
			if err != nil {
				return nil, err
			}
			g.Targets = health
			groups = append(groups, g)
		}
	}
	return groups, nil
}

// DescribeTargetGroupsByARNs returns target groups for the given ARNs (used by the
// ASG detail page), each enriched with target health.
func DescribeTargetGroupsByARNs(ctx context.Context, cfg aws.Config, arns []string) ([]TargetGroup, error) {
	if len(arns) == 0 {
		return nil, nil
	}
	client := elbv2.NewFromConfig(cfg)
	out, err := client.DescribeTargetGroups(ctx, &elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: arns,
	})
	if err != nil {
		return nil, fmt.Errorf("describe target groups: %w", err)
	}

	var groups []TargetGroup
	for _, tg := range out.TargetGroups {
		g := TargetGroup{
			Name:       aws.ToString(tg.TargetGroupName),
			ARN:        aws.ToString(tg.TargetGroupArn),
			Protocol:   string(tg.Protocol),
			Port:       aws.ToInt32(tg.Port),
			TargetType: string(tg.TargetType),
		}
		health, err := targetHealth(ctx, client, g.ARN)
		if err != nil {
			return nil, err
		}
		g.Targets = health
		groups = append(groups, g)
	}
	return groups, nil
}

func targetHealth(ctx context.Context, client *elbv2.Client, tgARN string) ([]TargetHealth, error) {
	out, err := client.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(tgARN),
	})
	if err != nil {
		return nil, fmt.Errorf("describe target health: %w", err)
	}

	var targets []TargetHealth
	for _, d := range out.TargetHealthDescriptions {
		t := TargetHealth{}
		if d.Target != nil {
			t.ID = aws.ToString(d.Target.Id)
			t.Port = aws.ToInt32(d.Target.Port)
			t.AZ = aws.ToString(d.Target.AvailabilityZone)
		}
		if d.TargetHealth != nil {
			t.State = string(d.TargetHealth.State)
			t.Reason = string(d.TargetHealth.Reason)
		}
		targets = append(targets, t)
	}
	return targets, nil
}
