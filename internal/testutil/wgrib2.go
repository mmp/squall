// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Wgrib2Path is the path to the wgrib2 binary.
var Wgrib2Path = "/Users/mmp/bin/wgrib2"

// ParseWgrib2 runs wgrib2 on a GRIB2 file and extracts data with full float32 precision.
//
// Uses wgrib2's -gridout (for coordinates) and -ieee (for binary float values) options
// to extract data with full precision, rather than CSV which has limited decimal places.
//
// Returns an array of FieldData structures in message order (message 1, message 2, etc.).
func ParseWgrib2(gribFile string) ([]*FieldData, error) {
	// First, get the inventory to know how many messages there are
	invCmd := exec.Command(Wgrib2Path, gribFile)
	invOutput, err := invCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wgrib2 inventory failed: %v\nOutput: %s", err, invOutput)
	}

	// Parse inventory to get message metadata
	messages, err := parseInventory(string(invOutput))
	if err != nil {
		return nil, fmt.Errorf("failed to parse inventory: %v", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in GRIB2 file")
	}

	// Process each message
	var result []*FieldData
	for i, msg := range messages {
		// Create temp files for this message's output
		coordFile, err := os.CreateTemp("", fmt.Sprintf("wgrib2_coords_%d_*.txt", i+1))
		if err != nil {
			return nil, fmt.Errorf("failed to create temp coord file: %v", err)
		}
		coordPath := coordFile.Name()
		_ = coordFile.Close()
		defer func() {
			_ = os.Remove(coordPath)
		}()

		valueFile, err := os.CreateTemp("", fmt.Sprintf("wgrib2_values_%d_*.bin", i+1))
		if err != nil {
			return nil, fmt.Errorf("failed to create temp value file: %v", err)
		}
		valuePath := valueFile.Name()
		_ = valueFile.Close()
		defer func() {
			_ = os.Remove(valuePath)
		}()

		// Run wgrib2 to extract coordinates and values for this specific message
		// -d specifies which message to process (1-indexed)
		// -order we:sn ensures consistent west-to-east, south-to-north ordering
		// -no_header removes fortran-style headers
		// -little_endian ensures proper endianness for Go's binary.Read
		cmd := exec.Command(Wgrib2Path, gribFile,
			"-d", fmt.Sprintf("%d", i+1),
			"-order", "we:sn",
			"-gridout", coordPath,
			"-no_header",
			"-little_endian",
			"-ieee", valuePath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("wgrib2 extraction failed for message %d: %v\nOutput: %s", i+1, err, output)
		}

		// Read coordinates from gridout file
		// Format: "i, j, lat, lon" (one per line)
		coords, err := readGridout(coordPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read coordinates for message %d: %v", i+1, err)
		}

		// Read values from IEEE binary file
		values, err := readIEEEBinary(valuePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read values for message %d: %v", i+1, err)
		}

		// Verify counts match
		if len(coords) != len(values) {
			return nil, fmt.Errorf("message %d: coordinate count (%d) != value count (%d)",
				i+1, len(coords), len(values))
		}

		// Build FieldData
		latitudes := make([]float32, len(coords))
		longitudes := make([]float32, len(coords))
		for j, coord := range coords {
			latitudes[j] = coord.lat
			longitudes[j] = coord.lon
		}

		fd := &FieldData{
			RefTime:    msg.refTime,
			VerTime:    msg.verTime,
			Field:      msg.field,
			Level:      msg.level,
			Latitudes:  latitudes,
			Longitudes: longitudes,
			Values:     values,
			Source:     "wgrib2",
		}

		result = append(result, fd)
	}

	return result, nil
}

// messageMetadata holds metadata parsed from wgrib2 inventory output.
type messageMetadata struct {
	msgNum  int
	refTime time.Time
	verTime time.Time
	field   string
	level   string
}

// coord holds a lat/lon coordinate pair.
type coord struct {
	lat float32
	lon float32
}

