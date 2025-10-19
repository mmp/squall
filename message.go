package mgrib2

import (
	"fmt"

	"github.com/mmp/mgrib2/section"
)

// Message represents a complete parsed GRIB2 message.
//
// A GRIB2 message contains all the information needed to describe and
// decode a single meteorological field, including metadata, grid definition,
// product description, and the packed data values.
type Message struct {
	// Section0 contains the indicator section with discipline and message length
	Section0 *section.Section0

	// Section1 contains identification information (center, time, etc.)
	Section1 *section.Section1

	// Section2 contains local use data (optional, may be nil)
	Section2 *section.Section2

	// Section3 contains the grid definition
	Section3 *section.Section3

	// Section4 contains the product definition
	Section4 *section.Section4

	// Section5 contains the data representation template
	Section5 *section.Section5

	// Section6 contains the bitmap (optional, may be nil if all points valid)
	Section6 *section.Section6

	// Section7 contains the packed data
	Section7 *section.Section7

	// RawData is the original message bytes (for debugging/analysis)
	RawData []byte
}

// ParseMessage parses a complete GRIB2 message from raw bytes.
//
// The input data should contain a single complete GRIB2 message starting
// with "GRIB" and ending with "7777".
//
// This function parses all 8 sections of the message:
//   - Section 0: Indicator (discipline, message length)
//   - Section 1: Identification (center, reference time, etc.)
//   - Section 2: Local use (optional)
//   - Section 3: Grid definition
//   - Section 4: Product definition
//   - Section 5: Data representation
//   - Section 6: Bitmap
//   - Section 7: Data
//   - Section 8: End marker "7777"
//
// Note: Currently assumes one field per message. Multi-field messages
// (where sections 3-7 repeat) are not yet supported.
func ParseMessage(data []byte) (*Message, error) {
	if err := ValidateMessageStructure(data); err != nil {
		return nil, err
	}

	msg := &Message{
		RawData: data,
	}

	offset := 0

	// Parse Section 0 (always 16 bytes)
	sec0, err := section.ParseSection0(data[offset : offset+16])
	if err != nil {
		return nil, &ParseError{
			Section:    0,
			Offset:     offset,
			Message:    "failed to parse Section 0",
			Underlying: err,
		}
	}
	msg.Section0 = sec0
	offset += 16

	// Parse Section 1 (variable length)
	sec1, err := parseSectionAt(data, offset, 1)
	if err != nil {
		return nil, err
	}
	msg.Section1 = sec1.(*section.Section1)
	offset += int(sec1.(*section.Section1).Length)

	// Check for optional Section 2
	if offset < len(data)-4 && data[offset+4] == 2 {
		sec2, err := parseSectionAt(data, offset, 2)
		if err != nil {
			return nil, err
		}
		msg.Section2 = sec2.(*section.Section2)
		offset += int(sec2.(*section.Section2).Length)
	}

	// Parse Section 3 (Grid Definition)
	sec3, err := parseSectionAt(data, offset, 3)
	if err != nil {
		return nil, err
	}
	msg.Section3 = sec3.(*section.Section3)
	offset += int(sec3.(*section.Section3).Length)

	// Parse Section 4 (Product Definition)
	sec4, err := parseSectionAt(data, offset, 4)
	if err != nil {
		return nil, err
	}
	msg.Section4 = sec4.(*section.Section4)
	offset += int(sec4.(*section.Section4).Length)

	// Parse Section 5 (Data Representation)
	sec5, err := parseSectionAt(data, offset, 5)
	if err != nil {
		return nil, err
	}
	msg.Section5 = sec5.(*section.Section5)
	offset += int(sec5.(*section.Section5).Length)

	// Parse Section 6 (Bitmap)
	// Section 6 needs the number of grid points from Section 3
	numGridPoints := uint32(msg.Section3.NumDataPoints)
	sec6Data := extractSectionData(data, offset, 6)
	if sec6Data == nil {
		return nil, &ParseError{
			Section: 6,
			Offset:  offset,
			Message: "failed to extract section 6 data",
		}
	}
	sec6, err := section.ParseSection6(sec6Data, numGridPoints)
	if err != nil {
		return nil, &ParseError{
			Section:    6,
			Offset:     offset,
			Message:    "failed to parse Section 6",
			Underlying: err,
		}
	}
	msg.Section6 = sec6
	offset += int(sec6.Length)

	// Parse Section 7 (Data)
	sec7, err := parseSectionAt(data, offset, 7)
	if err != nil {
		return nil, err
	}
	msg.Section7 = sec7.(*section.Section7)
	offset += int(sec7.(*section.Section7).Length)

	// The remaining 4 bytes should be the end marker "7777"
	// (already validated by ValidateMessageStructure)

	return msg, nil
}

