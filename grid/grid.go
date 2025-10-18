// Package grid provides grid definition types and parsers for GRIB2.
package grid

// Grid represents a GRIB2 grid definition.
// Different grid templates implement this interface.
type Grid interface {
	// TemplateNumber returns the grid definition template number (Table 3.1).
	TemplateNumber() int

	// NumPoints returns the total number of grid points.
	NumPoints() int

	// String returns a human-readable description of the grid.
	String() string
}
