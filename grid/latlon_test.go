package grid

import (
	"math"
	"testing"
)

func TestLatLonGridCoordinates(t *testing.T) {
	// Create a simple 3x3 grid
	// 90°N to 88°N, 0°E to 2°E, 1° spacing
	// Scanning mode 0x00: +i (west to east), -j (north to south)
	grid := &LatLonGrid{
		Ni:           3,
		Nj:           3,
		La1:          90000, // 90°N
		Lo1:          0,     // 0°E
		La2:          88000, // 88°N
		Lo2:          2000,  // 2°E
		Di:           1000,  // 1° longitude
		Dj:           1000,  // 1° latitude
		ScanningMode: 0x00,  // Standard: +i, -j, consecutive
	}

	lats := grid.Latitudes()
	lons := grid.Longitudes()

	// Should have 9 points
	if len(lats) != 9 {
		t.Fatalf("expected 9 latitude values, got %d", len(lats))
	}
	if len(lons) != 9 {
		t.Fatalf("expected 9 longitude values, got %d", len(lons))
	}

	// Expected coordinates (in scan order: row by row, west to east)
	expectedLats := []float32{
		90, 90, 90, // Row 1: 90°N
		89, 89, 89, // Row 2: 89°N
		88, 88, 88, // Row 3: 88°N
	}
	expectedLons := []float32{
		0, 1, 2, // Column 1, 2, 3
		0, 1, 2,
		0, 1, 2,
	}

	for i := range lats {
		if math.Abs(float64(lats[i]-expectedLats[i])) > 0.001 {
			t.Errorf("lat[%d]: got %.3f, want %.3f", i, lats[i], expectedLats[i])
		}
		if math.Abs(float64(lons[i]-expectedLons[i])) > 0.001 {
			t.Errorf("lon[%d]: got %.3f, want %.3f", i, lons[i], expectedLons[i])
		}
	}
}

func TestLatLonGridCoordinatesReversedI(t *testing.T) {
	// Grid scanning east to west (i negative)
	// Scanning mode 0x80: -i (east to west), -j (north to south)
	grid := &LatLonGrid{
		Ni:           3,
		Nj:           2,
		La1:          10000, // 10°N
		Lo1:          2000,  // 2°E (starting from east)
		La2:          9000,  // 9°N
		Lo2:          0,     // 0°E (ending at west)
		Di:           1000,  // 1° longitude
		Dj:           1000,  // 1° latitude
		ScanningMode: 0x80,  // -i, -j, consecutive
	}

	lats := grid.Latitudes()
	lons := grid.Longitudes()

	// Expected: scanning east to west
	expectedLats := []float32{
		10, 10, 10, // Row 1
		9, 9, 9, // Row 2
	}
	expectedLons := []float32{
		2, 1, 0, // East to west
		2, 1, 0,
	}

	for i := range lats {
		if math.Abs(float64(lats[i]-expectedLats[i])) > 0.001 {
			t.Errorf("lat[%d]: got %.3f, want %.3f", i, lats[i], expectedLats[i])
		}
		if math.Abs(float64(lons[i]-expectedLons[i])) > 0.001 {
			t.Errorf("lon[%d]: got %.3f, want %.3f", i, lons[i], expectedLons[i])
		}
	}
}

func TestLatLonGridCoordinatesReversedJ(t *testing.T) {
	// Grid scanning south to north (j positive)
	// Scanning mode 0x40: +i (west to east), +j (south to north)
	grid := &LatLonGrid{
		Ni:           2,
		Nj:           3,
		La1:          -10000, // -10°N (10°S, starting south)
		Lo1:          0,      // 0°E
		La2:          -8000,  // -8°N (8°S, ending north)
		Lo2:          1000,   // 1°E
		Di:           1000,   // 1° longitude
		Dj:           1000,   // 1° latitude
		ScanningMode: 0x40,   // +i, +j, consecutive
	}

	lats := grid.Latitudes()
	lons := grid.Longitudes()

	// Expected: scanning south to north
	expectedLats := []float32{
		-10, -10, // Row 1: 10°S
		-9, -9, // Row 2: 9°S
		-8, -8, // Row 3: 8°S
	}
	expectedLons := []float32{
		0, 1,
		0, 1,
		0, 1,
	}

	for i := range lats {
		if math.Abs(float64(lats[i]-expectedLats[i])) > 0.001 {
			t.Errorf("lat[%d]: got %.3f, want %.3f", i, lats[i], expectedLats[i])
		}
		if math.Abs(float64(lons[i]-expectedLons[i])) > 0.001 {
			t.Errorf("lon[%d]: got %.3f, want %.3f", i, lons[i], expectedLons[i])
		}
	}
}

