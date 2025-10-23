// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"os"

	grib "github.com/mmp/squall"
)

// ParseMgrib2 parses a GRIB2 file using squall (this implementation).
//
// Returns an array of FieldData structures in message order.
func ParseMgrib2(gribFile string) ([]*FieldData, error) {
	// Open GRIB2 file
	file, err := os.Open(gribFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Parse with squall (use sequential + skip errors for robustness)
	fields, err := grib.ReadWithOptions(file,
		grib.WithSequential(),
		grib.WithSkipErrors())
	if err != nil {
		return nil, fmt.Errorf("squall parse failed: %v", err)
	}

	// Convert to FieldData array (preserving message order)
	fieldArray := make([]*FieldData, 0, len(fields))

	for _, field := range fields {
		// TODO: Calculate verification time from forecast time
		// For now, use reference time for both
		verTime := field.ReferenceTime

		// Use short name for comparison with wgrib2 (if available)
		fieldName := field.Parameter.ShortName()
		if fieldName == "" {
			// Fall back to full name if no short name exists
			fieldName = field.Parameter.String()
		}

		fd := &FieldData{
			RefTime:    field.ReferenceTime,
			VerTime:    verTime,
			Field:      fieldName,
			Level:      field.Level,
			Latitudes:  field.Latitudes,
			Longitudes: field.Longitudes,
			Values:     field.Data,
			Source:     "squall",
		}

		fieldArray = append(fieldArray, fd)
	}

	return fieldArray, nil
}
