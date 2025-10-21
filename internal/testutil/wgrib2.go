// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Wgrib2Path is the path to the wgrib2 binary.
var Wgrib2Path = "/Users/mmp/bin/wgrib2"

// ParseWgrib2CSV runs wgrib2 on a GRIB2 file and parses the CSV output.
//
// The wgrib2 tool is invoked with the -csv flag to produce output in the format:
// "time0","time1","field","level",lon,lat,value
//
// Returns a map of field keys (field:level) to FieldData structures.
func ParseWgrib2CSV(gribFile string) (map[string]*FieldData, error) {
	// Run wgrib2 with CSV output
	cmd := exec.Command(Wgrib2Path, gribFile, "-csv", "-")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wgrib2 failed: %v\nOutput: %s", err, output)
	}

	// Parse CSV output
	reader := csv.NewReader(strings.NewReader(string(output)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV parse failed: %v", err)
	}

	// Group records by field key (field:level)
	fieldMap := make(map[string]*FieldData)

	for i, record := range records {
		if len(record) != 7 {
			return nil, fmt.Errorf("invalid CSV record at line %d: expected 7 fields, got %d", i+1, len(record))
		}

		// Parse fields
		refTimeStr := record[0]
		verTimeStr := record[1]
		field := record[2]
		level := record[3]
		lonStr := record[4]
		latStr := record[5]
		valueStr := record[6]

		// Parse coordinates
		lon, err := strconv.ParseFloat(lonStr, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude at line %d: %v", i+1, err)
		}

		lat, err := strconv.ParseFloat(latStr, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude at line %d: %v", i+1, err)
		}

		// Parse value
		value, err := strconv.ParseFloat(valueStr, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value at line %d: %v", i+1, err)
		}

		// Create field key
		key := fmt.Sprintf("%s:%s", field, level)

		// Get or create FieldData
		fd, exists := fieldMap[key]
		if !exists {
			// Parse reference time (format: 2025-01-15T12:00:00Z or similar)
			refTime, err := parseWgrib2Time(refTimeStr)
			if err != nil {
				return nil, fmt.Errorf("invalid reference time at line %d: %v", i+1, err)
			}

			// Parse verification time
			verTime, err := parseWgrib2Time(verTimeStr)
			if err != nil {
				return nil, fmt.Errorf("invalid verification time at line %d: %v", i+1, err)
			}

			fd = &FieldData{
				RefTime:    refTime,
				VerTime:    verTime,
				Field:      field,
				Level:      level,
				Latitudes:  []float32{},
				Longitudes: []float32{},
				Values:     []float32{},
				Source:     "wgrib2",
			}
			fieldMap[key] = fd
		}

		// Append data point
		fd.Latitudes = append(fd.Latitudes, float32(lat))
		fd.Longitudes = append(fd.Longitudes, float32(lon))
		fd.Values = append(fd.Values, float32(value))
	}

	return fieldMap, nil
}

// parseWgrib2Time parses time strings from wgrib2 CSV output.
//
// Wgrib2 outputs times in various formats. Common formats include:
// - "2025-01-15T12:00:00Z" (ISO 8601)
// - "2025011512" (compact format: YYYYMMDDhh)
// - "d=2025011512" (with prefix)
func parseWgrib2Time(s string) (time.Time, error) {
	// Remove common prefixes
	s = strings.TrimPrefix(s, "d=")

	// Try ISO 8601 format first
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006010215",     // YYYYMMDDhh
		"200601021504",   // YYYYMMDDhhmm
		"20060102150405", // YYYYMMDDhhmmss
	}

	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %s", s)
}