func TestLatLonGridCoordinatesDateLine(t *testing.T) {
	// Grid crossing the date line
	// 0°N, 358°E to 2°E
	grid := &LatLonGrid{
		Ni:           3,
		Nj:           2,
		La1:          0,      // 0°N
		Lo1:          358000, // 358°E
		La2:          -1000,  // 1°S
		Lo2:          0,      // 0°E
		Di:           1000,   // 1° longitude
		Dj:           1000,   // 1° latitude
		ScanningMode: 0x00,
	}

	lons := grid.Longitudes()

	// Should wrap around correctly: 358, 359, 0
	expectedLons := []float32{
		358, 359, 0,
		358, 359, 0,
	}

	for i := range lons {
		if math.Abs(float64(lons[i]-expectedLons[i])) > 0.001 {
			t.Errorf("lon[%d]: got %.3f, want %.3f", i, lons[i], expectedLons[i])
		}
	}
}

func TestLatLonGridCoordinatesNegativeLongitudes(t *testing.T) {
	// Grid with negative longitudes (western hemisphere)
	grid := &LatLonGrid{
		Ni:           3,
		Nj:           2,
		La1:          0,      // 0°N
		Lo1:          -10000, // -10°E (350°E)
		La2:          -1000,  // 1°S
		Lo2:          -8000,  // -8°E (352°E)
		Di:           1000,   // 1° longitude
		Dj:           1000,   // 1° latitude
		ScanningMode: 0x00,
	}

	lons := grid.Longitudes()

	// Should normalize negative longitudes to [0, 360)
	expectedLons := []float32{
		350, 351, 352,
		350, 351, 352,
	}

	for i := range lons {
		if math.Abs(float64(lons[i]-expectedLons[i])) > 0.001 {
			t.Errorf("lon[%d]: got %.3f, want %.3f", i, lons[i], expectedLons[i])
		}
	}
}

func TestLatLonGridCoordinatesGlobalGrid(t *testing.T) {
	// 2.5° global grid (commonly used in climate models)
	// 144 x 73 = 10512 points
	grid := &LatLonGrid{
		Ni:           144,
		Nj:           73,
		La1:          90000,  // 90°N
		Lo1:          0,      // 0°E
		La2:          -90000, // 90°S
		Lo2:          357500, // 357.5°E
		Di:           2500,   // 2.5° longitude
		Dj:           2500,   // 2.5° latitude
		ScanningMode: 0x00,
	}

	lats := grid.Latitudes()
	lons := grid.Longitudes()

	numPoints := 144 * 73
	if len(lats) != numPoints {
		t.Fatalf("expected %d latitude values, got %d", numPoints, len(lats))
	}
	if len(lons) != numPoints {
		t.Fatalf("expected %d longitude values, got %d", numPoints, len(lons))
	}

	// Check first point (north pole, prime meridian)
	if math.Abs(float64(lats[0]-90.0)) > 0.001 {
		t.Errorf("first lat: got %.3f, want 90.0", lats[0])
	}
	if math.Abs(float64(lons[0]-0.0)) > 0.001 {
		t.Errorf("first lon: got %.3f, want 0.0", lons[0])
	}

	// Check last point (south pole, 357.5°E)
	lastIdx := numPoints - 1
	if math.Abs(float64(lats[lastIdx]-(-90.0))) > 0.001 {
		t.Errorf("last lat: got %.3f, want -90.0", lats[lastIdx])
	}
	if math.Abs(float64(lons[lastIdx]-357.5)) > 0.001 {
		t.Errorf("last lon: got %.3f, want 357.5", lons[lastIdx])
	}

	// Check a middle point (row 36, col 72 -> index 36*144 + 72)
	// Should be at 0°N (90 - 36*2.5 = 0), 180°E (0 + 72*2.5 = 180)
	midIdx := 36*144 + 72
	if math.Abs(float64(lats[midIdx]-0.0)) > 0.001 {
		t.Errorf("middle lat: got %.3f, want 0.0", lats[midIdx])
	}
	if math.Abs(float64(lons[midIdx]-180.0)) > 0.001 {
		t.Errorf("middle lon: got %.3f, want 180.0", lons[midIdx])
	}
}

