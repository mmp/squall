// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"os"

	"github.com/mmp/mgrib2"
)

// ParseMgrib2 parses a GRIB2 file using the mgrib2 library (this implementation).
//
// Returns a map of field keys (parameter:level) to FieldData structures.
func ParseMgrib2(gribFile string) (map[string]*FieldData, error) {
	// Read GRIB2 file
	data, err := os.ReadFile(gribFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Parse with mgrib2
	fields, err := mgrib2.Read(data)
	if err != nil {
		return nil, fmt.Errorf("mgrib2 parse failed: %v", err)
	}

	// Convert to FieldData map
	fieldMap := make(map[string]*FieldData)

	for _, field := range fields {
		// Create field key using parameter name and level description
		key := fmt.Sprintf("%s:%s", field.ParameterName, field.Level)

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

		fieldMap[key] = fd
	}

	return fieldMap, nil
}
