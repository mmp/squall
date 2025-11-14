package grid

import (
	"math"
	"testing"
)

func TestParsePolarStereographicGrid(t *testing.T) {
	// Create a minimal valid Template 3.20 data (51 bytes)
	data := make([]byte, 51)

	// Skip shape of earth parameters (bytes 0-15)

	// Nx = 100 (bytes 16-19)
	data[16] = 0x00
	data[17] = 0x00
	data[18] = 0x00
	data[19] = 0x64 // 100

	// Ny = 50 (bytes 20-23)
	data[20] = 0x00
	data[21] = 0x00
	data[22] = 0x00
	data[23] = 0x32 // 50

	// La1 = 60000000 micro-degrees = 60° (bytes 24-27)
	// 60000000 = 0x03938700
	data[24] = 0x03
	data[25] = 0x93
	data[26] = 0x87
	data[27] = 0x00

	// Lo1 = 180000000 micro-degrees = 180° (bytes 28-31, unsigned)
	// 180000000 = 0x0ABA9500
	data[28] = 0x0A
	data[29] = 0xBA
	data[30] = 0x95
	data[31] = 0x00

	// ResFlags = 0x00 (byte 32)
	data[32] = 0x00

	// LaD = 60000000 micro-degrees = 60° (bytes 33-36)
	data[33] = 0x03
	data[34] = 0x93
	data[35] = 0x87
	data[36] = 0x00

	// LoV = 0 (bytes 37-40, orientation longitude)
	data[37] = 0x00
	data[38] = 0x00
	data[39] = 0x00
	data[40] = 0x00

	// Dx = 10000 millimeters (bytes 41-44)
	data[41] = 0x00
	data[42] = 0x00
	data[43] = 0x27
	data[44] = 0x10

	// Dy = 10000 millimeters (bytes 45-48)
	data[45] = 0x00
	data[46] = 0x00
	data[47] = 0x27
	data[48] = 0x10

	// ProjectionCenter = 0x00 (byte 49) - North Pole
	data[49] = 0x00

	// ScanningMode = 0x00 (byte 50)
	data[50] = 0x00

	grid, err := ParsePolarStereographicGrid(data)
	if err != nil {
		t.Fatalf("ParsePolarStereographicGrid failed: %v", err)
	}

	// Verify parsed values
	if grid.Nx != 100 {
		t.Errorf("Expected Nx=100, got %d", grid.Nx)
	}
	if grid.Ny != 50 {
		t.Errorf("Expected Ny=50, got %d", grid.Ny)
	}
	if grid.La1 != 60000000 {
		t.Errorf("Expected La1=60000000, got %d", grid.La1)
	}
	if grid.Lo1 != 180000000 {
		t.Errorf("Expected Lo1=180000000, got %d", grid.Lo1)
	}
	if grid.LaD != 60000000 {
		t.Errorf("Expected LaD=60000000, got %d", grid.LaD)
	}
	if grid.LoV != 0 {
		t.Errorf("Expected LoV=0, got %d", grid.LoV)
	}
	if grid.Dx != 10000 {
		t.Errorf("Expected Dx=10000, got %d", grid.Dx)
	}
	if grid.Dy != 10000 {
		t.Errorf("Expected Dy=10000, got %d", grid.Dy)
	}
	if grid.ProjectionCenter != 0x00 {
		t.Errorf("Expected ProjectionCenter=0x00, got 0x%02x", grid.ProjectionCenter)
	}
}

func TestParsePolarStereographicGridTooShort(t *testing.T) {
	data := make([]byte, 50) // Too short
	_, err := ParsePolarStereographicGrid(data)
	if err == nil {
		t.Error("Expected error for short data, got nil")
	}
}

func TestPolarStereographicGridTemplateNumber(t *testing.T) {
	grid := &PolarStereographicGrid{}
	if grid.TemplateNumber() != 20 {
		t.Errorf("Expected template number 20, got %d", grid.TemplateNumber())
	}
}

