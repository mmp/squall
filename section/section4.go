package section

import (
	"fmt"

	"github.com/mmp/mgrib2/internal"
	"github.com/mmp/mgrib2/product"
)

// Section4 represents the GRIB2 Product Definition Section (Section 4).
//
// This section describes what meteorological parameter is contained in the data,
// along with information about the generating process, forecast time, and
// vertical level.
type Section4 struct {
	Length                  uint32          // Total length of this section in bytes
	CoordinateValuesCount   uint16          // Number of coordinate values after template
	ProductDefinitionTemplate uint16        // Product definition template number (Table 4.0)
	Product                 product.Product // Parsed product (template-specific)
}

// ParseSection4 parses the GRIB2 Product Definition Section (Section 4).
//
// Section 4 structure (variable length, minimum 9 bytes + template):
//   Bytes 1-4:   Length of section (uint32)
//   Byte 5:      Section number (must be 4)
//   Bytes 6-7:   Number of coordinate values after template (uint16)
//   Bytes 8-9:   Product definition template number (Table 4.0)
//   Bytes 10-n:  Product definition (template-specific)
//
// Currently supported templates:
//   - 0: Analysis or forecast at a horizontal level or layer at a point in time
//   - 8: Average, accumulation, extreme values or other statistically processed values
//
// Returns an error if:
//   - The section is too short
//   - The section number is not 4
//   - The template number is not supported
func ParseSection4(data []byte) (*Section4, error) {
	if len(data) < 9 {
		return nil, fmt.Errorf("section 4 must be at least 9 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Read section length
	length, _ := r.Uint32()

	// Validate section length matches data
	if int(length) != len(data) {
		return nil, fmt.Errorf("section 4 length mismatch: header says %d bytes, have %d bytes", length, len(data))
	}

	// Read and validate section number
	sectionNum, _ := r.Uint8()
	if sectionNum != 4 {
		return nil, fmt.Errorf("expected section 4, got section %d", sectionNum)
	}

	// Read product definition metadata
	coordinateValuesCount, _ := r.Uint16()
	productDefinitionTemplateNumber, _ := r.Uint16()

	// Read template-specific data
	templateData, _ := r.Bytes(r.Remaining())

	// Parse product based on template number
	var parsedProduct product.Product
	var err error

	switch productDefinitionTemplateNumber {
	case 0:
		// Template 4.0: Analysis or forecast at a horizontal level or layer
		parsedProduct, err = product.ParseTemplate40(templateData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse product template 4.0: %w", err)
		}

	case 8:
		// Template 4.8: Average, accumulation, extreme values or statistically processed values
		parsedProduct, err = product.ParseTemplate48(templateData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse product template 4.8: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported product template: %d", productDefinitionTemplateNumber)
	}

	return &Section4{
		Length:                    length,
		CoordinateValuesCount:     coordinateValuesCount,
		ProductDefinitionTemplate: productDefinitionTemplateNumber,
		Product:                   parsedProduct,
	}, nil
}

// ProductDescription returns a human-readable description of the product.
func (s *Section4) ProductDescription() string {
	if s.Product != nil {
		return s.Product.String()
	}
	return fmt.Sprintf("Unknown product template %d", s.ProductDefinitionTemplate)
}
