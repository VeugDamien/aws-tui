package awsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
)

// ASGInstance is a flattened view of an instance belonging to an Auto Scaling group.
type ASGInstance struct {
	InstanceID     string
	AZ             string
	LifecycleState string
	HealthStatus   string
	Protected      bool
}

// AutoScalingGroup is a flattened view of an EC2 Auto Scaling group for display.
type AutoScalingGroup struct {
	Name            string
	ARN             string
	MinSize         int32
	MaxSize         int32
	DesiredCapacity int32
	HealthCheckType string
	LaunchType      string // "template", "config", "mixed" or "-"
	LaunchName      string // template or configuration name
	VPCZoneID       string
	CreatedTime     string
	Status          string // set when the group is being deleted
	AZs             []string
	LoadBalancers   []string // classic LB names
	TargetGroupARNs []string
	Instances       []ASGInstance
}

// InstanceCount returns the number of registered instances.
func (g AutoScalingGroup) InstanceCount() int { return len(g.Instances) }

// HealthyCount returns the number of instances reporting a "Healthy" status.
func (g AutoScalingGroup) HealthyCount() int {
	n := 0
	for _, i := range g.Instances {
		if i.HealthStatus == "Healthy" {
			n++
		}
	}
	return n
}

// ListAutoScalingGroups returns all Auto Scaling groups reachable with the given
// config, handling pagination transparently.
func ListAutoScalingGroups(ctx context.Context, cfg aws.Config) ([]AutoScalingGroup, error) {
	client := autoscaling.NewFromConfig(cfg)
	paginator := autoscaling.NewDescribeAutoScalingGroupsPaginator(client, &autoscaling.DescribeAutoScalingGroupsInput{})

	var groups []AutoScalingGroup
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe auto scaling groups: %w", err)
		}
		for _, g := range page.AutoScalingGroups {
			groups = append(groups, flattenASG(g))
		}
	}
	return groups, nil
}

func flattenASG(g types.AutoScalingGroup) AutoScalingGroup {
	out := AutoScalingGroup{
		Name:            aws.ToString(g.AutoScalingGroupName),
		ARN:             aws.ToString(g.AutoScalingGroupARN),
		MinSize:         aws.ToInt32(g.MinSize),
		MaxSize:         aws.ToInt32(g.MaxSize),
		DesiredCapacity: aws.ToInt32(g.DesiredCapacity),
		HealthCheckType: aws.ToString(g.HealthCheckType),
		VPCZoneID:       aws.ToString(g.VPCZoneIdentifier),
		Status:          aws.ToString(g.Status),
		AZs:             g.AvailabilityZones,
		LoadBalancers:   g.LoadBalancerNames,
		TargetGroupARNs: g.TargetGroupARNs,
	}
	if g.CreatedTime != nil {
		out.CreatedTime = g.CreatedTime.Format(time.RFC3339)
	}

	switch {
	case g.MixedInstancesPolicy != nil:
		out.LaunchType = "mixed"
		if g.MixedInstancesPolicy.LaunchTemplate != nil &&
			g.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification != nil {
			out.LaunchName = aws.ToString(g.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification.LaunchTemplateName)
		}
	case g.LaunchTemplate != nil:
		out.LaunchType = "template"
		out.LaunchName = aws.ToString(g.LaunchTemplate.LaunchTemplateName)
	case g.LaunchConfigurationName != nil:
		out.LaunchType = "config"
		out.LaunchName = aws.ToString(g.LaunchConfigurationName)
	default:
		out.LaunchType = "-"
	}

	for _, inst := range g.Instances {
		out.Instances = append(out.Instances, ASGInstance{
			InstanceID:     aws.ToString(inst.InstanceId),
			AZ:             aws.ToString(inst.AvailabilityZone),
			LifecycleState: string(inst.LifecycleState),
			HealthStatus:   aws.ToString(inst.HealthStatus),
			Protected:      aws.ToBool(inst.ProtectedFromScaleIn),
		})
	}
	return out
}

// ScalingActivity is a flattened view of a recent scaling activity.
type ScalingActivity struct {
	Description  string
	Cause        string
	StatusCode   string
	StartTime    string
	Progress     int32
	StatusReason string
}

// DescribeScalingActivities returns the most recent scaling activities for a group
// (newest first), capped at maxItems.
func DescribeScalingActivities(ctx context.Context, cfg aws.Config, groupName string, maxItems int32) ([]ScalingActivity, error) {
	client := autoscaling.NewFromConfig(cfg)
	out, err := client.DescribeScalingActivities(ctx, &autoscaling.DescribeScalingActivitiesInput{
		AutoScalingGroupName: aws.String(groupName),
		MaxRecords:           aws.Int32(maxItems),
	})
	if err != nil {
		return nil, fmt.Errorf("describe scaling activities: %w", err)
	}

	var activities []ScalingActivity
	for _, a := range out.Activities {
		act := ScalingActivity{
			Description:  aws.ToString(a.Description),
			Cause:        aws.ToString(a.Cause),
			StatusCode:   string(a.StatusCode),
			Progress:     aws.ToInt32(a.Progress),
			StatusReason: aws.ToString(a.StatusMessage),
		}
		if a.StartTime != nil {
			act.StartTime = a.StartTime.Format(time.RFC3339)
		}
		activities = append(activities, act)
	}
	return activities, nil
}
