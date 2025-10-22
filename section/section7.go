package section

import (
	"fmt"

	"github.com/mmp/mgrib2/internal"
)

// Section7 represents the GRIB2 Data Section (Section 7).
//
// This section contains the actual packed data values.
// The format of the data depends on the data representation template
// specified in Section 5.
type Section7 struct {
	Length uint32 // Total length of this section in bytes
	Data   []byte // Packed data (format depends on Section 5)
}

// ParseSection7 parses the GRIB2 Data Section (Section 7).
//
// Section 7 structure (variable length, minimum 5 bytes):
//
//	Bytes 1-4: Length of section (uint32)
//	Byte 5:    Section number (must be 7)
//	Bytes 6-n: Packed data (format depends on Section 5)
//
// The packed data is stored as-is and must be decoded using the
// data representation template from Section 5.
//
// Returns an error if:
//   - The section is too short
//   - The section number is not 7
func ParseSection7(data []byte) (*Section7, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("section 7 must be at least 5 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Read section length
	length, _ := r.Uint32()

	// Validate section length matches data
	if int(length) != len(data) {
		return nil, fmt.Errorf("section 7 length mismatch: header says %d bytes, have %d bytes", length, len(data))
	}

	// Read and validate section number
	sectionNum, _ := r.Uint8()
	if sectionNum != 7 {
		return nil, fmt.Errorf("expected section 7, got section %d", sectionNum)
	}

	// Read packed data (everything remaining)
	packedData, _ := r.Bytes(r.Remaining())

	return &Section7{
		Length: length,
		Data:   packedData,
	}, nil
}

// String returns a human-readable description.
func (s *Section7) String() string {
	return fmt.Sprintf("Data Section: %d bytes of packed data", len(s.Data))
}
