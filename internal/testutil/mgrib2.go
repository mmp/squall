// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"os"

	"github.com/mmp/mgrib2"
)

// ParseMgrib2 parses a GRIB2 file using the mgrib2 library (this implementation).
//
// Returns an array of FieldData structures in message order.
func ParseMgrib2(gribFile string) ([]*FieldData, error) {
	// Open GRIB2 file
	file, err := os.Open(gribFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Parse with mgrib2 (use sequential + skip errors for robustness)
	fields, err := mgrib2.ReadWithOptions(file,
		mgrib2.WithSequential(),
		mgrib2.WithSkipErrors())
	if err != nil {
		return nil, fmt.Errorf("mgrib2 parse failed: %v", err)
	}

	// Convert to FieldData array (preserving message order)
	fieldArray := make([]*FieldData, 0, len(fields))

	for _, field := range fields {
		// TODO: Calculate verification time from forecast time
		// For now, use reference time for both
		verTime := field.ReferenceTime

		fd := &FieldData{
			RefTime:    field.ReferenceTime,
			VerTime:    verTime,
			Field:      field.ParameterName,
			Level:      field.Level,
			Latitudes:  field.Latitudes,
			Longitudes: field.Longitudes,
			Values:     field.Data,
			Source:     "mgrib2",
		}

		fieldArray = append(fieldArray, fd)
	}

	return fieldArray, nil
}
