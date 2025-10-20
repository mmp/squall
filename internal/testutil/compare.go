// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"math"
	"time"
)

// FieldData represents a parsed GRIB2 field for comparison.
type FieldData struct {
	// Metadata
	RefTime  time.Time
	VerTime  time.Time
	Field    string
	Level    string

	// Grid data
	Latitudes  []float64
	Longitudes []float64
	Values     []float64

	// Source for debugging
	Source string
}

// ComparisonResult holds the results of comparing two fields.
type ComparisonResult struct {
	MetadataMatch   bool
	CoordinatesMatch bool
	DataMatch       bool

	MessageCount    int
	ExactMatches    int
	TotalPoints     int
	MaxULPDiff      int64
	MeanULPDiff     float64
	PointsExceeding int

	Source string
	Errors []string
}

// ULPDiff calculates the ULP (Units in Last Place) difference between two float64 values.
//
// ULP is a measure of the smallest representable difference between two floating-point
// numbers. A difference of 1 ULP means the numbers are as close as they can be while
// still being different.
func ULPDiff(a, b float64) int64 {
	// Handle special cases
	if math.IsNaN(a) || math.IsNaN(b) {
		if math.IsNaN(a) && math.IsNaN(b) {
			return 0
		}
		return math.MaxInt64
	}

	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		if a == b {
			return 0
		}
		return math.MaxInt64
	}

	// Handle exact equality
	if a == b {
		return 0
	}

	// Handle sign differences
	if (a < 0) != (b < 0) {
		// Different signs - calculate ULPs to zero and add
		aULP := math.Abs(float64(math.Float64bits(a)))
		bULP := math.Abs(float64(math.Float64bits(b)))
		return int64(aULP + bULP)
	}

	// Same sign - calculate bit difference
	aBits := math.Float64bits(a)
	bBits := math.Float64bits(b)

	var diff uint64
	if aBits > bBits {
		diff = aBits - bBits
	} else {
		diff = bBits - aBits
	}

	return int64(diff)
}

// CompareFloatsULP checks if two floats are within a specified ULP tolerance.
func CompareFloatsULP(a, b float64, maxULP int64) bool {
	// Handle missing values (9.999e20)
	if a > 9e20 && b > 9e20 {
		return true
	}
	if a > 9e20 || b > 9e20 {
		return false
	}

	return ULPDiff(a, b) <= maxULP
}

// CompareFields compares two FieldData structures with the specified ULP tolerance.
func CompareFields(a, b *FieldData, maxULP int64) *ComparisonResult {
	result := &ComparisonResult{
		Source: fmt.Sprintf("%s vs %s", a.Source, b.Source),
	}

	// Compare metadata
	result.MetadataMatch = compareMetadata(a, b, result)

	// Compare coordinates
	result.CoordinatesMatch = compareCoordinates(a, b, result)

	// Compare data values
	if len(a.Values) != len(b.Values) {
		result.Errors = append(result.Errors, fmt.Sprintf("value count mismatch: %d vs %d", len(a.Values), len(b.Values)))
		result.DataMatch = false
		return result
	}

	result.TotalPoints = len(a.Values)
	var ulpSum int64

	for i := range a.Values {
		ulpDiff := ULPDiff(a.Values[i], b.Values[i])

		if ulpDiff == 0 {
			result.ExactMatches++
		}

		if ulpDiff > result.MaxULPDiff {
			result.MaxULPDiff = ulpDiff
		}

		ulpSum += ulpDiff

		if ulpDiff > maxULP {
			result.PointsExceeding++
			// Only report first few excessive differences
			if result.PointsExceeding <= 5 {
				result.Errors = append(result.Errors,
					fmt.Sprintf("point %d: ULP diff %d exceeds tolerance %d (%.6f vs %.6f)",
						i, ulpDiff, maxULP, a.Values[i], b.Values[i]))
			}
		}
	}

	if result.TotalPoints > 0 {
		result.MeanULPDiff = float64(ulpSum) / float64(result.TotalPoints)
	}

	result.DataMatch = result.PointsExceeding == 0

	return result
}

// compareMetadata checks if metadata fields match.
func compareMetadata(a, b *FieldData, result *ComparisonResult) bool {
	match := true

	if !a.RefTime.Equal(b.RefTime) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("reference time mismatch: %v vs %v", a.RefTime, b.RefTime))
		match = false
	}

	if !a.VerTime.Equal(b.VerTime) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("verification time mismatch: %v vs %v", a.VerTime, b.VerTime))
		match = false
	}

	if a.Field != b.Field {
		result.Errors = append(result.Errors,
			fmt.Sprintf("field name mismatch: %s vs %s", a.Field, b.Field))
		match = false
	}

	if a.Level != b.Level {
		result.Errors = append(result.Errors,
			fmt.Sprintf("level mismatch: %s vs %s", a.Level, b.Level))
		match = false
	}

	return match
}

// compareCoordinates checks if coordinate arrays match.
func compareCoordinates(a, b *FieldData, result *ComparisonResult) bool {
	match := true

	if len(a.Latitudes) != len(b.Latitudes) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("latitude count mismatch: %d vs %d", len(a.Latitudes), len(b.Latitudes)))
		match = false
	}

	if len(a.Longitudes) != len(b.Longitudes) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("longitude count mismatch: %d vs %d", len(a.Longitudes), len(b.Longitudes)))
		match = false
	}

	// Sample check first, middle, and last coordinates
	if len(a.Latitudes) > 0 && len(b.Latitudes) > 0 {
		indices := []int{0}
		if len(a.Latitudes) > 1 {
			indices = append(indices, len(a.Latitudes)/2, len(a.Latitudes)-1)
		}

		for _, i := range indices {
			if i < len(a.Latitudes) && i < len(b.Latitudes) {
				if math.Abs(a.Latitudes[i]-b.Latitudes[i]) > 0.001 {
					result.Errors = append(result.Errors,
						fmt.Sprintf("latitude[%d] mismatch: %.6f vs %.6f", i, a.Latitudes[i], b.Latitudes[i]))
					match = false
				}
				if math.Abs(a.Longitudes[i]-b.Longitudes[i]) > 0.001 {
					result.Errors = append(result.Errors,
						fmt.Sprintf("longitude[%d] mismatch: %.6f vs %.6f", i, a.Longitudes[i], b.Longitudes[i]))
					match = false
				}
			}
		}
	}

	return match
}

// String returns a human-readable summary of the comparison result.
func (r *ComparisonResult) String() string {
	status := "✓"
	if !r.MetadataMatch || !r.CoordinatesMatch || !r.DataMatch {
		status = "✗"
	}

	summary := fmt.Sprintf("%s Comparison Result:\n", status)
	summary += fmt.Sprintf("  Metadata: %v\n", statusSymbol(r.MetadataMatch))
	summary += fmt.Sprintf("  Coordinates: %v\n", statusSymbol(r.CoordinatesMatch))

	if r.TotalPoints > 0 {
		exactPct := 100.0 * float64(r.ExactMatches) / float64(r.TotalPoints)
		summary += fmt.Sprintf("  Data: %v (%d points, %.1f%% exact, max ULP: %d, mean ULP: %.1f)\n",
			statusSymbol(r.DataMatch), r.TotalPoints, exactPct, r.MaxULPDiff, r.MeanULPDiff)
	}

	if len(r.Errors) > 0 {
		summary += "\n  Errors:\n"
		for _, err := range r.Errors {
			summary += fmt.Sprintf("    - %s\n", err)
		}
	}

	return summary
}

func statusSymbol(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
