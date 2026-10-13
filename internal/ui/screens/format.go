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

// brailleDotBits maps a (col,row) position inside a 2x4 braille cell to the bit to
// set in the base rune U+2800. Columns are 0..1 (left/right), rows 0..3 (top to
// bottom). This matches the Unicode braille dot numbering.
var brailleDotBits = [2][4]rune{
	{0x01, 0x02, 0x04, 0x40}, // left column:  dots 1,2,3,7
	{0x08, 0x10, 0x20, 0x80}, // right column: dots 4,5,6,8
}

// BrailleGraph renders values as a multi-line braille plot of the given character
// width and height. Each cell packs a 2x4 dot grid, so the effective resolution is
// (width*2) horizontal samples by (height*4) vertical levels — far finer than a
// one-line sparkline. Values are scaled against a zero-based axis using scaleMax
// (pass the series max, or 100 for percentages) so magnitudes stay comparable.
// It returns exactly height lines, top (highest value) to bottom.
func BrailleGraph(values []float64, width, height int, scaleMax float64) []string {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	lines := make([]string, height)
	if len(values) == 0 {
		blank := strings.Repeat(" ", width)
		for i := range lines {
			lines[i] = blank
		}
		return lines
	}

	cols := width * 2  // horizontal dot columns
	rows := height * 4 // vertical dot rows
	pts := resample(values, cols)

	if scaleMax <= 0 {
		// Fall back to the data max so a flat-zero series still renders cleanly.
		for _, v := range pts {
			if v > scaleMax {
				scaleMax = v
			}
		}
		if scaleMax <= 0 {
			scaleMax = 1
		}
	}

	// grid[cellRow][cellCol] accumulates dot bits.
	grid := make([][]rune, height)
	for r := range grid {
		grid[r] = make([]rune, width)
	}

	// lineThickness is the number of dot-rows lit at each value (line mode). A
	// value of 2 keeps the trace visible without filling the area underneath.
	const lineThickness = 2

	for x, v := range pts {
		// Height in dot-rows from the bottom, clamped to [0, rows].
		frac := v / scaleMax
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		// Dot-row of the value, counted from the bottom (0-based). A zero value
		// sits on the bottom row; the scale max on the top row.
		level := int(math.Round(frac * float64(rows-1)))
		cellCol := x / 2
		dotCol := x % 2
		// Line mode: light only the dots at (and just below) the value level, so a
		// flat high series reads as a thin line near the top rather than a solid
		// block filling the whole area.
		for t := 0; t < lineThickness; t++ {
			dr := level - t
			if dr < 0 {
				break
			}
			rowFromTop := rows - 1 - dr
			cellRow := rowFromTop / 4
			dotRow := rowFromTop % 4
			grid[cellRow][cellCol] |= brailleDotBits[dotCol][dotRow]
		}
	}

	for r := 0; r < height; r++ {
		var sb strings.Builder
		for c := 0; c < width; c++ {
			sb.WriteRune(0x2800 + grid[r][c])
		}
		lines[r] = sb.String()
	}
	return lines
}

// resample maps values onto exactly n points (averaging when downsampling,
// nearest-neighbour when upsampling), reusing the existing helpers.
func resample(values []float64, n int) []float64 {
	switch {
	case n <= 0:
		return nil
	case len(values) == n:
		return values
	case len(values) > n:
		return downsample(values, n)
	default:
		pts := make([]float64, n)
		for i := 0; i < n; i++ {
			src := i * len(values) / n
			if src >= len(values) {
				src = len(values) - 1
			}
			pts[i] = values[src]
		}
		return pts
	}
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
