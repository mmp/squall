package product

import "testing"

func TestSurfaceValue(t *testing.T) {
	for _, tc := range []struct {
		scaleFactor uint8
		scaledValue uint32
		want        float64
	}{
		{0, 50000, 50000},
		{0, 101320, 101320},
		{2, 101325, 1013.25},
		{3, 15, 0.015},
		{0x81, 5, 50},         // scale factor -1
		{0x84, 1, 10000},      // scale factor -4
		{0, 0x80000005, -5},   // negative value
		{1, 0x8000000F, -1.5}, // negative value with a scale factor
		{0xFF, 0xFFFFFFFF, 0}, // missing
		{0xFF, 0, 0},          // missing scale factor
		{0, 0xFFFFFFFF, 0},    // missing value
	} {
		if got := SurfaceValue(tc.scaleFactor, tc.scaledValue); got != tc.want {
			t.Errorf("SurfaceValue(%#x, %#x) = %v, want %v", tc.scaleFactor, tc.scaledValue, got, tc.want)
		}
	}
}
