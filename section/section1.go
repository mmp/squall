package section

import (
	"fmt"
	"time"

	"github.com/mmp/squall/internal"
	"github.com/mmp/squall/tables"
)

// Section1 represents the GRIB2 Identification Section (Section 1).
//
// This section contains metadata about the message origin, reference time,
// and type of data. It is variable length but has a minimum of 21 bytes.
type Section1 struct {
	Length                uint32    // Total length of this section in bytes
	OriginatingCenter     uint16    // Originating/generating center (Table Common C-1)
	OriginatingSubcenter  uint16    // Originating/generating sub-center
	MasterTablesVersion   uint8     // GRIB master tables version number
	LocalTablesVersion    uint8     // GRIB local tables version number
	SignificanceOfRefTime uint8     // Significance of reference time (Table 1.2)
	ReferenceTime         time.Time // Reference time (year, month, day, hour, minute, second)
	ProductionStatus      uint8     // Production status of data (Table 1.3)
	TypeOfData            uint8     // Type of processed data (Table 1.4)
}

// ParseSection1 parses the GRIB2 Identification Section (Section 1).
//
// Section 1 structure (minimum 21 bytes):
//
//	Bytes 1-4:   Length of section (uint32)
//	Byte 5:      Section number (must be 1)
//	Bytes 6-7:   Originating center (Table C-1)
//	Bytes 8-9:   Originating sub-center
//	Byte 10:     Master tables version
//	Byte 11:     Local tables version
//	Byte 12:     Significance of reference time (Table 1.2)
//	Bytes 13-14: Year (4 digits)
//	Byte 15:     Month
//	Byte 16:     Day
//	Byte 17:     Hour
//	Byte 18:     Minute
//	Byte 19:     Second
//	Byte 20:     Production status (Table 1.3)
//	Byte 21:     Type of processed data (Table 1.4)
//
// Returns an error if:
//   - The section is shorter than 21 bytes
//   - The section number is not 1
//   - The date/time values are invalid
func ParseSection1(data []byte) (*Section1, error) {
	if len(data) < 21 {
		return nil, fmt.Errorf("section 1 must be at least 21 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Read section length
	length, _ := r.Uint32()

	// Validate section length matches data
	if int(length) != len(data) {
		return nil, fmt.Errorf("section 1 length mismatch: header says %d bytes, have %d bytes", length, len(data))
	}

	// Read and validate section number
	sectionNum, _ := r.Uint8()
	if sectionNum != 1 {
		return nil, fmt.Errorf("expected section 1, got section %d", sectionNum)
	}

	// Read center identification
	originatingCenter, _ := r.Uint16()
	originatingSubcenter, _ := r.Uint16()

	// Read table versions
	masterTablesVersion, _ := r.Uint8()
	localTablesVersion, _ := r.Uint8()

	// Read time significance
	significanceOfRefTime, _ := r.Uint8()

	// Read reference time components
	year, _ := r.Uint16()
	month, _ := r.Uint8()
	day, _ := r.Uint8()
	hour, _ := r.Uint8()
	minute, _ := r.Uint8()
	second, _ := r.Uint8()

	// Validate and construct time
	// Note: GRIB2 times are in UTC
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("invalid month: %d (must be 1-12)", month)
	}
	if day < 1 || day > 31 {
		return nil, fmt.Errorf("invalid day: %d (must be 1-31)", day)
	}
	if hour > 23 {
		return nil, fmt.Errorf("invalid hour: %d (must be 0-23)", hour)
	}
	if minute > 59 {
		return nil, fmt.Errorf("invalid minute: %d (must be 0-59)", minute)
	}
	if second > 59 {
		return nil, fmt.Errorf("invalid second: %d (must be 0-59)", second)
	}

	refTime := time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second), 0, time.UTC)

	// Read production status and data type
	productionStatus, _ := r.Uint8()
	typeOfData, _ := r.Uint8()

	return &Section1{
		Length:                length,
		OriginatingCenter:     originatingCenter,
		OriginatingSubcenter:  originatingSubcenter,
		MasterTablesVersion:   masterTablesVersion,
		LocalTablesVersion:    localTablesVersion,
		SignificanceOfRefTime: significanceOfRefTime,
		ReferenceTime:         refTime,
		ProductionStatus:      productionStatus,
		TypeOfData:            typeOfData,
	}, nil
}

// CenterName returns the human-readable name of the originating center.
func (s *Section1) CenterName() string {
	return tables.GetCenterName(int(s.OriginatingCenter))
}

// TimeSignificanceName returns the human-readable name of the reference time significance.
func (s *Section1) TimeSignificanceName() string {
	return tables.GetTimeSignificanceName(int(s.SignificanceOfRefTime))
}

// ProductionStatusName returns the human-readable name of the production status.
func (s *Section1) ProductionStatusName() string {
	return tables.GetProductionStatusName(int(s.ProductionStatus))
}

// DataTypeName returns the human-readable name of the data type.
func (s *Section1) DataTypeName() string {
	return tables.GetDataTypeName(int(s.TypeOfData))
}
