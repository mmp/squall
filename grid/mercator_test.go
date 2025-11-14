package grid

import (
	"math"
	"testing"
)

func TestParseMercatorGrid(t *testing.T) {
	// Create a minimal valid Template 3.10 data (58 bytes)
	data := make([]byte, 58)

	// Skip shape of earth parameters (bytes 0-15)

	// Ni = 100 (bytes 16-19)
	data[16] = 0x00
	data[17] = 0x00
	data[18] = 0x00
	data[19] = 0x64 // 100

	// Nj = 50 (bytes 20-23)
	data[20] = 0x00
	data[21] = 0x00
	data[22] = 0x00
	data[23] = 0x32 // 50

	// La1 = 20000000 micro-degrees = 20° (bytes 24-27)
	data[24] = 0x01
	data[25] = 0x31
	data[26] = 0x2D
	data[27] = 0x00

	// Lo1 = 250000000 micro-degrees = 250° (bytes 28-31)
	data[28] = 0x0E
	data[29] = 0xE6
	data[30] = 0xB2
	data[31] = 0x80

	// ResFlags = 0x00 (byte 32)
	data[32] = 0x00

	// LaD = 20000000 micro-degrees = 20° (bytes 33-36)
	data[33] = 0x01
	data[34] = 0x31
	data[35] = 0x2D
	data[36] = 0x00

	// La2 = 40000000 micro-degrees = 40° (bytes 37-40)
	data[37] = 0x02
	data[38] = 0x62
	data[39] = 0x5A
	data[40] = 0x00

	// Lo2 = 280000000 micro-degrees = 280° (bytes 41-44)
	data[41] = 0x10
	data[42] = 0xB0
	data[43] = 0x76
	data[44] = 0x00

	// ScanningMode = 0x00 (byte 45)
	data[45] = 0x00

	// Orientation = 0 (bytes 46-49)
	// Di = 10000 millimeters (bytes 50-53)
	data[50] = 0x00
	data[51] = 0x00
	data[52] = 0x27
	data[53] = 0x10

	// Dj = 10000 millimeters (bytes 54-57)
	data[54] = 0x00
	data[55] = 0x00
	data[56] = 0x27
	data[57] = 0x10

	grid, err := ParseMercatorGrid(data)
	if err != nil {
		t.Fatalf("ParseMercatorGrid failed: %v", err)
	}

	// Verify parsed values
	if grid.Ni != 100 {
		t.Errorf("Expected Ni=100, got %d", grid.Ni)
	}
	if grid.Nj != 50 {
		t.Errorf("Expected Nj=50, got %d", grid.Nj)
	}
	if grid.La1 != 20000000 {
		t.Errorf("Expected La1=20000000, got %d", grid.La1)
	}
	if grid.Lo1 != 250000000 {
		t.Errorf("Expected Lo1=250000000, got %d", grid.Lo1)
	}
	if grid.LaD != 20000000 {
		t.Errorf("Expected LaD=20000000, got %d", grid.LaD)
	}
	if grid.La2 != 40000000 {
		t.Errorf("Expected La2=40000000, got %d", grid.La2)
	}
	if grid.Lo2 != 280000000 {
		t.Errorf("Expected Lo2=280000000, got %d", grid.Lo2)
	}
	if grid.Di != 10000 {
		t.Errorf("Expected Di=10000, got %d", grid.Di)
	}
	if grid.Dj != 10000 {
		t.Errorf("Expected Dj=10000, got %d", grid.Dj)
	}
}

func TestParseMercatorGridTooShort(t *testing.T) {
	data := make([]byte, 50) // Too short
	_, err := ParseMercatorGrid(data)
	if err == nil {
		t.Error("Expected error for short data, got nil")
	}
}

func TestMercatorGridTemplateNumber(t *testing.T) {
	grid := &MercatorGrid{}
	if grid.TemplateNumber() != 10 {
		t.Errorf("Expected template number 10, got %d", grid.TemplateNumber())
	}
}

func TestMercatorGridNumPoints(t *testing.T) {
	grid := &MercatorGrid{Ni: 100, Nj: 50}
	if grid.NumPoints() != 5000 {
		t.Errorf("Expected 5000 points, got %d", grid.NumPoints())
	}
}

func TestMercatorGridString(t *testing.T) {
	grid := &MercatorGrid{
		Ni:  100,
		Nj:  50,
		La1: 20000000,  // 20°
		Lo1: 250000000, // 250°
		LaD: 20000000,  // 20°
	}
	str := grid.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
	t.Logf("String representation: %s", str)
}

func TestMercatorFirstLastGridPoint(t *testing.T) {
	grid := &MercatorGrid{
		La1: 20000000,  // 20°
		Lo1: 250000000, // 250°
		La2: 40000000,  // 40°
		Lo2: 280000000, // 280°
	}

	lat1, lon1 := grid.FirstGridPoint()
	if math.Abs(lat1-20.0) > 1e-5 {
		t.Errorf("Expected first grid point lat=20.0, got %.6f", lat1)
	}
	if math.Abs(lon1-250.0) > 1e-5 {
		t.Errorf("Expected first grid point lon=250.0, got %.6f", lon1)
	}

	lat2, lon2 := grid.LastGridPoint()
	if math.Abs(lat2-40.0) > 1e-5 {
		t.Errorf("Expected last grid point lat=40.0, got %.6f", lat2)
	}
	if math.Abs(lon2-280.0) > 1e-5 {
		t.Errorf("Expected last grid point lon=280.0, got %.6f", lon2)
	}
}