// parseInventory parses wgrib2 inventory output to extract message metadata.
//
// Example inventory line:
// 1:0:d=2025101511:HGT:50 mb:anl:
func parseInventory(inv string) ([]messageMetadata, error) {
	var messages []messageMetadata

	// Regex to parse inventory lines
	// Format: msgnum:offset:d=YYYYMMDDHH:field:level:forecast
	re := regexp.MustCompile(`^(\d+):\d+:d=(\d{10})(?:\d{2})?:([^:]+):([^:]+):(.*)$`)

	scanner := bufio.NewScanner(strings.NewReader(inv))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if matches == nil {
			// Try to be lenient - some messages have different formats
			// Just extract what we can
			continue
		}

		msgNum, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid message number: %v", err)
		}

		// Parse reference time (format: YYYYMMDDhh or YYYYMMDDhhmm)
		refTimeStr := matches[2]
		refTime, err := parseWgrib2Time("d=" + refTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid reference time: %v", err)
		}

		field := matches[3]
		level := matches[4]
		forecast := strings.TrimSpace(matches[5])

		// Compute verification time
		// If forecast is "anl" (analysis), verTime = refTime
		// If forecast is like "1 hour fcst", parse the duration
		verTime := refTime
		if forecast != "anl" && forecast != "" {
			duration, err := parseForecastTime(forecast)
			if err == nil {
				verTime = refTime.Add(duration)
			}
			// If parse fails, just use refTime (best effort)
		}

		messages = append(messages, messageMetadata{
			msgNum:  msgNum,
			refTime: refTime,
			verTime: verTime,
			field:   field,
			level:   level,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading inventory: %v", err)
	}

	return messages, nil
}

// parseForecastTime parses forecast time strings like "1 hour fcst", "6 hour fcst", etc.
func parseForecastTime(s string) (time.Duration, error) {
	// Common formats: "1 hour fcst", "6 hour fcst", "1 day fcst", etc.
	re := regexp.MustCompile(`^(\d+)\s+(hour|day|min|minute)s?\s+fcst$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("unrecognized forecast format: %s", s)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	unit := matches[2]
	switch unit {
	case "min", "minute":
		return time.Duration(value) * time.Minute, nil
	case "hour":
		return time.Duration(value) * time.Hour, nil
	case "day":
		return time.Duration(value) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown time unit: %s", unit)
	}
}

// parseWgrib2Time parses time strings from wgrib2 output.
//
// Wgrib2 outputs times in various formats. Common formats include:
// - "d=2025101511" (compact format: d=YYYYMMDDhh)
// - "d=202510151100" (with minutes: d=YYYYMMDDhhmm)
// - "2025-01-15T12:00:00Z" (ISO 8601)
func parseWgrib2Time(s string) (time.Time, error) {
	// Remove common prefixes
	s = strings.TrimPrefix(s, "d=")

	// Try various formats
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

// readGridout reads wgrib2 -gridout output.
//
// Format: "i, j, lat, lon" (one per line)
// Example:
//
//	1,         1, 40.409, 263.379
//	2,         1, 40.409, 263.415
func readGridout(path string) ([]coord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	var coords []coord
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse format: "i, j, lat, lon"
		// Split by comma
		parts := strings.Split(line, ",")
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid gridout line (expected 4 fields): %s", line)
		}

		// Extract lat and lon (parts[2] and parts[3])
		latStr := strings.TrimSpace(parts[2])
		lonStr := strings.TrimSpace(parts[3])

		lat, err := strconv.ParseFloat(latStr, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude: %s", latStr)
		}

		lon, err := strconv.ParseFloat(lonStr, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude: %s", lonStr)
		}

		coords = append(coords, coord{
			lat: float32(lat),
			lon: float32(lon),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return coords, nil
}

// readIEEEBinary reads wgrib2 -ieee binary output (little-endian float32 values).
func readIEEEBinary(path string) ([]float32, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	// Get file size to pre-allocate slice
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	numValues := stat.Size() / 4 // Each float32 is 4 bytes
	if stat.Size()%4 != 0 {
		return nil, fmt.Errorf("file size (%d bytes) not divisible by 4", stat.Size())
	}

	values := make([]float32, numValues)

	// Read all float32 values (little-endian)
	err = binary.Read(file, binary.LittleEndian, &values)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary floats: %v", err)
	}

	// Check for NaN/Inf (wgrib2 uses these for missing values)
	// Replace with Go's convention of NaN
	for i := range values {
		if math.IsInf(float64(values[i]), 0) {
			values[i] = float32(math.NaN())
		}
	}

	return values, nil
}
