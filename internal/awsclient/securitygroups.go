package awsclient

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// SGRule is a single, flattened security-group rule (one CIDR/source per line).
type SGRule struct {
	Protocol string // tcp, udp, icmp, all
	Ports    string // "22", "80-443", "all"
	Source   string // CIDR, or referenced group id, or prefix list
}

// SecurityGroup holds a group's identity and its inbound/outbound rules.
type SecurityGroup struct {
	ID       string
	Name     string
	Inbound  []SGRule
	Outbound []SGRule
}

// DescribeSecurityGroups fetches the given security groups and flattens their
// rules. Requires ec2:DescribeSecurityGroups.
func DescribeSecurityGroups(ctx context.Context, cfg aws.Config, ids []string) ([]SecurityGroup, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	client := ec2.NewFromConfig(cfg)
	out, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("describe security groups: %w", err)
	}

	groups := make([]SecurityGroup, 0, len(out.SecurityGroups))
	for _, sg := range out.SecurityGroups {
		g := SecurityGroup{
			ID:       aws.ToString(sg.GroupId),
			Name:     aws.ToString(sg.GroupName),
			Inbound:  flattenPerms(sg.IpPermissions),
			Outbound: flattenPerms(sg.IpPermissionsEgress),
		}
		groups = append(groups, g)
	}
	// Preserve the order of the requested ids for stable display.
	order := map[string]int{}
	for i, id := range ids {
		order[id] = i
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return order[groups[i].ID] < order[groups[j].ID]
	})
	return groups, nil
}

func flattenPerms(perms []types.IpPermission) []SGRule {
	var rules []SGRule
	for _, p := range perms {
		proto := protoName(aws.ToString(p.IpProtocol))
		ports := portRange(p.FromPort, p.ToPort, proto)

		for _, r := range p.IpRanges {
			src := aws.ToString(r.CidrIp)
			if d := aws.ToString(r.Description); d != "" {
				src += " (" + d + ")"
			}
			rules = append(rules, SGRule{Protocol: proto, Ports: ports, Source: src})
		}
		for _, r := range p.Ipv6Ranges {
			rules = append(rules, SGRule{Protocol: proto, Ports: ports, Source: aws.ToString(r.CidrIpv6)})
		}
		for _, g := range p.UserIdGroupPairs {
			src := aws.ToString(g.GroupId)
			if d := aws.ToString(g.Description); d != "" {
				src += " (" + d + ")"
			}
			rules = append(rules, SGRule{Protocol: proto, Ports: ports, Source: src})
		}
		for _, pl := range p.PrefixListIds {
			rules = append(rules, SGRule{Protocol: proto, Ports: ports, Source: aws.ToString(pl.PrefixListId)})
		}
	}
	return rules
}

// protoName maps an IP protocol number/string to a friendly name.
func protoName(p string) string {
	switch p {
	case "-1", "":
		return "all"
	case "6":
		return "tcp"
	case "17":
		return "udp"
	case "1":
		return "icmp"
	default:
		return p
	}
}

// portRange renders the port range for display.
func portRange(from, to *int32, proto string) string {
	if proto == "all" {
		return "all"
	}
	if from == nil && to == nil {
		return "all"
	}
	f, t := aws.ToInt32(from), aws.ToInt32(to)
	if f == -1 || (f == 0 && t == 0 && proto == "icmp") {
		return "all"
	}
	if f == t {
		return fmt.Sprintf("%d", f)
	}
	return fmt.Sprintf("%d-%d", f, t)
}

// JoinSGNames returns a compact "name (id)" listing.
func JoinSGNames(ids, names []string) string {
	var parts []string
	for i := range ids {
		if i < len(names) && names[i] != "" {
			parts = append(parts, names[i]+" ("+ids[i]+")")
		} else {
			parts = append(parts, ids[i])
		}
	}
	return strings.Join(parts, ", ")
}