// extractSectionData reads a section's length and extracts its data.
func extractSectionData(data []byte, offset int, expectedSection uint8) []byte {
	if offset+5 > len(data) {
		return nil
	}

	// Read section length (first 4 bytes)
	sectionLength := uint32(data[offset])<<24 | uint32(data[offset+1])<<16 |
		uint32(data[offset+2])<<8 | uint32(data[offset+3])

	// Validate we have enough data
	if offset+int(sectionLength) > len(data) {
		return nil
	}

	return data[offset : offset+int(sectionLength)]
}

// parseSectionAt reads a section length and parses the appropriate section type.
func parseSectionAt(data []byte, offset int, expectedSection uint8) (interface{}, error) {
	sectionData := extractSectionData(data, offset, expectedSection)
	if sectionData == nil {
		return nil, &ParseError{
			Section: int(expectedSection),
			Offset:  offset,
			Message: fmt.Sprintf("failed to extract section %d data", expectedSection),
		}
	}

	// Parse based on section type
	switch expectedSection {
	case 1:
		return section.ParseSection1(sectionData)
	case 2:
		return section.ParseSection2(sectionData)
	case 3:
		return section.ParseSection3(sectionData)
	case 4:
		return section.ParseSection4(sectionData)
	case 5:
		return section.ParseSection5(sectionData)
	case 7:
		return section.ParseSection7(sectionData)
	default:
		return nil, &ParseError{
			Section: int(expectedSection),
			Offset:  offset,
			Message: fmt.Sprintf("unsupported section number: %d", expectedSection),
		}
	}
}

// DecodeData decodes the data values from this message.
//
// Returns a slice of float64 values in grid scan order.
// Missing/undefined values are represented as 9.999e20.
//
// This method combines the data representation (Section 5), bitmap (Section 6),
// and packed data (Section 7) to produce the final decoded values.
func (m *Message) DecodeData() ([]float64, error) {
	if m.Section5 == nil || m.Section5.Representation == nil {
		return nil, fmt.Errorf("message has no data representation (Section 5)")
	}

	if m.Section7 == nil {
		return nil, fmt.Errorf("message has no data section (Section 7)")
	}

	// Get bitmap if present
	var bitmap []bool
	if m.Section6 != nil && m.Section6.HasBitmap() {
		bitmap = m.Section6.Bitmap
	}

	// Decode using the representation template
	values, err := m.Section5.Representation.Decode(m.Section7.Data, bitmap)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	return values, nil
}

// Coordinates returns the lat/lon coordinates for this message's grid.
//
// Returns two slices (latitudes and longitudes) in grid scan order,
// matching the order of values returned by DecodeData().
//
// Currently only supports LatLonGrid (Template 3.0). Returns an error
// for other grid types.
func (m *Message) Coordinates() (latitudes, longitudes []float64, err error) {
	if m.Section3 == nil || m.Section3.Grid == nil {
		return nil, nil, fmt.Errorf("message has no grid definition (Section 3)")
	}

	// Check if it's a LatLonGrid
	switch grid := m.Section3.Grid.(type) {
	case interface {
		Coordinates() ([]float64, []float64)
	}:
		lats, lons := grid.Coordinates()
		return lats, lons, nil
	default:
		return nil, nil, fmt.Errorf("grid type %T does not support coordinate generation", m.Section3.Grid)
	}
}

// String returns a human-readable summary of the message.
func (m *Message) String() string {
	if m.Section0 == nil {
		return "Invalid GRIB2 message"
	}

	discipline := "Unknown"
	if m.Section0 != nil {
		discipline = m.Section0.DisciplineName()
	}

	grid := "Unknown"
	if m.Section3 != nil && m.Section3.Grid != nil {
		grid = m.Section3.Grid.String()
	}

	product := "Unknown"
	if m.Section4 != nil && m.Section4.Product != nil {
		product = m.Section4.Product.String()
	}

	return fmt.Sprintf("GRIB2 Message: Discipline=%s, Grid=%s, Product=%s",
		discipline, grid, product)
}
