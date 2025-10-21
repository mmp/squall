package grid

import (
	"math"
	"testing"
)

func TestLambertConformalGrid_Coordinates(t *testing.T) {
	tests := []struct {
		name     string
		grid     *LambertConformalGrid
		testPts  map[int]struct{ lat, lon float64 } // index -> expected lat/lon
		tolerance float64
	}{
		{
			name: "HRRR Iowa Subset",
			grid: &LambertConformalGrid{
				Nx:           184,
				Ny:           123,
				La1:          40409178,  // 40.409178°
				Lo1:          263379162, // 263.379162°
				LoV:          262500000, // 262.500000°
				Latin1:       38500000,  // 38.500000°
				Latin2:       38500000,  // 38.500000°
				Dx:           3000000,   // 3000 m (in millimeters)
				Dy:           3000000,   // 3000 m (in millimeters)
				ScanningMode: 0x40,      // WE:SN
			},
			testPts: map[int]struct{ lat, lon float64 }{
				0:     {40.409178, 263.379162}, // (1,1)
				183:   {40.188670, 269.844169}, // (184,1)
				22448: {43.693514, 263.422460}, // (1,123)
				22631: {43.462999, 270.204243}, // (184,123)
			},
			tolerance: 0.001, // ~100m
		},
		{
			name: "HRRR CONUS",
			grid: &LambertConformalGrid{
				Nx:           1799,
				Ny:           1059,
				La1:          21138123,  // 21.138123°
				Lo1:          237280472, // 237.280472°
				LoV:          262500000, // 262.500000°
				Latin1:       38500000,  // 38.500000°
				Latin2:       38500000,  // 38.500000°
				Dx:           3000000,   // 3000 m (in millimeters)
				Dy:           3000000,   // 3000 m (in millimeters)
				ScanningMode: 0x40,      // WE:SN
			},
			testPts: map[int]struct{ lat, lon float64 }{
				0:       {21.138123, 237.280472}, // (1,1) bottom-left
				1798:    {21.140547, 287.710282}, // (1799,1) bottom-right
				1903342: {47.838623, 225.904520}, // (1,1059) top-left
				1905140: {47.842195, 299.082807}, // (1799,1059) top-right
				898600:  {37.687913, 262.494090}, // (900,500) center
			},
			tolerance: 0.001, // ~100m
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lats, lons := tt.grid.Coordinates()

			if len(lats) != int(tt.grid.Nx*tt.grid.Ny) {
				t.Errorf("Expected %d latitudes, got %d", tt.grid.Nx*tt.grid.Ny, len(lats))
			}
			if len(lons) != int(tt.grid.Nx*tt.grid.Ny) {
				t.Errorf("Expected %d longitudes, got %d", tt.grid.Nx*tt.grid.Ny, len(lons))
			}

			for idx, expected := range tt.testPts {
				if idx >= len(lats) || idx >= len(lons) {
					t.Errorf("Index %d out of range", idx)
					continue
				}

				latErr := math.Abs(float64(lats[idx]) - expected.lat)
				lonErr := math.Abs(float64(lons[idx]) - expected.lon)

				if latErr > tt.tolerance {
					t.Errorf("Index %d: latitude error %.6f° exceeds tolerance %.6f° (expected %.6f, got %.6f)",
						idx, latErr, tt.tolerance, expected.lat, lats[idx])
				}
				if lonErr > tt.tolerance {
					t.Errorf("Index %d: longitude error %.6f° exceeds tolerance %.6f° (expected %.6f, got %.6f)",
						idx, lonErr, tt.tolerance, expected.lon, lons[idx])
				}
			}
		})
	}
}

func TestLambertConformalGrid_NumPoints(t *testing.T) {
	g := &LambertConformalGrid{
		Nx: 184,
		Ny: 123,
	}

	expected := 184 * 123
	if g.NumPoints() != expected {
		t.Errorf("NumPoints() = %d, want %d", g.NumPoints(), expected)
	}
}

func TestLambertConformalGrid_TemplateNumber(t *testing.T) {
	g := &LambertConformalGrid{}
	if g.TemplateNumber() != 30 {
		t.Errorf("TemplateNumber() = %d, want 30", g.TemplateNumber())
	}
}