func TestPolarStereographicGridNumPoints(t *testing.T) {
	grid := &PolarStereographicGrid{Nx: 100, Ny: 50}
	if grid.NumPoints() != 5000 {
		t.Errorf("Expected 5000 points, got %d", grid.NumPoints())
	}
}

func TestPolarStereographicIsNorthPole(t *testing.T) {
	tests := []struct {
		name          string
		projCenter    uint8
		expectedNorth bool
	}{
		{"North Pole (0x00)", 0x00, true},
		{"North Pole (0x40)", 0x40, true},
		{"South Pole (0x80)", 0x80, false},
		{"South Pole (0xC0)", 0xC0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid := &PolarStereographicGrid{ProjectionCenter: tt.projCenter}
			if grid.IsNorthPole() != tt.expectedNorth {
				t.Errorf("Expected IsNorthPole=%v, got %v", tt.expectedNorth, grid.IsNorthPole())
			}
		})
	}
}

func TestPolarStereographicGridType(t *testing.T) {
	northGrid := &PolarStereographicGrid{ProjectionCenter: 0x00}
	if northGrid.GridType() != "Polar Stereographic (North Pole)" {
		t.Errorf("Unexpected north grid type: %s", northGrid.GridType())
	}

	southGrid := &PolarStereographicGrid{ProjectionCenter: 0x80}
	if southGrid.GridType() != "Polar Stereographic (South Pole)" {
		t.Errorf("Unexpected south grid type: %s", southGrid.GridType())
	}
}

func TestPolarStereographicGridString(t *testing.T) {
	grid := &PolarStereographicGrid{
		Nx:               100,
		Ny:               50,
		La1:              60000000,  // 60°
		Lo1:              180000000, // 180°
		LoV:              0,
		ProjectionCenter: 0x00, // North Pole
	}
	str := grid.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
	t.Logf("String representation: %s", str)
}

func TestPolarStereographicFirstGridPoint(t *testing.T) {
	grid := &PolarStereographicGrid{
		La1: 60000000,  // 60°
		Lo1: 180000000, // 180°
	}

	lat, lon := grid.FirstGridPoint()
	if math.Abs(lat-60.0) > 1e-5 {
		t.Errorf("Expected first grid point lat=60.0, got %.6f", lat)
	}
	if math.Abs(lon-180.0) > 1e-5 {
		t.Errorf("Expected first grid point lon=180.0, got %.6f", lon)
	}
}