func TestLatLonGridCoordinatesNonConsecutive(t *testing.T) {
	// Grid with non-consecutive scanning (adjacent points in j direction)
	// Scanning mode 0x20: +i, -j, non-consecutive (j varies fastest)
	grid := &LatLonGrid{
		Ni:           2,
		Nj:           3,
		La1:          10000, // 10°N
		Lo1:          0,     // 0°E
		La2:          8000,  // 8°N
		Lo2:          1000,  // 1°E
		Di:           1000,  // 1° longitude
		Dj:           1000,  // 1° latitude
		ScanningMode: 0x20,  // +i, -j, j consecutive
	}

	lats := grid.Latitudes()
	lons := grid.Longitudes()

	// Expected: j varies fastest (column by column)
	expectedLats := []float32{
		10, 9, 8, // Column 1
		10, 9, 8, // Column 2
	}
	expectedLons := []float32{
		0, 0, 0, // Column 1: 0°E
		1, 1, 1, // Column 2: 1°E
	}

	for i := range lats {
		if math.Abs(float64(lats[i]-expectedLats[i])) > 0.001 {
			t.Errorf("lat[%d]: got %.3f, want %.3f", i, lats[i], expectedLats[i])
		}
		if math.Abs(float64(lons[i]-expectedLons[i])) > 0.001 {
			t.Errorf("lon[%d]: got %.3f, want %.3f", i, lons[i], expectedLons[i])
		}
	}
}

func TestLatLonGridScanningFlags(t *testing.T) {
	tests := []struct {
		name       string
		scanMode   uint8
		wantINeg   bool
		wantJPos   bool
		wantConsec bool
	}{
		{"Standard", 0x00, false, false, true},
		{"Reversed I", 0x80, true, false, true},
		{"Reversed J", 0x40, false, true, true},
		{"Non-consecutive", 0x20, false, false, false},
		{"All reversed", 0xC0, true, true, true},
		{"Reversed I non-consec", 0xA0, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid := &LatLonGrid{ScanningMode: tt.scanMode}
			iNeg, jPos, consec := grid.ScanningFlags()

			if iNeg != tt.wantINeg {
				t.Errorf("iNegative: got %v, want %v", iNeg, tt.wantINeg)
			}
			if jPos != tt.wantJPos {
				t.Errorf("jPositive: got %v, want %v", jPos, tt.wantJPos)
			}
			if consec != tt.wantConsec {
				t.Errorf("consecutive: got %v, want %v", consec, tt.wantConsec)
			}
		})
	}
}

func TestLatLonGridCoordinatesMethod(t *testing.T) {
	grid := &LatLonGrid{
		Ni:           2,
		Nj:           2,
		La1:          10000,
		Lo1:          0,
		La2:          9000,
		Lo2:          1000,
		Di:           1000,
		Dj:           1000,
		ScanningMode: 0x00,
	}

	lats, lons := grid.Coordinates()

	if len(lats) != 4 {
		t.Errorf("expected 4 latitudes, got %d", len(lats))
	}
	if len(lons) != 4 {
		t.Errorf("expected 4 longitudes, got %d", len(lons))
	}

	// Verify first and last points
	if math.Abs(float64(lats[0]-10.0)) > 0.001 {
		t.Errorf("first lat: got %.3f, want 10.0", lats[0])
	}
	if math.Abs(float64(lons[0]-0.0)) > 0.001 {
		t.Errorf("first lon: got %.3f, want 0.0", lons[0])
	}
}
