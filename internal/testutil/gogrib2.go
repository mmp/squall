// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"os"

	"github.com/mmp/go-grib2"
)

// ParseGoGrib2 parses a GRIB2 file using the go-grib2 library.
//
// Returns a map of field keys (name:level) to FieldData structures.
func ParseGoGrib2(gribFile string) (map[string]*FieldData, error) {
	// Read GRIB2 file
	data, err := os.ReadFile(gribFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Parse with go-grib2
	gribs, err := grib2.Read(data)
	if err != nil {
		return nil, fmt.Errorf("go-grib2 parse failed: %v", err)
	}

	// Convert to FieldData map
	fieldMap := make(map[string]*FieldData)

	for _, grib := range gribs {
		// Create field key
		key := fmt.Sprintf("%s:%s", grib.Name, grib.Level)

		// Extract coordinates and values
		latitudes := make([]float64, len(grib.Values))
		longitudes := make([]float64, len(grib.Values))
		values := make([]float64, len(grib.Values))

		for i, v := range grib.Values {
			latitudes[i] = float64(v.Latitude)
			longitudes[i] = float64(v.Longitude)
			values[i] = float64(v.Value)
		}

		fd := &FieldData{
			RefTime:    grib.RefTime,
			VerTime:    grib.VerfTime,
			Field:      grib.Name,
			Level:      grib.Level,
			Latitudes:  latitudes,
			Longitudes: longitudes,
			Values:     values,
			Source:     "go-grib2",
		}

		fieldMap[key] = fd
	}

	return fieldMap, nil
}
