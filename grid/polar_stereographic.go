package grid

import (
	"fmt"
	"math"

	"github.com/mmp/squall/internal"
)

// PolarStereographicGrid represents Grid Definition Template 3.20:
// Polar Stereographic projection.
//
// This projection is commonly used for polar regions (Arctic and Antarctic)
// for weather forecasting and sea ice monitoring.
type PolarStereographicGrid struct {
	Nx               uint32 // Number of points along x-axis
	Ny               uint32 // Number of points along y-axis
	La1              int32  // Latitude of first grid point (micro-degrees)
	Lo1              uint32 // Longitude of first grid point (micro-degrees, unsigned)
	ResFlags         uint8  // Resolution and component flags
	LaD              int32  // Reference latitude for Dx/Dy specification (micro-degrees)
	LoV              int32  // Orientation of the grid (longitude parallel to y-axis, micro-degrees)
	Dx               uint32 // X-direction grid length (millimeters)
	Dy               uint32 // Y-direction grid length (millimeters)
	ProjectionCenter uint8  // Projection center flag (north/south pole)
	ScanningMode     uint8  // Scanning mode flags
}

// ParsePolarStereographicGrid parses Grid Definition Template 3.20.
func ParsePolarStereographicGrid(data []byte) (*PolarStereographicGrid, error) {
	if len(data) < 51 {
		return nil, fmt.Errorf("template 3.20 requires at least 51 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Skip shape of earth (1 byte) and related parameters (15 bytes)
	// Following same pattern as other grid types
	_ = r.Skip(16)

	// Read grid dimensions
	nx, _ := r.Uint32()
	ny, _ := r.Uint32()
	la1, _ := r.Int32()
	lo1, _ := r.Uint32() // Note: unsigned for Lo1 in this template
	resFlags, _ := r.Uint8()
	laD, _ := r.Int32()
	loV, _ := r.Int32()
	dx, _ := r.Uint32()
	dy, _ := r.Uint32()
	projCenter, _ := r.Uint8()
	scanMode, _ := r.Uint8()

	return &PolarStereographicGrid{
		Nx:               nx,
		Ny:               ny,
		La1:              la1,
		Lo1:              lo1,
		ResFlags:         resFlags,
		LaD:              laD,
		LoV:              loV,
		Dx:               dx,
		Dy:               dy,
		ProjectionCenter: projCenter,
		ScanningMode:     scanMode,
	}, nil
}

// TemplateNumber returns 20 for Polar Stereographic.
func (g *PolarStereographicGrid) TemplateNumber() int {
	return 20
}

// GridType returns "Polar Stereographic".
func (g *PolarStereographicGrid) GridType() string {
	if g.IsNorthPole() {
		return "Polar Stereographic (North Pole)"
	}
	return "Polar Stereographic (South Pole)"
}

// NumPoints returns the total number of grid points.
func (g *PolarStereographicGrid) NumPoints() int {
	return int(g.Nx * g.Ny)
}

// IsNorthPole returns true if this is a North Pole projection.
// Bit 0 of ProjectionCenter: 0 = North Pole, 1 = South Pole
func (g *PolarStereographicGrid) IsNorthPole() bool {
	return (g.ProjectionCenter & 0x80) == 0
}

// Latitudes generates latitude values for all grid points.
func (g *PolarStereographicGrid) Latitudes() []float32 {
	lats, _ := g.Coordinates()
	return lats
}

// Longitudes generates longitude values for all grid points.
func (g *PolarStereographicGrid) Longitudes() []float32 {
	_, lons := g.Coordinates()
	return lons
}

// Coordinates generates latitude and longitude arrays for all grid points.
//
// Uses inverse Polar Stereographic projection to convert from grid coordinates
// to geographic coordinates. Handles both North and South polar projections.
func (g *PolarStereographicGrid) Coordinates() ([]float32, []float32) {
	nPoints := int(g.Nx * g.Ny)
	lats := make([]float32, nPoints)
	lons := make([]float32, nPoints)

	// Convert to degrees and radians - use float64 for trig operations
	lat1 := float64(g.La1) / 1e6 // Latitude of first grid point
	lon1 := float64(g.Lo1) / 1e6 // Longitude of first grid point
	laD := float64(g.LaD) / 1e6  // Reference latitude for grid spacing
	loV := float64(g.LoV) / 1e6  // Orientation longitude

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180.0
	lon1Rad := lon1 * math.Pi / 180.0
	laDRad := laD * math.Pi / 180.0
	loVRad := loV * math.Pi / 180.0

	// Earth radius in meters (WGS84)
	const earthRadius = 6371229.0

	// Grid spacing in meters (Dx and Dy are stored in millimeters)
	dx := float64(g.Dx) / 1000.0
	dy := float64(g.Dy) / 1000.0

	// For polar stereographic projection with standard parallel at LaD,
	// following USGS GCTP formulas for spherical Earth:
	// mcs = cos(LaD) and tcs = tan((90° - LaD)/2)
	// The scale factor is: mcs/tcs
	mcs := math.Cos(math.Abs(laDRad))
	tcs := math.Tan((math.Pi/2.0 - math.Abs(laDRad)) / 2.0)

	// Determine if North or South pole projection
	isNorth := g.IsNorthPole()

	// Calculate projection coordinates of first grid point (La1, Lo1)
	// Forward polar stereographic projection using USGS GCTP formula
	var x0, y0 float64
	if isNorth {
		// North Pole projection
		// t = tan((π/2 - lat) / 2)
		// rho = R * mcs * t / tcs
		t := math.Tan((math.Pi/2.0 - lat1Rad) / 2.0)
		rho := earthRadius * mcs * t / tcs
		theta := lon1Rad - loVRad
		x0 = rho * math.Sin(theta)
		y0 = -rho * math.Cos(theta)
	} else {
		// South Pole projection
		// t = tan((π/2 + lat) / 2)
		// rho = R * mcs * t / tcs
		t := math.Tan((math.Pi/2.0 + lat1Rad) / 2.0)
		rho := earthRadius * mcs * t / tcs
		theta := lon1Rad - loVRad
		x0 = rho * math.Sin(theta)
		y0 = rho * math.Cos(theta)
	}

	// Determine scanning direction
	iPositive := (g.ScanningMode & 0x80) == 0 // bit 0: 0 = +i, 1 = -i
	jPositive := (g.ScanningMode & 0x40) != 0 // bit 1: 0 = -j, 1 = +j

	idx := 0
	for j := uint32(0); j < g.Ny; j++ {
		for i := uint32(0); i < g.Nx; i++ {
			// Calculate grid coordinates relative to first point
			var deltaX, deltaY float64
			if iPositive {
				deltaX = float64(i) * dx
			} else {
				deltaX = -float64(i) * dx
			}
			if jPositive {
				deltaY = float64(j) * dy
			} else {
				deltaY = -float64(j) * dy
			}

			// Projection coordinates for this grid point
			x := x0 + deltaX
			y := y0 + deltaY

			// Inverse polar stereographic projection
			rho := math.Sqrt(x*x + y*y)

			var lat, lon float64
			if isNorth {
				// North Pole projection
				if rho == 0 {
					lat = math.Pi / 2.0 // At the pole (in radians)
					lon = 0.0
				} else {
					// USGS GCTP formula for inverse:
					// ts = rho * tcs / (R * mcs)
					// lat = π/2 - 2*arctan(ts)
					ts := rho * tcs / (earthRadius * mcs)
					lat = (math.Pi / 2.0) - 2.0*math.Atan(ts)
					theta := math.Atan2(x, -y)
					lon = loVRad + theta
				}
			} else {
				// South Pole projection
				if rho == 0 {
					lat = -math.Pi / 2.0 // At the pole (in radians)
					lon = 0.0
				} else {
					// USGS GCTP formula for inverse:
					// ts = rho * tcs / (R * mcs)
					// lat = -π/2 + 2*arctan(ts)
					ts := rho * tcs / (earthRadius * mcs)
					lat = -(math.Pi / 2.0) + 2.0*math.Atan(ts)
					theta := math.Atan2(x, y)
					lon = loVRad + theta
				}
			}

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
func (g *PolarStereographicGrid) String() string {
	pole := "North"
	if !g.IsNorthPole() {
		pole = "South"
	}
	return fmt.Sprintf("Polar Stereographic (%s): %dx%d grid, La1=%.3f, Lo1=%.3f, LoV=%.3f",
		pole, g.Nx, g.Ny,
		float64(g.La1)/1e6, float64(g.Lo1)/1e6, float64(g.LoV)/1e6)
}

// FirstGridPoint returns the latitude and longitude of the first grid point in degrees.
func (g *PolarStereographicGrid) FirstGridPoint() (lat, lon float64) {
	return float64(g.La1) / 1e6, float64(g.Lo1) / 1e6
}

// ScanningFlags returns the scanning mode flags as individual booleans.
//
// Returns:
//   - iNegative: true if points scan in -i direction
//   - jPositive: true if points scan in +j direction
//   - consecutive: true if adjacent points in i direction are consecutive
func (g *PolarStereographicGrid) ScanningFlags() (iNegative, jPositive, consecutive bool) {
	iNegative = (g.ScanningMode & 0x80) != 0   // Bit 0
	jPositive = (g.ScanningMode & 0x40) != 0   // Bit 1
	consecutive = (g.ScanningMode & 0x20) == 0 // Bit 2 (0 = consecutive)
	return
}
