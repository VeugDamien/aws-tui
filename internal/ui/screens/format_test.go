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

func TestBrailleGraphDimensions(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	for _, dim := range []struct{ w, h int }{{40, 3}, {12, 4}, {80, 6}} {
		lines := BrailleGraph(vals, dim.w, dim.h, 8)
		if len(lines) != dim.h {
			t.Errorf("height=%d -> %d lines, want %d", dim.h, len(lines), dim.h)
		}
		for i, l := range lines {
			if n := len([]rune(l)); n != dim.w {
				t.Errorf("line %d width = %d, want %d", i, n, dim.w)
			}
		}
	}
}

func TestBrailleGraphEmpty(t *testing.T) {
	lines := BrailleGraph(nil, 10, 3, 100)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for _, l := range lines {
		if l != strings.Repeat(" ", 10) {
			t.Errorf("empty graph line = %q, want 10 spaces", l)
		}
	}
}

// hasDots reports whether a braille line contains any raised dot (i.e. a rune
// other than the blank braille cell U+2800). Note: the blank braille cell is not
// an ASCII space, so strings.TrimSpace does not detect it.
func hasDots(line string) bool {
	for _, r := range line {
		if r != 0x2800 {
			return true
		}
	}
	return false
}

func TestBrailleGraphScaling(t *testing.T) {
	// A flat series at the scale max should fill the whole column (top row set);
	// a flat series at zero should leave the top row blank.
	full := BrailleGraph([]float64{100, 100, 100, 100}, 8, 3, 100)
	if !hasDots(full[0]) {
		t.Error("series at scale max should light the top row")
	}
	zero := BrailleGraph([]float64{0, 0, 0, 0}, 8, 3, 100)
	if hasDots(zero[0]) {
		t.Error("series at zero should leave the top row blank")
	}
}
