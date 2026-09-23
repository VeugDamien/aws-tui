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
