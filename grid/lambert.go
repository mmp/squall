package grid

import (
	"fmt"
	"math"

	"github.com/mmp/mgrib2/internal"
)

// LambertConformalGrid represents Grid Definition Template 3.30:
// Lambert Conformal projection.
//
// This projection is commonly used for regional models like HRRR and NAM.
type LambertConformalGrid struct {
	Nx                 uint32  // Number of points along x-axis
	Ny                 uint32  // Number of points along y-axis
	La1                int32   // Latitude of first grid point (micro-degrees)
	Lo1                int32   // Longitude of first grid point (micro-degrees)
	ResolutionFlags    uint8   // Resolution and component flags
	LaD                int32   // Latitude where Dx and Dy are specified (micro-degrees)
	LoV                int32   // Longitude of meridian parallel to y-axis (micro-degrees)
	Dx                 uint32  // X-direction grid length (meters)
	Dy                 uint32  // Y-direction grid length (meters)
	ProjectionCenter   uint8   // Projection center flag
	ScanningMode       uint8   // Scanning mode flags
	Latin1             int32   // First latitude from pole at which secant cone cuts sphere (micro-degrees)
	Latin2             int32   // Second latitude from pole (micro-degrees)
	LatSouthPole       int32   // Latitude of southern pole (micro-degrees)
	LonSouthPole       int32   // Longitude of southern pole (micro-degrees)
}

// ParseLambertConformalGrid parses Grid Definition Template 3.30.
func ParseLambertConformalGrid(data []byte) (*LambertConformalGrid, error) {
	if len(data) < 69 {
		return nil, fmt.Errorf("template 3.30 requires at least 69 bytes, got %d", len(data))
	}

	r := internal.NewReader(data[14:]) // Skip to template-specific data

	nx, _ := r.Uint32()
	ny, _ := r.Uint32()
	la1, _ := r.Int32()
	lo1, _ := r.Int32()
	resFlags, _ := r.Uint8()
	laD, _ := r.Int32()
	loV, _ := r.Int32()
	dx, _ := r.Uint32()
	dy, _ := r.Uint32()
	projCenter, _ := r.Uint8()
	scanMode, _ := r.Uint8()
	latin1, _ := r.Int32()
	latin2, _ := r.Int32()
	latSP, _ := r.Int32()
	lonSP, _ := r.Int32()

	return &LambertConformalGrid{
		Nx:               nx,
		Ny:               ny,
		La1:              la1,
		Lo1:              lo1,
		ResolutionFlags:  resFlags,
		LaD:              laD,
		LoV:              loV,
		Dx:               dx,
		Dy:               dy,
		ProjectionCenter: projCenter,
		ScanningMode:     scanMode,
		Latin1:           latin1,
		Latin2:           latin2,
		LatSouthPole:     latSP,
		LonSouthPole:     lonSP,
	}, nil
}

// TemplateNumber returns 30 for Lambert Conformal.
func (g *LambertConformalGrid) TemplateNumber() int {
	return 30
}

// GridType returns "Lambert Conformal".
func (g *LambertConformalGrid) GridType() string {
	return "Lambert Conformal"
}

// NumPoints returns the total number of grid points.
func (g *LambertConformalGrid) NumPoints() int {
	return int(g.Nx * g.Ny)
}

// Latitudes generates latitude values for all grid points.
//
// For Lambert Conformal projection, this requires inverse projection
// from grid coordinates (i, j) to geographic coordinates (lat, lon).
func (g *LambertConformalGrid) Latitudes() []float64 {
	lats, _ := g.Coordinates()
	return lats
}

// Longitudes generates longitude values for all grid points.
func (g *LambertConformalGrid) Longitudes() []float64 {
	_, lons := g.Coordinates()
	return lons
}

// Coordinates generates latitude and longitude arrays for all grid points.
//
// Uses inverse Lambert Conformal projection to convert from grid coordinates
// to geographic coordinates.
func (g *LambertConformalGrid) Coordinates() ([]float64, []float64) {
	nPoints := int(g.Nx * g.Ny)
	lats := make([]float64, nPoints)
	lons := make([]float64, nPoints)

	// Convert to degrees and radians
	lat1 := float64(g.La1) / 1e6
	lonV := float64(g.LoV) / 1e6
	latin1 := float64(g.Latin1) / 1e6
	latin2 := float64(g.Latin2) / 1e6

	// Convert to radians for projection calculations
	lat1Rad := lat1 * math.Pi / 180.0
	latin1Rad := latin1 * math.Pi / 180.0
	latin2Rad := latin2 * math.Pi / 180.0
	lonVRad := lonV * math.Pi / 180.0

	// Earth radius in meters (WGS84)
	const earthRadius = 6371229.0

	// Calculate projection parameters
	var n float64 // Cone constant
	if math.Abs(latin1-latin2) < 1e-6 {
		n = math.Sin(latin1Rad)
	} else {
		n = math.Log(math.Cos(latin1Rad)/math.Cos(latin2Rad)) /
			math.Log(math.Tan((math.Pi/4.0)+(latin2Rad/2.0))/math.Tan((math.Pi/4.0)+(latin1Rad/2.0)))
	}

	F := (math.Cos(latin1Rad) * math.Pow(math.Tan((math.Pi/4.0)+(latin1Rad/2.0)), n)) / n
	rho0 := earthRadius * F * math.Pow(math.Tan((math.Pi/4.0)+(lat1Rad/2.0)), -n)

	// Grid spacing in meters
	dx := float64(g.Dx)
	dy := float64(g.Dy)

	// Determine scanning direction
	iPositive := (g.ScanningMode & 0x80) == 0 // bit 1: 0 = +i, 1 = -i
	jPositive := (g.ScanningMode & 0x40) != 0 // bit 2: 0 = -j, 1 = +j

	idx := 0
	for j := uint32(0); j < g.Ny; j++ {
		for i := uint32(0); i < g.Nx; i++ {
			// Calculate grid coordinates relative to first point
			var x, y float64
			if iPositive {
				x = float64(i) * dx
			} else {
				x = float64(g.Nx-1-i) * dx
			}
			if jPositive {
				y = float64(j) * dy
			} else {
				y = float64(g.Ny-1-j) * dy
			}

			// Inverse Lambert Conformal projection
			rho := math.Sqrt(x*x + (rho0-y)*(rho0-y))
			if n < 0 {
				rho = -rho
			}

			theta := math.Atan2(x, rho0-y)

			// Convert to geographic coordinates
			lat := (2.0 * math.Atan(math.Pow((earthRadius*F)/rho, 1.0/n))) - (math.Pi / 2.0)
			lon := lonVRad + (theta / n)

			// Convert to degrees
			lats[idx] = lat * 180.0 / math.Pi
			lons[idx] = lon * 180.0 / math.Pi

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
func (g *LambertConformalGrid) String() string {
	return fmt.Sprintf("Lambert Conformal: %dx%d grid, La1=%.3f, Lo1=%.3f, LoV=%.3f",
		g.Nx, g.Ny,
		float64(g.La1)/1e6, float64(g.Lo1)/1e6, float64(g.LoV)/1e6)
}
