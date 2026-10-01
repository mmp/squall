package squall

import (
	"fmt"

	"github.com/mmp/squall/section"
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
// A message may hold more than one field; ParseMessage returns the first.
// Use ParseMessageFields to get all of them.
func ParseMessage(data []byte) (*Message, error) {
	fields, err := ParseMessageFields(data)
	if err != nil {
		return nil, err
	}
	return fields[0], nil
}

// ParseMessageFields parses all of the fields in a GRIB2 message, returning
// a Message for each.
//
// Most messages hold a single field, but sequences of Sections 2-7, 3-7, or
// 4-7 may be repeated to pack several fields into one message; NCEP does
// this, for example, for the U and V wind components in the NAM nests.
// Sections that are not repeated are shared by the fields that follow
// them. A field whose Section 6 has bitmap indicator 254 ("previously
// defined bitmap") is given the bitmap most recently defined in the
// message.
func ParseMessageFields(data []byte) ([]*Message, error) {
	if err := ValidateMessageStructure(data); err != nil {
		return nil, err
	}

	// Parse Section 0 (always 16 bytes)
	sec0, err := section.ParseSection0(data[:16])
	if err != nil {
		return nil, &ParseError{
			Section:    0,
			Offset:     0,
			Message:    "failed to parse Section 0",
			Underlying: err,
		}
	}
	offset := 16

	// Parse Section 1 (variable length)
	sec1, err := parseSectionAt(data, offset, 1)
	if err != nil {
		return nil, err
	}
	offset += int(sec1.(*section.Section1).Length)

	// The sections that apply to the next field, updated as they appear.
	cur := Message{
		Section0: sec0,
		Section1: sec1.(*section.Section1),
		RawData:  data,
	}
	var bitmap []bool // the most recently defined bitmap
	// Sections 4-6 must appear for each field.
	var have4, have5, have6 bool

	var fields []*Message
	end := len(data) - 4 // the "7777" end marker
	for offset < end {
		if offset+5 > end {
			return nil, &ParseError{
				Section: -1,
				Offset:  offset,
				Message: "truncated section header",
			}
		}
		num := data[offset+4]

		var length uint32
		switch num {
		case 2:
			sec2, err := parseSectionAt(data, offset, 2)
			if err != nil {
				return nil, err
			}
			cur.Section2 = sec2.(*section.Section2)
			length = cur.Section2.Length

		case 3:
			sec3, err := parseSectionAt(data, offset, 3)
			if err != nil {
				return nil, err
			}
			cur.Section3 = sec3.(*section.Section3)
			length = cur.Section3.Length

		case 4:
			sec4, err := parseSectionAt(data, offset, 4)
			if err != nil {
				return nil, err
			}
			cur.Section4 = sec4.(*section.Section4)
			length = cur.Section4.Length
			have4 = true

		case 5:
			sec5, err := parseSectionAt(data, offset, 5)
			if err != nil {
				return nil, err
			}
			cur.Section5 = sec5.(*section.Section5)
			length = cur.Section5.Length
			have5 = true

		case 6:
			// Section 6 needs the number of grid points from Section 3
			if cur.Section3 == nil {
				return nil, &ParseError{Section: 6, Offset: offset, Message: "Section 6 precedes Section 3"}
			}
			sec6Data := extractSectionData(data, offset, 6)
			if sec6Data == nil {
				return nil, &ParseError{
					Section: 6,
					Offset:  offset,
					Message: "failed to extract section 6 data",
				}
			}
			sec6, err := section.ParseSection6(sec6Data, cur.Section3.NumDataPoints)
			if err != nil {
				return nil, &ParseError{
					Section:    6,
					Offset:     offset,
					Message:    "failed to parse Section 6",
					Underlying: err,
				}
			}
			switch sec6.BitmapIndicator {
			case 0:
				bitmap = sec6.Bitmap
			case 254:
				if bitmap == nil {
					return nil, &ParseError{
						Section: 6,
						Offset:  offset,
						Message: "bitmap indicator 254 without a previously defined bitmap",
					}
				}
				sec6.Bitmap = bitmap
			}
			cur.Section6 = sec6
			length = sec6.Length
			have6 = true

		case 7:
			if cur.Section3 == nil || !have4 || !have5 || !have6 {
				return nil, &ParseError{
					Section: 7,
					Offset:  offset,
					Message: "Section 7 is not preceded by Sections 3, 4, 5, and 6",
				}
			}
			sec7, err := parseSectionAt(data, offset, 7)
			if err != nil {
				return nil, err
			}
			field := cur
			field.Section7 = sec7.(*section.Section7)
			fields = append(fields, &field)
			length = field.Section7.Length
			have4, have5, have6 = false, false, false

		default:
			return nil, &ParseError{
				Section: int(num),
				Offset:  offset,
				Message: fmt.Sprintf("unexpected section number %d", num),
			}
		}

		if length == 0 {
			return nil, &ParseError{Section: int(num), Offset: offset, Message: "zero-length section"}
		}
		offset += int(length)
	}

	if len(fields) == 0 {
		return nil, &ParseError{
			Section: 7,
			Offset:  offset,
			Message: "message has no data section",
		}
	}
	return fields, nil
}

// extractSectionData reads a section's length and extracts its data.
func extractSectionData(data []byte, offset int, _ uint8) []byte {
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
// Returns a slice of float32 values in grid scan order.
// Missing/undefined values are represented as 9.999e20.
//
// This method combines the data representation (Section 5), bitmap (Section 6),
// and packed data (Section 7) to produce the final decoded values.
func (m *Message) DecodeData() ([]float32, error) {
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
func (m *Message) Coordinates() (latitudes, longitudes []float32, err error) {
	if m.Section3 == nil || m.Section3.Grid == nil {
		return nil, nil, fmt.Errorf("message has no grid definition (Section 3)")
	}

	// Check if it's a LatLonGrid
	switch grid := m.Section3.Grid.(type) {
	case interface {
		Coordinates() ([]float32, []float32)
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