func TestMercatorScanningFlags(t *testing.T) {
	tests := []struct {
		name        string
		scanMode    uint8
		iNeg        bool
		jPos        bool
		consecutive bool
	}{
		{"default (0x00)", 0x00, false, false, true},
		{"i negative (0x80)", 0x80, true, false, true},
		{"j positive (0x40)", 0x40, false, true, true},
		{"not consecutive (0x20)", 0x20, false, false, false},
		{"combined (0xC0)", 0xC0, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid := &MercatorGrid{ScanningMode: tt.scanMode}
			iNeg, jPos, consec := grid.ScanningFlags()
			if iNeg != tt.iNeg {
				t.Errorf("iNegative: expected %v, got %v", tt.iNeg, iNeg)
			}
			if jPos != tt.jPos {
				t.Errorf("jPositive: expected %v, got %v", tt.jPos, jPos)
			}
			if consec != tt.consecutive {
				t.Errorf("consecutive: expected %v, got %v", tt.consecutive, consec)
			}
		})
	}
}

func TestMercatorCoordinates(t *testing.T) {
	// Create a small grid for testing coordinate generation
	grid := &MercatorGrid{
		Ni:           3,
		Nj:           2,
		La1:          0,        // 0° (equator)
		Lo1:          0,        // 0°
		LaD:          0,        // 0° reference latitude
		La2:          10000000, // 10°
		Lo2:          20000000, // 20°
		ScanningMode: 0x00,     // +i (W to E), -j (N to S)
		Di:           10000000, // 10 km
		Dj:           10000000, // 10 km
	}

	lats, lons := grid.Coordinates()

	// Verify we got the right number of points
	expectedPoints := 6 // 3 x 2
	if len(lats) != expectedPoints {
		t.Errorf("Expected %d latitudes, got %d", expectedPoints, len(lats))
	}
	if len(lons) != expectedPoints {
		t.Errorf("Expected %d longitudes, got %d", expectedPoints, len(lons))
	}

	// Verify first point is approximately at La1, Lo1
	// Note: Due to projection math, won't be exact
	if math.Abs(float64(lats[0])-0.0) > 0.1 {
		t.Errorf("First point latitude: expected ~0.0, got %.6f", lats[0])
	}
	if math.Abs(float64(lons[0])-0.0) > 0.1 {
		t.Errorf("First point longitude: expected ~0.0, got %.6f", lons[0])
	}

	// Verify all longitudes are normalized to [0, 360)
	for i, lon := range lons {
		if lon < 0 || lon >= 360 {
			t.Errorf("Longitude at index %d out of range: %.6f", i, lon)
		}
	}

	// Log coordinates for inspection
	t.Logf("Generated coordinates:")
	for i := 0; i < len(lats); i++ {
		t.Logf("  Point %d: lat=%.6f, lon=%.6f", i, lats[i], lons[i])
	}
}

func TestMercatorCoordinatesWithNegativeScan(t *testing.T) {
	// Test with east-to-west scanning
	grid := &MercatorGrid{
		Ni:           2,
		Nj:           2,
		La1:          0,
		Lo1:          10000000, // 10°
		LaD:          0,
		ScanningMode: 0x80, // -i (E to W), -j (N to S)
		Di:           10000000,
		Dj:           10000000,
	}

	lats, lons := grid.Coordinates()

	if len(lats) != 4 || len(lons) != 4 {
		t.Fatalf("Expected 4 points, got %d lats and %d lons", len(lats), len(lons))
	}

	// With -i scanning, longitudes should decrease
	// (though projection math may complicate this)
	t.Logf("Negative scan coordinates:")
	for i := 0; i < len(lats); i++ {
		t.Logf("  Point %d: lat=%.6f, lon=%.6f", i, lats[i], lons[i])
	}
}

func TestMercatorLatitudes(t *testing.T) {
	grid := &MercatorGrid{
		Ni:           2,
		Nj:           2,
		La1:          0,
		Lo1:          0,
		LaD:          0,
		ScanningMode: 0x00,
		Di:           10000000,
		Dj:           10000000,
	}

	lats := grid.Latitudes()
	if len(lats) != 4 {
		t.Errorf("Expected 4 latitudes, got %d", len(lats))
	}
}

func TestMercatorLongitudes(t *testing.T) {
	grid := &MercatorGrid{
		Ni:           2,
		Nj:           2,
		La1:          0,
		Lo1:          0,
		LaD:          0,
		ScanningMode: 0x00,
		Di:           10000000,
		Dj:           10000000,
	}

	lons := grid.Longitudes()
	if len(lons) != 4 {
		t.Errorf("Expected 4 longitudes, got %d", len(lons))
	}
}
