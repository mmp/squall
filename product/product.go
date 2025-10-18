// Package product provides product definition types and parsers for GRIB2.
package product

// Product represents a GRIB2 product definition.
// Different product templates implement this interface.
type Product interface {
	// TemplateNumber returns the product definition template number (Table 4.0).
	TemplateNumber() int

	// GetParameterCategory returns the parameter category code (Table 4.1).
	GetParameterCategory() uint8

	// GetParameterNumber returns the parameter number code (Table 4.2).
	GetParameterNumber() uint8

	// String returns a human-readable description of the product.
	String() string
}
