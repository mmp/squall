package section

import (
	"fmt"

	"github.com/mmp/squall/internal"
)

// Section2 represents the GRIB2 Local Use Section (Section 2).
//
// This section is optional and reserved for local or experimental use.
// The structure is center-specific and we store it as opaque bytes.
// If the section is not present in the message, it should be nil.
type Section2 struct {
	Length uint32 // Total length of this section in bytes
	Data   []byte // Local use data (opaque, center-specific)
}

// ParseSection2 parses the GRIB2 Local Use Section (Section 2).
//
// Section 2 structure (variable length, minimum 5 bytes):
//
//	Bytes 1-4:   Length of section (uint32)
//	Byte 5:      Section number (must be 2)
//	Bytes 6-n:   Local use data (opaque bytes)
//
// The data is stored as-is without interpretation. Applications that
// need to use Section 2 data should implement their own parsing based
// on the originating center.
//
// Returns an error if:
//   - The section is shorter than 5 bytes
//   - The section number is not 2
//   - The length field doesn't match the data size
func ParseSection2(data []byte) (*Section2, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("section 2 must be at least 5 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Read section length
	length, _ := r.Uint32()

	// Validate section length matches data
	if int(length) != len(data) {
		return nil, fmt.Errorf("section 2 length mismatch: header says %d bytes, have %d bytes", length, len(data))
	}

	// Read and validate section number
	sectionNum, _ := r.Uint8()
	if sectionNum != 2 {
		return nil, fmt.Errorf("expected section 2, got section %d", sectionNum)
	}

	// Read remaining bytes as opaque local use data
	localData, _ := r.Bytes(len(data) - 5)

	return &Section2{
		Length: length,
		Data:   localData,
	}, nil
}

// IsEmpty returns true if the section contains no local use data.
func (s *Section2) IsEmpty() bool {
	return len(s.Data) == 0
}
