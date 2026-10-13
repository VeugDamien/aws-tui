package screens

import (
	"strings"
	"testing"
)

func TestSparklineFixedWidth(t *testing.T) {
	vals := []float64{1, 2, 3, 4} // fewer points than width -> upsample
	for _, w := range []int{4, 12, 48} {
		got := []rune(SparklineFixed(vals, w))
		if len(got) != w {
			t.Errorf("SparklineFixed width=%d -> %d runes, want %d", w, len(got), w)
		}
	}
	// More points than width -> downsample.
	many := make([]float64, 100)
	for i := range many {
		many[i] = float64(i)
	}
	if got := []rune(SparklineFixed(many, 20)); len(got) != 20 {
		t.Errorf("downsample width = %d, want 20", len(got))
	}
	// Empty -> blanks of width.
	if got := SparklineFixed(nil, 5); got != "     " {
		t.Errorf("empty SparklineFixed = %q, want 5 spaces", got)
	}
}

func TestTimeAxis(t *testing.T) {
	got := timeAxis(60, 48)
	if !strings.HasPrefix(got, "-1h") {
		t.Errorf("axis should start with -1h, got %q", got)
	}
	if !strings.HasSuffix(got, "now") {
		t.Errorf("axis should end with now, got %q", got)
	}
	if l := len([]rune(got)); l != 48 {
		t.Errorf("axis width = %d, want 48", l)
	}

	// Minutes (non-hour) window.
	if a := timeAxis(30, 40); !strings.HasPrefix(a, "-30m") {
		t.Errorf("axis should start with -30m, got %q", a)
	}
}

func TestSparklineScaledWidth(t *testing.T) {
	vals := []float64{1, 2, 3, 4}
	for _, w := range []int{4, 12, 48} {
		got := []rune(SparklineScaled(vals, w, 4))
		if len(got) != w {
			t.Errorf("width=%d -> %d runes, want %d", w, len(got), w)
		}
	}
	// Empty -> blanks of width.
	if got := SparklineScaled(nil, 5, 100); got != "     " {
		t.Errorf("empty SparklineScaled = %q, want 5 spaces", got)
	}
}

func TestSparklineScaledZeroBaseline(t *testing.T) {
	// A series at the scale max renders the top block everywhere.
	full := SparklineScaled([]float64{10, 10, 10}, 3, 10)
	for _, r := range full {
		if r != '█' {
			t.Errorf("series at scale max should be full blocks, got %q", full)
			break
		}
	}
	// A tiny value against a large scale renders the lowest block (not blank),
	// so even low-but-present series stay visible.
	low := SparklineScaled([]float64{1, 1, 1}, 3, 100)
	for _, r := range low {
		if r == '█' {
			t.Errorf("tiny value against large scale should not be full, got %q", low)
			break
		}
	}
}
