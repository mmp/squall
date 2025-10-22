package section

import (
	"fmt"

	"github.com/mmp/squall/internal"
)

// Section6 represents the GRIB2 Bit Map Section (Section 6).
//
// This section indicates which grid points have valid data values.
// Some grid points may be undefined/missing (e.g., land points in ocean data).
type Section6 struct {
	Length          uint32 // Total length of this section in bytes
	BitmapIndicator uint8  // Bitmap indicator (Table 6.0)
	Bitmap          []bool // Bitmap: true = data present, false = missing (nil if not applicable)
}

// ParseSection6 parses the GRIB2 Bit Map Section (Section 6).
//
// Section 6 structure (variable length, minimum 6 bytes):
//
//	Bytes 1-4: Length of section (uint32)
//	Byte 5:    Section number (must be 6)
//	Byte 6:    Bit-map indicator (Table 6.0)
//	Bytes 7-n: Bit map (if indicator = 0)
//
// Bitmap Indicator values (Table 6.0):
//
//	0   = Bitmap applies and is specified in this section
//	254 = Previously defined bitmap applies (not currently supported)
//	255 = Bitmap does not apply - all grid points are valid
//
// When indicator = 0, the bitmap contains one bit per grid point:
//
//	1 = data value is present
//	0 = data value is absent/missing
//
// The numGridPoints parameter is required when indicator = 0 to determine
// how many bits to read from the bitmap.
//
// Returns an error if:
//   - The section is too short
//   - The section number is not 6
//   - The bitmap indicator is not supported (currently only 0 and 255)
func ParseSection6(data []byte, numGridPoints uint32) (*Section6, error) {
	if len(data) < 6 {
		return nil, fmt.Errorf("section 6 must be at least 6 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Read section length
	length, _ := r.Uint32()

	// Validate section length matches data
	if int(length) != len(data) {
		return nil, fmt.Errorf("section 6 length mismatch: header says %d bytes, have %d bytes", length, len(data))
	}

	// Read and validate section number
	sectionNum, _ := r.Uint8()
	if sectionNum != 6 {
		return nil, fmt.Errorf("expected section 6, got section %d", sectionNum)
	}

	// Read bitmap indicator
	bitmapIndicator, _ := r.Uint8()

	var bitmap []bool

	switch bitmapIndicator {
	case 0:
		// Bitmap is specified in this section
		bitmapData, _ := r.Bytes(r.Remaining())
		var err error
		bitmap, err = parseBitmap(bitmapData, numGridPoints)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bitmap: %w", err)
		}

	case 254:
		// Previously defined bitmap applies
		return nil, fmt.Errorf("bitmap indicator 254 (previously defined bitmap) not yet supported")

	case 255:
		// Bitmap does not apply - all grid points are valid
		bitmap = nil

	default:
		return nil, fmt.Errorf("unsupported bitmap indicator: %d", bitmapIndicator)
	}

	return &Section6{
		Length:          length,
		BitmapIndicator: bitmapIndicator,
		Bitmap:          bitmap,
	}, nil
}

// parseBitmap extracts a bitmap from packed bit data.
//
// The bitmap is packed into bytes with 8 bits per byte.
// Bit value 1 = data present, 0 = data missing.
// Bits are read in order (most significant bit first).
func parseBitmap(data []byte, numGridPoints uint32) ([]bool, error) {
	// Calculate expected number of bytes
	expectedBytes := (numGridPoints + 7) / 8
	if uint32(len(data)) < expectedBytes {
		return nil, fmt.Errorf("bitmap data too short: need %d bytes for %d grid points, got %d",
			expectedBytes, numGridPoints, len(data))
	}

	bitmap := make([]bool, numGridPoints)
	bitIdx := uint32(0)

	for byteIdx := 0; byteIdx < len(data) && bitIdx < numGridPoints; byteIdx++ {
		b := data[byteIdx]

		// Read bits from most significant to least significant
		for bit := 7; bit >= 0 && bitIdx < numGridPoints; bit-- {
			bitmap[bitIdx] = (b & (1 << uint(bit))) != 0
			bitIdx++
		}
	}

	return bitmap, nil
}

// HasBitmap returns true if this section contains a bitmap.
func (s *Section6) HasBitmap() bool {
	return s.Bitmap != nil
}

// CountValidPoints returns the number of grid points with valid data.
// If there's no bitmap, returns the total number of grid points.
func (s *Section6) CountValidPoints() uint32 {
	if s.Bitmap == nil {
		return 0 // Unknown - caller must track
	}

	count := uint32(0)
	for _, valid := range s.Bitmap {
		if valid {
			count++
		}
	}
	return count
}

// String returns a human-readable description.
func (s *Section6) String() string {
	switch s.BitmapIndicator {
	case 0:
		validPoints := s.CountValidPoints()
		totalPoints := uint32(len(s.Bitmap))
		return fmt.Sprintf("Bitmap: %d/%d valid points", validPoints, totalPoints)
	case 254:
		return "Bitmap: Previously defined"
	case 255:
		return "Bitmap: Not applicable (all points valid)"
	default:
		return fmt.Sprintf("Bitmap: Unknown indicator %d", s.BitmapIndicator)
	}
}
