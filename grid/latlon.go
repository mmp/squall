package grid

import (
	"fmt"
	"math"

	"github.com/mmp/squall/internal"
)

// LatLonGrid represents a GRIB2 Latitude/Longitude grid (Template 3.0).
//
// This is the most common grid type, consisting of a regular grid with
// constant spacing in latitude and longitude.
//
// Angles are in units of Unit() degrees: normally microdegrees, unless the
// basic angle and subdivisions specify otherwise.
type LatLonGrid struct {
	Ni           uint32 // Number of points along a parallel (longitude)
	Nj           uint32 // Number of points along a meridian (latitude)
	BasicAngle   uint32 // Basic angle of the initial production domain (0 for the default units)
	Subdivisions uint32 // Subdivisions of the basic angle used to define angles
	La1          int32  // Latitude of first grid point
	Lo1          int32  // Longitude of first grid point
	ResFlags     uint8  // Resolution and component flags
	La2          int32  // Latitude of last grid point
	Lo2          int32  // Longitude of last grid point
	Di           uint32 // i direction increment
	Dj           uint32 // j direction increment
	ScanningMode uint8  // Scanning mode (Table 3.4)
}

// ParseLatLonGrid parses a Lat/Lon grid from template data (Template 3.0).
//
// The template data (octets 15-72 of Section 3) should be 58 bytes.
func ParseLatLonGrid(data []byte) (*LatLonGrid, error) {
	if len(data) < 58 {
		return nil, fmt.Errorf("template 3.0 requires at least 58 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Skip shape of earth (1 byte) and related parameters (15 bytes)
	// We'll implement proper earth shape handling in a future phase
	_ = r.Skip(16)

	// Read grid dimensions
	ni, _ := r.Uint32()
	nj, _ := r.Uint32()

	basicAngle, _ := r.Uint32()
	subdivisions, _ := r.Uint32()

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
		BasicAngle:   basicAngle,
		Subdivisions: subdivisions,
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

// Unit returns the size in degrees of the units of the grid's angles:
// 10^-6 unless the basic angle and its subdivisions are given.
func (g *LatLonGrid) Unit() float64 {
	if g.BasicAngle == 0 || g.BasicAngle == 0xFFFFFFFF || g.Subdivisions == 0 || g.Subdivisions == 0xFFFFFFFF {
		return 1e-6
	}
	return float64(g.BasicAngle) / float64(g.Subdivisions)
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
	lat1, lon1 := g.FirstGridPoint()
	lat2, lon2 := g.LastGridPoint()
	return fmt.Sprintf("Lat/Lon grid: %d x %d points (%.3f°, %.3f°) to (%.3f°, %.3f°)",
		g.Ni, g.Nj, lat1, lon1, lat2, lon2)
}

// FirstGridPoint returns the latitude and longitude of the first grid point in degrees.
func (g *LatLonGrid) FirstGridPoint() (lat, lon float64) {
	return g.degrees(float64(g.La1)), g.degrees(float64(g.Lo1))
}

// LastGridPoint returns the latitude and longitude of the last grid point in degrees.
func (g *LatLonGrid) LastGridPoint() (lat, lon float64) {
	return g.degrees(float64(g.La2)), g.degrees(float64(g.Lo2))
}

// Increment returns the i and j direction increments in degrees.
func (g *LatLonGrid) Increment() (di, dj float64) {
	return g.degrees(float64(g.Di)), g.degrees(float64(g.Dj))
}

// degrees converts an angle in the grid's units to degrees. The explicit
// conversion rounds the product; otherwise the compiler may fuse it with a
// later addition, so that, for example, a longitude computed as
// lon1 - 2*dlon is -9e-17 rather than 0.
func (g *LatLonGrid) degrees(v float64) float64 {
	return float64(v * g.Unit()) //nolint:unconvert // see above
}

// ScanningFlags returns the scanning mode flags as individual booleans.
//
// Returns:
//   - iNegative: true if points scan in -i direction (east to west)
//   - jPositive: true if points scan in +j direction (south to north)
//   - consecutive: true if adjacent points in i direction are consecutive
func (g *LatLonGrid) ScanningFlags() (iNegative, jPositive, consecutive bool) {
	iNegative = (g.ScanningMode & 0x80) != 0   // Bit 0
	jPositive = (g.ScanningMode & 0x40) != 0   // Bit 1
	consecutive = (g.ScanningMode & 0x20) == 0 // Bit 2 (0 = consecutive)
	return
}

// Latitudes returns the latitude of each grid point, in degrees and in
// grid scan order.
func (g *LatLonGrid) Latitudes() []float32 {
	lats, _ := g.Coordinates()
	return lats
}

// Longitudes returns the longitude of each grid point, in degrees in the
// range [0, 360) and in grid scan order.
func (g *LatLonGrid) Longitudes() []float32 {
	_, lons := g.Coordinates()
	return lons
}

// Coordinates returns the latitude and longitude of each grid point, in
// degrees and in grid scan order. Longitudes are in the range [0, 360).
//
// As in wgrib2, the spacing between points is computed from the first and
// last grid points, rather than taken from Di and Dj, which may have been
// rounded.
func (g *LatLonGrid) Coordinates() (latitudes, longitudes []float32) {
	iNegative, jPositive, consecutive := g.ScanningFlags()
	alternating := g.ScanningMode&0x10 != 0 // boustrophedonic rows

	lat1, lon1 := g.FirstGridPoint()
	lat2, lon2 := g.LastGridPoint()

	var dlat float64
	if g.Nj > 1 {
		dlat = math.Abs(lat2-lat1) / float64(g.Nj-1)
	}

	// Find the west and east edges to get the longitude spacing.
	w, e := lon1, lon2
	if iNegative {
		w, e = lon2, lon1
	}
	if e <= w {
		e += 360
	}
	if e-w > 360 {
		e -= 360
	}
	var dlon float64
	if g.Ni > 1 {
		dlon = math.Abs((e - w) / float64(g.Ni-1))
	}

	n := int(g.Ni) * int(g.Nj)
	latitudes = make([]float32, n)
	longitudes = make([]float32, n)
	for j := range int(g.Nj) {
		y := float64(j)
		if !jPositive {
			y = -y
		}
		lat := float32(lat1 + y*dlat)
		for i := range int(g.Ni) {
			x := float64(i)
			if alternating && j%2 == 1 {
				x = float64(int(g.Ni) - 1 - i)
			}
			if iNegative {
				x = -x
			}
			lon := lon1 + x*dlon
			if lon >= 360 {
				lon -= 360
			}
			if lon < 0 {
				lon += 360
			}

			idx := j*int(g.Ni) + i
			if !consecutive {
				idx = i*int(g.Nj) + j
			}
			latitudes[idx] = lat
			longitudes[idx] = float32(lon)
		}
	}
	return latitudes, longitudes
}
