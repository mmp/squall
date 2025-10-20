package section

import (
	"fmt"

	"github.com/mmp/mgrib2/data"
	"github.com/mmp/mgrib2/internal"
)

// Section5 represents the GRIB2 Data Representation Section (Section 5).
//
// This section describes how the data values are packed/compressed,
// including the packing method, number of bits per value, and scaling parameters.
type Section5 struct {
	Length                       uint32               // Total length of this section in bytes
	NumDataValues                uint32               // Number of data values
	DataRepresentationTemplate   uint16               // Data representation template number (Table 5.0)
	Representation               data.Representation  // Parsed representation (template-specific)
}

// ParseSection5 parses the GRIB2 Data Representation Section (Section 5).
//
// Section 5 structure (variable length, minimum 11 bytes + template):
//   Bytes 1-4:   Length of section (uint32)
//   Byte 5:      Section number (must be 5)
//   Bytes 6-9:   Number of data values (uint32)
//   Bytes 10-11: Data representation template number (Table 5.0)
//   Bytes 12-n:  Data representation (template-specific)
//
// Currently supported templates:
//   - 0: Simple packing (most common, ~80% of files)
//
// Returns an error if:
//   - The section is too short
//   - The section number is not 5
//   - The template number is not supported
func ParseSection5(sectionData []byte) (*Section5, error) {
	if len(sectionData) < 11 {
		return nil, fmt.Errorf("section 5 must be at least 11 bytes, got %d", len(sectionData))
	}

	r := internal.NewReader(sectionData)

	// Read section length
	length, _ := r.Uint32()

	// Validate section length matches data
	if int(length) != len(sectionData) {
		return nil, fmt.Errorf("section 5 length mismatch: header says %d bytes, have %d bytes", length, len(sectionData))
	}

	// Read and validate section number
	sectionNum, _ := r.Uint8()
	if sectionNum != 5 {
		return nil, fmt.Errorf("expected section 5, got section %d", sectionNum)
	}

	// Read data representation metadata
	numDataValues, _ := r.Uint32()
	dataRepresentationTemplateNumber, _ := r.Uint16()

	// Read template-specific data
	templateData, _ := r.Bytes(r.Remaining())

	// Parse representation based on template number
	var parsedRepresentation data.Representation
	var err error

	switch dataRepresentationTemplateNumber {
	case 0:
		// Template 5.0: Simple packing
		parsedRepresentation, err = data.ParseTemplate50(numDataValues, templateData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse data representation template 5.0: %w", err)
		}

	case 3:
		// Template 5.3: Complex packing with spatial differencing
		parsedRepresentation, err = data.ParseTemplate53(numDataValues, templateData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse data representation template 5.3: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported data representation template: %d", dataRepresentationTemplateNumber)
	}

	return &Section5{
		Length:                     length,
		NumDataValues:              numDataValues,
		DataRepresentationTemplate: dataRepresentationTemplateNumber,
		Representation:             parsedRepresentation,
	}, nil
}

// RepresentationDescription returns a human-readable description of the data representation.
func (s *Section5) RepresentationDescription() string {
	if s.Representation != nil {
		return s.Representation.String()
	}
	return fmt.Sprintf("Unknown data representation template %d", s.DataRepresentationTemplate)
}
