package awsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Tag is a key/value instance tag.
type Tag struct {
	Key   string
	Value string
}

// Instance is a flattened view of an EC2 instance for display.
type Instance struct {
	Name       string
	InstanceID string
	State      string
	Type       string
	PrivateIP  string
	AZ         string
	Platform   string

	// Enriched fields (all derived from DescribeInstances, no extra API call).
	PublicIP         string
	VPCID            string
	SubnetID         string
	ImageID          string
	KeyName          string
	LaunchTime       string
	IAMProfile       string
	Monitoring       string
	Architecture     string
	RootDeviceName   string
	RootDeviceType   string
	SecurityGroupIDs []string
	SecurityGroups   []string // group names
	Tags             []Tag
}

// ListInstances returns all EC2 instances reachable with the given config,
// handling pagination transparently.
func ListInstances(ctx context.Context, cfg aws.Config) ([]Instance, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	var instances []Instance
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe instances: %w", err)
		}
		for _, res := range page.Reservations {
			for _, inst := range res.Instances {
				instances = append(instances, flatten(inst))
			}
		}
	}
	return instances, nil
}

func flatten(inst types.Instance) Instance {
	out := Instance{
		InstanceID:     aws.ToString(inst.InstanceId),
		Type:           string(inst.InstanceType),
		PrivateIP:      aws.ToString(inst.PrivateIpAddress),
		Platform:       aws.ToString(inst.PlatformDetails),
		PublicIP:       aws.ToString(inst.PublicIpAddress),
		VPCID:          aws.ToString(inst.VpcId),
		SubnetID:       aws.ToString(inst.SubnetId),
		ImageID:        aws.ToString(inst.ImageId),
		KeyName:        aws.ToString(inst.KeyName),
		Architecture:   string(inst.Architecture),
		RootDeviceName: aws.ToString(inst.RootDeviceName),
		RootDeviceType: string(inst.RootDeviceType),
	}
	if inst.State != nil {
		out.State = string(inst.State.Name)
	}
	if inst.Placement != nil {
		out.AZ = aws.ToString(inst.Placement.AvailabilityZone)
	}
	if inst.LaunchTime != nil {
		out.LaunchTime = inst.LaunchTime.Format(time.RFC3339)
	}
	if inst.IamInstanceProfile != nil {
		out.IAMProfile = aws.ToString(inst.IamInstanceProfile.Arn)
	}
	if inst.Monitoring != nil {
		out.Monitoring = string(inst.Monitoring.State)
	}
	for _, sg := range inst.SecurityGroups {
		out.SecurityGroupIDs = append(out.SecurityGroupIDs, aws.ToString(sg.GroupId))
		out.SecurityGroups = append(out.SecurityGroups, aws.ToString(sg.GroupName))
	}
	for _, tag := range inst.Tags {
		k, v := aws.ToString(tag.Key), aws.ToString(tag.Value)
		out.Tags = append(out.Tags, Tag{Key: k, Value: v})
		if k == "Name" {
			out.Name = v
		}
	}
	return out
}
