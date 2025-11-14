package grid

import (
	"fmt"
	"math"

	"github.com/mmp/squall/internal"
)

// MercatorGrid represents Grid Definition Template 3.10:
// Mercator projection.
//
// This projection is a cylindrical map projection commonly used for
// ocean and maritime data.
type MercatorGrid struct {
	Ni           uint32 // Number of points along a parallel (longitude)
	Nj           uint32 // Number of points along a meridian (latitude)
	La1          int32  // Latitude of first grid point (micro-degrees)
	Lo1          int32  // Longitude of first grid point (micro-degrees)
	ResFlags     uint8  // Resolution and component flags
	LaD          int32  // Latitude where Mercator projection intersects Earth (micro-degrees)
	La2          int32  // Latitude of last grid point (micro-degrees)
	Lo2          int32  // Longitude of last grid point (micro-degrees)
	ScanningMode uint8  // Scanning mode flags
	Orientation  uint32 // Grid orientation angle (millidegrees, 0-90°)
	Di           uint32 // Longitudinal direction grid length (millimeters at LaD)
	Dj           uint32 // Latitudinal direction grid length (millimeters at LaD)
}

// ParseMercatorGrid parses Grid Definition Template 3.10.
func ParseMercatorGrid(data []byte) (*MercatorGrid, error) {
	if len(data) < 58 {
		return nil, fmt.Errorf("template 3.10 requires at least 58 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Skip shape of earth (1 byte) and related parameters (15 bytes)
	// Following same pattern as LatLonGrid and LambertConformalGrid
	_ = r.Skip(16)

	// Read grid dimensions
	ni, _ := r.Uint32()
	nj, _ := r.Uint32()
	la1, _ := r.Int32()
	lo1, _ := r.Int32()
	resFlags, _ := r.Uint8()
	laD, _ := r.Int32()
	la2, _ := r.Int32()
	lo2, _ := r.Int32()
	scanMode, _ := r.Uint8()
	orientation, _ := r.Uint32()
	di, _ := r.Uint32()
	dj, _ := r.Uint32()

	return &MercatorGrid{
		Ni:           ni,
		Nj:           nj,
		La1:          la1,
		Lo1:          lo1,
		ResFlags:     resFlags,
		LaD:          laD,
		La2:          la2,
		Lo2:          lo2,
		ScanningMode: scanMode,
		Orientation:  orientation,
		Di:           di,
		Dj:           dj,
	}, nil
}

// TemplateNumber returns 10 for Mercator.
func (g *MercatorGrid) TemplateNumber() int {
	return 10
}

// GridType returns "Mercator".
func (g *MercatorGrid) GridType() string {
	return "Mercator"
}

// NumPoints returns the total number of grid points.
func (g *MercatorGrid) NumPoints() int {
	return int(g.Ni * g.Nj)
}

// Latitudes generates latitude values for all grid points.
func (g *MercatorGrid) Latitudes() []float32 {
	lats, _ := g.Coordinates()
	return lats
}

// Longitudes generates longitude values for all grid points.
func (g *MercatorGrid) Longitudes() []float32 {
	_, lons := g.Coordinates()
	return lons
}

// Coordinates generates latitude and longitude arrays for all grid points.
//
// Uses inverse Mercator projection to convert from grid coordinates
// to geographic coordinates.
func (g *MercatorGrid) Coordinates() ([]float32, []float32) {
	nPoints := int(g.Ni * g.Nj)
	lats := make([]float32, nPoints)
	lons := make([]float32, nPoints)

	// Convert to degrees - use float64 for trig operations
	lat1 := float64(g.La1) / 1e6 // Latitude of first grid point
	lon1 := float64(g.Lo1) / 1e6 // Longitude of first grid point
	laD := float64(g.LaD) / 1e6  // Reference latitude for grid spacing

	// Convert to radians for projection calculations
	lat1Rad := lat1 * math.Pi / 180.0
	laDRad := laD * math.Pi / 180.0

	// Earth radius in meters (WGS84)
	const earthRadius = 6371229.0

	// Grid spacing in meters (Di and Dj are stored in millimeters)
	dx := float64(g.Di) / 1000.0
	dy := float64(g.Dj) / 1000.0

	// Scale factor at reference latitude LaD
	// In Mercator projection, distances are scaled by 1/cos(LaD)
	scaleFactor := 1.0 / math.Cos(laDRad)

	// Calculate the projection coordinates of the first grid point
	// Forward Mercator: x = R * λ, y = R * ln(tan(π/4 + φ/2))
	lon1Rad := lon1 * math.Pi / 180.0
	x0 := earthRadius * lon1Rad
	y0 := earthRadius * math.Log(math.Tan(math.Pi/4.0+lat1Rad/2.0))

	// Determine scanning direction
	iPositive := (g.ScanningMode & 0x80) == 0 // bit 0: 0 = +i (west to east), 1 = -i
	jPositive := (g.ScanningMode & 0x40) != 0 // bit 1: 0 = -j (north to south), 1 = +j

	idx := 0
	for j := uint32(0); j < g.Nj; j++ {
		for i := uint32(0); i < g.Ni; i++ {
			// Calculate grid coordinates relative to first point
			var deltaX, deltaY float64
			if iPositive {
				deltaX = float64(i) * dx * scaleFactor
			} else {
				deltaX = -float64(i) * dx * scaleFactor
			}
			if jPositive {
				deltaY = float64(j) * dy * scaleFactor
			} else {
				deltaY = -float64(j) * dy * scaleFactor
			}

			// Projection coordinates for this grid point
			x := x0 + deltaX
			y := y0 + deltaY

			// Inverse Mercator projection
			// λ = x / R
			// φ = 2 * arctan(exp(y/R)) - π/2
			lon := x / earthRadius
			lat := 2.0*math.Atan(math.Exp(y/earthRadius)) - (math.Pi / 2.0)

			// Convert to degrees and store as float32
			lats[idx] = float32(lat * 180.0 / math.Pi)
			lons[idx] = float32(lon * 180.0 / math.Pi)

			// Normalize longitude to [0, 360)
			for lons[idx] < 0 {
				lons[idx] += 360
			}
			for lons[idx] >= 360 {
				lons[idx] -= 360
			}

			idx++
		}
	}

	return lats, lons
}

// String returns a human-readable description.
func (g *MercatorGrid) String() string {
	return fmt.Sprintf("Mercator: %dx%d grid, La1=%.3f, Lo1=%.3f, LaD=%.3f",
		g.Ni, g.Nj,
		float64(g.La1)/1e6, float64(g.Lo1)/1e6, float64(g.LaD)/1e6)
}

// FirstGridPoint returns the latitude and longitude of the first grid point in degrees.
func (g *MercatorGrid) FirstGridPoint() (lat, lon float64) {
	return float64(g.La1) / 1e6, float64(g.Lo1) / 1e6
}

// LastGridPoint returns the latitude and longitude of the last grid point in degrees.
func (g *MercatorGrid) LastGridPoint() (lat, lon float64) {
	return float64(g.La2) / 1e6, float64(g.Lo2) / 1e6
}

// ScanningFlags returns the scanning mode flags as individual booleans.
//
// Returns:
//   - iNegative: true if points scan in -i direction (east to west)
//   - jPositive: true if points scan in +j direction (south to north)
//   - consecutive: true if adjacent points in i direction are consecutive
func (g *MercatorGrid) ScanningFlags() (iNegative, jPositive, consecutive bool) {
	iNegative = (g.ScanningMode & 0x80) != 0   // Bit 0
	jPositive = (g.ScanningMode & 0x40) != 0   // Bit 1
	consecutive = (g.ScanningMode & 0x20) == 0 // Bit 2 (0 = consecutive)
	return
}
