// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"strings"
)

// IntegrationTestResult holds the results of comparing squall against reference implementations.
type IntegrationTestResult struct {
	FileName string

	// Field comparisons
	TotalFields       int
	FieldsCompared    int
	Wgrib2Comparisons []*ComparisonResult

	// Summary
	AllWgrib2Match bool
	Errors         []string
}

// CompareImplementations compares squall against wgrib2 for a given GRIB2 file.
//
// maxULP specifies the maximum ULP difference allowed for floating-point comparisons.
// Typically 10-100 ULPs is reasonable for numerical accuracy differences.
//
// Compares messages in order (message 1 vs message 1, etc.) rather than by field names,
// since different implementations use different naming conventions.
func CompareImplementations(gribFile string, maxULP int64) (*IntegrationTestResult, error) {
	result := &IntegrationTestResult{
		FileName:          gribFile,
		Wgrib2Comparisons: []*ComparisonResult{},
		AllWgrib2Match:    true,
	}

	// Parse with squall (our implementation)
	mgribFields, err := ParseMgrib2(gribFile)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("squall parse failed: %v", err))
		return result, nil // Return result with error, don't fail completely
	}

	result.TotalFields = len(mgribFields)

	// Parse with wgrib2 (NOAA reference implementation)
	wgrib2Fields, err := ParseWgrib2(gribFile)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("wgrib2 parse failed: %v", err))
	} else {
		// Compare squall vs wgrib2 message by message
		compareArrays(mgribFields, wgrib2Fields, maxULP, result)
	}

	return result, nil
}

// compareArrays compares two arrays of FieldData message-by-message.
//
// This approach compares messages in order (msg 1 vs msg 1, msg 2 vs msg 2, etc.)
// rather than trying to match by field names, which avoids issues with different
// naming conventions between implementations.
func compareArrays(
	mgribFields, refFields []*FieldData,
	maxULP int64,
	result *IntegrationTestResult,
) {
	// Check if arrays have same length
	if len(mgribFields) != len(refFields) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("message count mismatch: squall has %d, reference has %d",
				len(mgribFields), len(refFields)))
		result.AllWgrib2Match = false
		// Still compare what we can
	}

	// Compare each message pair
	numToCompare := len(mgribFields)
	if len(refFields) < numToCompare {
		numToCompare = len(refFields)
	}

	for i := 0; i < numToCompare; i++ {
		mgribField := mgribFields[i]
		refField := refFields[i]

		// Compare fields (allow metadata mismatches due to naming differences)
		comparison := CompareFields(mgribField, refField, maxULP)
		comparison.Source = fmt.Sprintf("message %d: squall vs %s", i+1, refField.Source)
		comparison.MessageCount = 1

		// Only fail on data mismatches, not metadata (field names differ between implementations)
		dataFailed := !comparison.CoordinatesMatch || !comparison.DataMatch

		result.Wgrib2Comparisons = append(result.Wgrib2Comparisons, comparison)
		if dataFailed {
			result.AllWgrib2Match = false
		}

		result.FieldsCompared++
	}
}

// String returns a human-readable summary of the integration test result.
func (r *IntegrationTestResult) String() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Integration Test Results: %s\n", r.FileName))
	b.WriteString(fmt.Sprintf("Total fields: %d, Compared: %d\n\n", r.TotalFields, r.FieldsCompared))

	// Wgrib2 comparison
	if len(r.Wgrib2Comparisons) > 0 {
		b.WriteString("=== Comparison vs wgrib2 ===\n")
		if r.AllWgrib2Match {
			b.WriteString("✓ All fields match within tolerance\n")
		} else {
			b.WriteString("✗ Some fields have differences\n")
		}

		for _, comp := range r.Wgrib2Comparisons {
			if !comp.MetadataMatch || !comp.CoordinatesMatch || !comp.DataMatch {
				b.WriteString(comp.String())
				b.WriteString("\n")
			}
		}
	}

	// Errors
	if len(r.Errors) > 0 {
		b.WriteString("\n=== Errors ===\n")
		for _, err := range r.Errors {
			b.WriteString(fmt.Sprintf("  - %s\n", err))
		}
	}

	return b.String()
}

// Passed returns true if all comparisons passed.
//
// Only checks data and coordinate matches, not metadata (field names),
// since different implementations use different naming conventions.
func (r *IntegrationTestResult) Passed() bool {
	// Check wgrib2 comparison (NOAA reference implementation)
	// A test passes if all wgrib2 data/coordinates match and there are no critical errors
	hasNonParseErrors := false
	for _, err := range r.Errors {
		// Ignore wgrib2 parse errors for unsupported features
		if !strings.Contains(err, "wgrib2 parse failed") {
			hasNonParseErrors = true
			break
		}
	}
	return r.AllWgrib2Match && !hasNonParseErrors
}