func TestPolarStereographicScanningFlags(t *testing.T) {
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
			grid := &PolarStereographicGrid{ScanningMode: tt.scanMode}
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

func TestPolarStereographicCoordinatesNorthPole(t *testing.T) {
	// Create a small North Pole grid for testing
	grid := &PolarStereographicGrid{
		Nx:               3,
		Ny:               2,
		La1:              70000000, // 70°N
		Lo1:              0,        // 0° (Prime Meridian)
		LaD:              60000000, // 60°N reference latitude
		LoV:              0,        // 0° orientation
		Dx:               10000000, // 10 km
		Dy:               10000000, // 10 km
		ProjectionCenter: 0x00,     // North Pole
		ScanningMode:     0x00,     // +i, -j
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
	if math.Abs(float64(lats[0])-70.0) > 1.0 {
		t.Errorf("First point latitude: expected ~70.0, got %.6f", lats[0])
	}

	// Verify all latitudes are in valid range
	for i, lat := range lats {
		if lat < -90 || lat > 90 {
			t.Errorf("Latitude at index %d out of range: %.6f", i, lat)
		}
	}

	// Verify all longitudes are normalized to [0, 360)
	for i, lon := range lons {
		if lon < 0 || lon >= 360 {
			t.Errorf("Longitude at index %d out of range: %.6f", i, lon)
		}
	}

	// Log coordinates for inspection
	t.Logf("North Pole grid coordinates:")
	for i := 0; i < len(lats); i++ {
		t.Logf("  Point %d: lat=%.6f, lon=%.6f", i, lats[i], lons[i])
	}
}

func TestPolarStereographicCoordinatesSouthPole(t *testing.T) {
	// Create a small South Pole grid for testing
	grid := &PolarStereographicGrid{
		Nx:               3,
		Ny:               2,
		La1:              -70000000, // 70°S
		Lo1:              0,         // 0° (Prime Meridian)
		LaD:              -60000000, // 60°S reference latitude
		LoV:              0,         // 0° orientation
		Dx:               10000000,  // 10 km
		Dy:               10000000,  // 10 km
		ProjectionCenter: 0x80,      // South Pole
		ScanningMode:     0x00,      // +i, -j
	}

	lats, lons := grid.Coordinates()

	// Verify we got the right number of points
	expectedPoints := 6
	if len(lats) != expectedPoints {
		t.Errorf("Expected %d latitudes, got %d", expectedPoints, len(lats))
	}
	if len(lons) != expectedPoints {
		t.Errorf("Expected %d longitudes, got %d", expectedPoints, len(lons))
	}

	// Verify first point is approximately at La1, Lo1
	if math.Abs(float64(lats[0])+70.0) > 1.0 {
		t.Errorf("First point latitude: expected ~-70.0, got %.6f", lats[0])
	}

	// Verify all latitudes are in valid range
	for i, lat := range lats {
		if lat < -90 || lat > 90 {
			t.Errorf("Latitude at index %d out of range: %.6f", i, lat)
		}
	}

	// Verify all longitudes are normalized to [0, 360)
	for i, lon := range lons {
		if lon < 0 || lon >= 360 {
			t.Errorf("Longitude at index %d out of range: %.6f", i, lon)
		}
	}

	// Log coordinates for inspection
	t.Logf("South Pole grid coordinates:")
	for i := 0; i < len(lats); i++ {
		t.Logf("  Point %d: lat=%.6f, lon=%.6f", i, lats[i], lons[i])
	}
}

func TestPolarStereographicLatitudes(t *testing.T) {
	grid := &PolarStereographicGrid{
		Nx:               2,
		Ny:               2,
		La1:              70000000,
		Lo1:              0,
		LaD:              60000000,
		LoV:              0,
		Dx:               10000000,
		Dy:               10000000,
		ProjectionCenter: 0x00,
		ScanningMode:     0x00,
	}

	lats := grid.Latitudes()
	if len(lats) != 4 {
		t.Errorf("Expected 4 latitudes, got %d", len(lats))
	}
}

func TestPolarStereographicLongitudes(t *testing.T) {
	grid := &PolarStereographicGrid{
		Nx:               2,
		Ny:               2,
		La1:              70000000,
		Lo1:              0,
		LaD:              60000000,
		LoV:              0,
		Dx:               10000000,
		Dy:               10000000,
		ProjectionCenter: 0x00,
		ScanningMode:     0x00,
	}

	lons := grid.Longitudes()
	if len(lons) != 4 {
		t.Errorf("Expected 4 longitudes, got %d", len(lons))
	}
}

func TestPolarStereographicCoordinatesAtPole(t *testing.T) {
	// Test grid that includes the pole (90°N)
	grid := &PolarStereographicGrid{
		Nx:               2,
		Ny:               2,
		La1:              90000000, // Exactly at North Pole
		Lo1:              0,
		LaD:              60000000,
		LoV:              0,
		Dx:               10000000,
		Dy:               10000000,
		ProjectionCenter: 0x00,
		ScanningMode:     0x00,
	}

	lats, lons := grid.Coordinates()

	// Should handle pole gracefully (rho = 0 case)
	if len(lats) != 4 || len(lons) != 4 {
		t.Errorf("Expected 4 points, got %d lats and %d lons", len(lats), len(lons))
	}

	// First point should be very close to 90°
	if math.Abs(float64(lats[0])-90.0) > 1.0 {
		t.Errorf("Expected first point near 90°, got %.6f", lats[0])
	}

	t.Logf("Coordinates near North Pole:")
	for i := 0; i < len(lats); i++ {
		t.Logf("  Point %d: lat=%.6f, lon=%.6f", i, lats[i], lons[i])
	}
}
