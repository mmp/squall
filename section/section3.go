package section

import (
	"fmt"

	"github.com/mmp/squall/grid"
	"github.com/mmp/squall/internal"
)

// Section3 represents the GRIB2 Grid Definition Section (Section 3).
//
// This section describes the geographic grid on which the data is defined.
type Section3 struct {
	Length                uint32    // Total length of this section in bytes
	Source                uint8     // Source of grid definition (Table 3.0)
	NumDataPoints         uint32    // Number of data points
	NumOctetsOptionalList uint8     // Number of octets for optional list
	InterpretOptionalList uint8     // Interpretation of optional list
	TemplateNumber        uint16    // Grid definition template number (Table 3.1)
	Grid                  grid.Grid // Parsed grid (template-specific)
}

// ParseSection3 parses the GRIB2 Grid Definition Section (Section 3).
//
// Section 3 structure (variable length, minimum 14 bytes + template):
//
//	Bytes 1-4:   Length of section (uint32)
//	Byte 5:      Section number (must be 3)
//	Byte 6:      Source of grid definition (Table 3.0)
//	Bytes 7-10:  Number of data points (uint32)
//	Byte 11:     Number of octets for optional list
//	Byte 12:     Interpretation of optional list
//	Bytes 13-14: Grid definition template number (Table 3.1)
//	Bytes 15-n:  Grid definition (template-specific)
//
// Currently supported templates:
//   - 0: Latitude/Longitude (equidistant cylindrical)
//   - 30: Lambert Conformal
//
// Returns an error if:
//   - The section is too short
//   - The section number is not 3
//   - The template number is not supported
func ParseSection3(data []byte) (*Section3, error) {
	if len(data) < 14 {
		return nil, fmt.Errorf("section 3 must be at least 14 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Read section length
	length, _ := r.Uint32()

	// Validate section length matches data
	if int(length) != len(data) {
		return nil, fmt.Errorf("section 3 length mismatch: header says %d bytes, have %d bytes", length, len(data))
	}

	// Read and validate section number
	sectionNum, _ := r.Uint8()
	if sectionNum != 3 {
		return nil, fmt.Errorf("expected section 3, got section %d", sectionNum)
	}

	// Read grid definition metadata
	source, _ := r.Uint8()
	numDataPoints, _ := r.Uint32()
	numOctetsOptionalList, _ := r.Uint8()
	interpretOptionalList, _ := r.Uint8()
	templateNumber, _ := r.Uint16()

	// Read template-specific data
	templateData, _ := r.Bytes(r.Remaining())

	// Parse grid based on template number
	var parsedGrid grid.Grid
	var err error

	switch templateNumber {
	case 0:
		// Template 3.0: Latitude/Longitude
		parsedGrid, err = grid.ParseLatLonGrid(templateData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse lat/lon grid: %w", err)
		}

	case 30:
		// Template 3.30: Lambert Conformal
		parsedGrid, err = grid.ParseLambertConformalGrid(templateData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Lambert Conformal grid: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported grid template: %d", templateNumber)
	}

	return &Section3{
		Length:                length,
		Source:                source,
		NumDataPoints:         numDataPoints,
		NumOctetsOptionalList: numOctetsOptionalList,
		InterpretOptionalList: interpretOptionalList,
		TemplateNumber:        templateNumber,
		Grid:                  parsedGrid,
	}, nil
}

// GridDescription returns a human-readable description of the grid.
func (s *Section3) GridDescription() string {
	if s.Grid != nil {
		return s.Grid.String()
	}
	return fmt.Sprintf("Unknown grid template %d", s.TemplateNumber)
}
