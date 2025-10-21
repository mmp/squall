// Package data provides data representation types and decoders for GRIB2.
package data

// Representation represents a GRIB2 data representation template.
// Different templates implement this interface to provide decoding capabilities.
type Representation interface {
	// TemplateNumber returns the data representation template number (Table 5.0).
	TemplateNumber() int

	// NumDataValues returns the number of data values to be unpacked.
	NumDataValues() uint32

	// BitsPerValue returns the number of bits used to pack each value.
	BitsPerValue() uint8

	// Decode unpacks the data from packed bytes and applies scaling.
	// The bitmap parameter is optional (nil means all points are valid).
	// Returns a slice of float32 values in grid order.
	Decode(packedData []byte, bitmap []bool) ([]float32, error)

	// String returns a human-readable description.
	String() string
}
