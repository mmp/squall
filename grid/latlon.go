package grid

import (
	"fmt"

	"github.com/mmp/mgrib2/internal"
)

// LatLonGrid represents a GRIB2 Latitude/Longitude grid (Template 3.0).
//
// This is the most common grid type, consisting of a regular grid with
// constant spacing in latitude and longitude.
type LatLonGrid struct {
	Ni           uint32  // Number of points along a parallel (longitude)
	Nj           uint32  // Number of points along a meridian (latitude)
	La1          int32   // Latitude of first grid point (millidegrees)
	Lo1          int32   // Longitude of first grid point (millidegrees)
	ResFlags     uint8   // Resolution and component flags
	La2          int32   // Latitude of last grid point (millidegrees)
	Lo2          int32   // Longitude of last grid point (millidegrees)
	Di           uint32  // i direction increment (millidegrees)
	Dj           uint32  // j direction increment (millidegrees)
	ScanningMode uint8   // Scanning mode (Table 3.4)
}

// ParseLatLonGrid parses a Lat/Lon grid from template data (Template 3.0).
//
// The template data should be 72 bytes for Template 3.0.
func ParseLatLonGrid(data []byte) (*LatLonGrid, error) {
	if len(data) < 72 {
		return nil, fmt.Errorf("template 3.0 requires at least 72 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Skip shape of earth (1 byte) and related parameters (15 bytes)
	// We'll implement proper earth shape handling in a future phase
	r.Skip(16)

	// Read grid dimensions
	ni, _ := r.Uint32()
	nj, _ := r.Uint32()

	// Skip basic angle and subdivisions (8 bytes)
	r.Skip(8)

	// Read grid points
	la1, _ := r.Int32()
	lo1, _ := r.Int32()
	resFlags, _ := r.Uint8()
	la2, _ := r.Int32()
	lo2, _ := r.Int32()
	di, _ := r.Uint32()
	dj, _ := r.Uint32()
	scanningMode, _ := r.Uint8()

	return &LatLonGrid{
		Ni:           ni,
		Nj:           nj,
		La1:          la1,
		Lo1:          lo1,
		ResFlags:     resFlags,
		La2:          la2,
		Lo2:          lo2,
		Di:           di,
		Dj:           dj,
		ScanningMode: scanningMode,
	}, nil
}

// TemplateNumber returns 0 for Lat/Lon grids.
func (g *LatLonGrid) TemplateNumber() int {
	return 0
}

// NumPoints returns the total number of grid points.
func (g *LatLonGrid) NumPoints() int {
	return int(g.Ni * g.Nj)
}

// String returns a human-readable description of the grid.
func (g *LatLonGrid) String() string {
	return fmt.Sprintf("Lat/Lon grid: %d x %d points (%.3f°, %.3f°) to (%.3f°, %.3f°)",
		g.Ni, g.Nj,
		float64(g.La1)/1000.0, float64(g.Lo1)/1000.0,
		float64(g.La2)/1000.0, float64(g.Lo2)/1000.0)
}

// FirstGridPoint returns the latitude and longitude of the first grid point in degrees.
func (g *LatLonGrid) FirstGridPoint() (lat, lon float64) {
	return float64(g.La1) / 1000.0, float64(g.Lo1) / 1000.0
}

// LastGridPoint returns the latitude and longitude of the last grid point in degrees.
func (g *LatLonGrid) LastGridPoint() (lat, lon float64) {
	return float64(g.La2) / 1000.0, float64(g.Lo2) / 1000.0
}

// Increment returns the i and j direction increments in degrees.
func (g *LatLonGrid) Increment() (di, dj float64) {
	return float64(g.Di) / 1000.0, float64(g.Dj) / 1000.0
}

// ScanningFlags returns the scanning mode flags as individual booleans.
//
// Returns:
//   - iNegative: true if points scan in -i direction (east to west)
//   - jPositive: true if points scan in +j direction (south to north)
//   - consecutive: true if adjacent points in i direction are consecutive
func (g *LatLonGrid) ScanningFlags() (iNegative, jPositive, consecutive bool) {
	iNegative = (g.ScanningMode & 0x80) != 0  // Bit 0
	jPositive = (g.ScanningMode & 0x40) != 0  // Bit 1
	consecutive = (g.ScanningMode & 0x20) == 0 // Bit 2 (0 = consecutive)
	return
}

// Latitudes generates an array of latitude values for all grid points.
//
// The latitudes are returned in grid scan order (respecting the scanning mode).
// For the most common scanning mode (0x00: +i west-to-east, -j north-to-south),
// the latitudes vary by row.
//
// Returns a slice of latitude values in degrees, one per grid point.
func (g *LatLonGrid) Latitudes() []float32 {
	_, jPositive, consecutive := g.ScanningFlags()

	numPoints := g.NumPoints()
	lats := make([]float32, numPoints)

	// Calculate latitude increment in degrees
	dj := float32(g.Dj) / 1000.0
	if !jPositive {
		dj = -dj // Scanning north to south
	}

	// Starting latitude
	lat1 := float32(g.La1) / 1000.0

	if consecutive {
		// Consecutive points in i direction (most common)
		// Latitude varies by j (row)
		for j := uint32(0); j < g.Nj; j++ {
			lat := lat1 + float32(j)*dj
			for i := uint32(0); i < g.Ni; i++ {
				idx := j*g.Ni + i
				lats[idx] = lat
			}
		}
	} else {
		// Consecutive points in j direction
		// Latitude varies within each column
		for i := uint32(0); i < g.Ni; i++ {
			for j := uint32(0); j < g.Nj; j++ {
				lat := lat1 + float32(j)*dj
				idx := i*g.Nj + j
				lats[idx] = lat
			}
		}
	}

	return lats
}

// Longitudes generates an array of longitude values for all grid points.
//
// The longitudes are returned in grid scan order (respecting the scanning mode).
// For the most common scanning mode (0x00: +i west-to-east, -j north-to-south),
// the longitudes vary by column.
//
// Longitudes are normalized to the range [0, 360) degrees.
//
// Returns a slice of longitude values in degrees, one per grid point.
func (g *LatLonGrid) Longitudes() []float32 {
	iNegative, _, consecutive := g.ScanningFlags()

	numPoints := g.NumPoints()
	lons := make([]float32, numPoints)

	// Calculate longitude increment in degrees
	di := float32(g.Di) / 1000.0
	if iNegative {
		di = -di // Scanning east to west
	}

	// Starting longitude
	lon1 := float32(g.Lo1) / 1000.0

	if consecutive {
		// Consecutive points in i direction (most common)
		// Longitude varies by i (column)
		for j := uint32(0); j < g.Nj; j++ {
			for i := uint32(0); i < g.Ni; i++ {
				lon := lon1 + float32(i)*di
				// Normalize to [0, 360)
				for lon < 0 {
					lon += 360.0
				}
				for lon >= 360.0 {
					lon -= 360.0
				}
				idx := j*g.Ni + i
				lons[idx] = lon
			}
		}
	} else {
		// Consecutive points in j direction
		// Longitude varies between columns
		for i := uint32(0); i < g.Ni; i++ {
			lon := lon1 + float32(i)*di
			// Normalize to [0, 360)
			for lon < 0 {
				lon += 360.0
			}
			for lon >= 360.0 {
				lon -= 360.0
			}
			for j := uint32(0); j < g.Nj; j++ {
				idx := i*g.Nj + j
				lons[idx] = lon
			}
		}
	}

	return lons
}

// Coordinates generates lat/lon coordinate pairs for all grid points.
//
// Returns two slices (latitudes and longitudes) in grid scan order.
// The slices have length equal to NumPoints().
//
// This is a convenience method that calls Latitudes() and Longitudes().
func (g *LatLonGrid) Coordinates() (latitudes, longitudes []float32) {
	return g.Latitudes(), g.Longitudes()
}
