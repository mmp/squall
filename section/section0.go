// Package section provides parsers for GRIB2 message sections.
package section

import (
	"fmt"

	"github.com/mmp/mgrib2/internal"
)

// Section0 represents the GRIB2 Indicator Section (Section 0).
//
// This is the first 16 bytes of every GRIB2 message and contains:
//   - Magic number "GRIB" to identify the file format
//   - Discipline code indicating the type of data (meteorological, hydrological, etc.)
//   - Edition number (must be 2 for GRIB2)
//   - Total message length in bytes
//
// Section 0 is always exactly 16 bytes.
type Section0 struct {
	Discipline    uint8  // Discipline (Table 0.0: 0=Meteorological, 1=Hydrological, etc.)
	Edition       uint8  // GRIB edition number (must be 2)
	MessageLength uint64 // Total length of GRIB message in bytes (including this section)
}

// ParseSection0 parses the GRIB2 Indicator Section (Section 0).
//
// Section 0 structure (16 bytes total):
//
//	Bytes 1-4:   "GRIB" magic number
//	Bytes 5-6:   Reserved (must be 0x0000)
//	Byte 7:      Discipline (Table 0.0)
//	Byte 8:      GRIB edition number (must be 2)
//	Bytes 9-16:  Total message length (uint64)
//
// Returns an error if:
//   - The data is less than 16 bytes
//   - The magic number is not "GRIB"
//   - The edition number is not 2
//   - The reserved bytes are not zero (warning only in this implementation)
func ParseSection0(data []byte) (*Section0, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("section 0 must be exactly 16 bytes, got %d", len(data))
	}

	// Check magic number "GRIB"
	if data[0] != 'G' || data[1] != 'R' || data[2] != 'I' || data[3] != 'B' {
		return nil, fmt.Errorf("invalid GRIB magic number: got %q, expected \"GRIB\"",
			string(data[0:4]))
	}

	r := internal.NewReader(data)

	// Skip "GRIB" magic (already validated)
	r.Skip(4)

	// Read and validate reserved bytes
	reserved, _ := r.Uint16()
	if reserved != 0 {
		// WMO spec says this should be 0, but we'll just warn
		// Some implementations might use this for other purposes
		// Don't fail, but could log if we had logging
	}

	// Read discipline
	discipline, _ := r.Uint8()

	// Read and validate edition
	edition, _ := r.Uint8()
	if edition != 2 {
		return nil, fmt.Errorf("unsupported GRIB edition: got %d, expected 2 (this is a GRIB2 parser)", edition)
	}

	// Read message length
	messageLength, _ := r.Uint64()

	// Validate message length is reasonable
	if messageLength < 16 {
		return nil, fmt.Errorf("invalid message length %d (must be at least 16 bytes)", messageLength)
	}

	return &Section0{
		Discipline:    discipline,
		Edition:       edition,
		MessageLength: messageLength,
	}, nil
}

// DisciplineName returns the human-readable name for the discipline code.
// Returns "Unknown" if the discipline code is not recognized.
func (s *Section0) DisciplineName() string {
	return GetDisciplineName(s.Discipline)
}

// GetDisciplineName returns the human-readable name for a discipline code.
// This is based on WMO Table 0.0.
func GetDisciplineName(discipline uint8) string {
	// WMO Code Table 0.0: Discipline of processed data
	switch discipline {
	case 0:
		return "Meteorological products"
	case 1:
		return "Hydrological products"
	case 2:
		return "Land surface products"
	case 3:
		return "Space products"
	case 4:
		return "Space weather products"
	case 10:
		return "Oceanographic products"
	case 20:
		return "Health and socioeconomic impacts"
	default:
		if discipline >= 192 {
			return fmt.Sprintf("Reserved for local use (%d)", discipline)
		}
		return fmt.Sprintf("Unknown discipline (%d)", discipline)
	}
}
