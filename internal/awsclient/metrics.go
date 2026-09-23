package awsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// MetricSeries is a named series of datapoints (ordered by time).
type MetricSeries struct {
	Label  string
	Unit   string
	Values []float64
	Latest float64
	Avg    float64
	Max    float64
}

// InstanceMetrics bundles the metrics displayed on the detail page.
type InstanceMetrics struct {
	CPU        MetricSeries // percent
	NetworkIn  MetricSeries // bytes
	NetworkOut MetricSeries // bytes

	WindowMinutes int // total time window covered (e.g. 60)
	PeriodMinutes int // granularity per datapoint (e.g. 5)
}

// periodForWindow picks a CloudWatch period (seconds) giving a readable number of
// datapoints (~12-48) for the requested window, snapped to a valid CW period.
func periodForWindow(windowMin int) int32 {
	switch {
	case windowMin <= 60:
		return 300 // 5 min  -> 12 pts
	case windowMin <= 180:
		return 300 // 5 min  -> 36 pts
	case windowMin <= 720:
		return 900 // 15 min -> up to 48 pts
	default:
		return 3600 // 1 h    -> 24 pts for 24h
	}
}

// GetInstanceMetrics fetches CPU and network metrics for an instance over the last
// windowMinutes, choosing a sensible granularity. Requires cloudwatch:GetMetricData.
func GetInstanceMetrics(ctx context.Context, cfg aws.Config, instanceID string, windowMinutes int) (InstanceMetrics, error) {
	if windowMinutes <= 0 {
		windowMinutes = 60
	}
	client := cloudwatch.NewFromConfig(cfg)

	end := time.Now()
	start := end.Add(-time.Duration(windowMinutes) * time.Minute)
	period := periodForWindow(windowMinutes)

	dim := []types.Dimension{{
		Name:  aws.String("InstanceId"),
		Value: aws.String(instanceID),
	}}

	metric := func(id, name string) types.MetricDataQuery {
		return types.MetricDataQuery{
			Id: aws.String(id),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String("AWS/EC2"),
					MetricName: aws.String(name),
					Dimensions: dim,
				},
				Period: aws.Int32(period),
				Stat:   aws.String("Average"),
			},
			ReturnData: aws.Bool(true),
		}
	}

	out, err := client.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(start),
		EndTime:   aws.Time(end),
		ScanBy:    types.ScanByTimestampAscending,
		MetricDataQueries: []types.MetricDataQuery{
			metric("cpu", "CPUUtilization"),
			metric("netin", "NetworkIn"),
			metric("netout", "NetworkOut"),
		},
	})
	if err != nil {
		return InstanceMetrics{}, fmt.Errorf("get metric data: %w", err)
	}

	var m InstanceMetrics
	m.WindowMinutes = windowMinutes
	m.PeriodMinutes = int(period / 60)
	for _, res := range out.MetricDataResults {
		s := MetricSeries{Values: res.Values}
		s.computeStats()
		switch aws.ToString(res.Id) {
		case "cpu":
			s.Label, s.Unit = "CPU", "%"
			m.CPU = s
		case "netin":
			s.Label, s.Unit = "Network In", "B"
			m.NetworkIn = s
		case "netout":
			s.Label, s.Unit = "Network Out", "B"
			m.NetworkOut = s
		}
	}
	return m, nil
}

func (s *MetricSeries) computeStats() {
	if len(s.Values) == 0 {
		return
	}
	var sum float64
	s.Max = s.Values[0]
	for _, v := range s.Values {
		sum += v
		if v > s.Max {
			s.Max = v
		}
	}
	s.Avg = sum / float64(len(s.Values))
	s.Latest = s.Values[len(s.Values)-1]
}
