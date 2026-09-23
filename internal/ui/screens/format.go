package screens

import (
	"fmt"
	"math"
	"strings"
)

// sparkChars are the eight block levels used for sparklines.
var sparkChars = []rune("▁▂▃▄▅▆▇█")

// SparklineFixed renders a sparkline of exactly width characters, upsampling
// (nearest-neighbour) when there are fewer points than width and downsampling when
// there are more. This keeps the graph aligned with a fixed-width time axis.
func SparklineFixed(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return strings.Repeat(" ", maxInt(width, 0))
	}
	if len(values) == width {
		return renderSpark(values)
	}
	if len(values) > width {
		return renderSpark(downsample(values, width))
	}
	// Upsample: map each output column to the nearest input point.
	pts := make([]float64, width)
	for i := 0; i < width; i++ {
		src := i * len(values) / width
		if src >= len(values) {
			src = len(values) - 1
		}
		pts[i] = values[src]
	}
	return renderSpark(pts)
}

// renderSpark maps values to block characters scaled between their min and max.
func renderSpark(pts []float64) string {
	min, max := pts[0], pts[0]
	for _, v := range pts {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	out := make([]rune, len(pts))
	for i, v := range pts {
		var level int
		if span > 0 {
			level = int(math.Round((v - min) / span * float64(len(sparkChars)-1)))
		}
		if level < 0 {
			level = 0
		}
		if level >= len(sparkChars) {
			level = len(sparkChars) - 1
		}
		out[i] = sparkChars[level]
	}
	return string(out)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func downsample(values []float64, n int) []float64 {
	out := make([]float64, n)
	bucket := float64(len(values)) / float64(n)
	for i := 0; i < n; i++ {
		start := int(float64(i) * bucket)
		end := int(float64(i+1) * bucket)
		if end > len(values) {
			end = len(values)
		}
		if start >= end {
			start = end - 1
			if start < 0 {
				start = 0
			}
		}
		var sum float64
		for _, v := range values[start:end] {
			sum += v
		}
		out[i] = sum / float64(end-start)
	}
	return out
}

// humanBytes formats a byte count with a binary unit suffix.
func humanBytes(v float64) string {
	const unit = 1024.0
	if v < unit {
		return fmt.Sprintf("%.0f B", v)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	exp := 0
	val := v
	for val >= unit && exp < len(units) {
		val /= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", val, units[exp-1])
}
