package data

import "testing"

func TestIntPower(t *testing.T) {
	// A variable, so that the products below are rounded float64
	// multiplications rather than exact constant arithmetic.
	tenth := 0.1
	for _, tc := range []struct {
		x    float64
		y    int
		want float64
	}{
		{2, 0, 1},
		{2, 3, 8},
		{2, -3, 0.125},
		{10, 3, 1000},
		// Like wgrib2, negative powers of ten are products of 0.1, not
		// the closest doubles to the exact values.
		{10, -1, tenth},
		{10, -2, tenth * tenth},
		{10, -3, tenth * (tenth * tenth)},
	} {
		if got := intPower(tc.x, tc.y); got != tc.want {
			t.Errorf("intPower(%g, %d) = %.20g, want %.20g", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestUnpacker(t *testing.T) {
	// The values are computed in float64 and rounded once.
	tenth := 0.1
	u := newUnpacker(10123844, 2, 2)
	x := 12345.0
	if got, want := u.value(x), float32((x*4+10123844)*(tenth*tenth)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// R + X*2^E is not representable in float32 here; rounding it before
	// scaling would give 16777218/10.
	u = newUnpacker(16777216, -1, 1)
	if got, want := u.value(3), float32(16777217.5*tenth); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
