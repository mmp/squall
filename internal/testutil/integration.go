// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"strings"
)

// IntegrationTestResult holds the results of comparing mgrib2 against reference implementations.
type IntegrationTestResult struct {
	FileName string

	// Field comparisons
	TotalFields       int
	FieldsCompared    int
	Wgrib2Comparisons []*ComparisonResult
	GoGrib2Comparisons []*ComparisonResult

	// Summary
	AllWgrib2Match  bool
	AllGoGrib2Match bool
	Errors          []string
}

// CompareImplementations compares mgrib2 against wgrib2 and go-grib2 for a given GRIB2 file.
//
// maxULP specifies the maximum ULP difference allowed for floating-point comparisons.
// Typically 10-100 ULPs is reasonable for numerical accuracy differences.
//
// For large files (many messages), wgrib2 CSV comparison is skipped to avoid memory issues.
func CompareImplementations(gribFile string, maxULP int64) (*IntegrationTestResult, error) {
	result := &IntegrationTestResult{
		FileName:           gribFile,
		Wgrib2Comparisons:  []*ComparisonResult{},
		GoGrib2Comparisons: []*ComparisonResult{},
		AllWgrib2Match:     true,
		AllGoGrib2Match:    true,
	}

	// Parse with mgrib2 (our implementation)
	mgribFields, err := ParseMgrib2(gribFile)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("mgrib2 parse failed: %v", err))
		return result, nil // Return result with error, don't fail completely
	}

	// Skip wgrib2 CSV for large files (CSV output can be huge - millions of lines)
	// Only use wgrib2 for files with < 100 messages
	if len(mgribFields) < 100 {
		// Parse with wgrib2
		wgrib2Fields, err := ParseWgrib2CSV(gribFile)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("wgrib2 parse failed: %v", err))
		} else {
			// Compare mgrib2 vs wgrib2
			compareAgainstReference(mgribFields, wgrib2Fields, maxULP, result, true)
		}
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("Skipping wgrib2 CSV comparison (file has %d messages, limit is 100)", len(mgribFields)))
	}

	// Parse with go-grib2
	goGrib2Fields, err := ParseGoGrib2(gribFile)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("go-grib2 parse failed: %v", err))
	} else {
		// Compare mgrib2 vs go-grib2
		compareAgainstReference(mgribFields, goGrib2Fields, maxULP, result, false)
	}

	result.TotalFields = len(mgribFields)

	return result, nil
}

// compareAgainstReference compares mgrib2 fields against a reference implementation.
func compareAgainstReference(
	mgribFields, refFields map[string]*FieldData,
	maxULP int64,
	result *IntegrationTestResult,
	isWgrib2 bool,
) {
	// Compare each mgrib2 field against reference
	for key, mgribField := range mgribFields {
		refField, exists := refFields[key]
		if !exists {
			// Try fuzzy matching on field name (different implementations may use different names)
			refField = findMatchingField(mgribField, refFields)
			if refField == nil {
				result.Errors = append(result.Errors,
					fmt.Sprintf("field %s not found in reference implementation", key))
				if isWgrib2 {
					result.AllWgrib2Match = false
				} else {
					result.AllGoGrib2Match = false
				}
				continue
			}
		}

		// Compare fields
		comparison := CompareFields(mgribField, refField, maxULP)
		comparison.Source = fmt.Sprintf("mgrib2 vs %s", refField.Source)

		if isWgrib2 {
			result.Wgrib2Comparisons = append(result.Wgrib2Comparisons, comparison)
			if !comparison.MetadataMatch || !comparison.CoordinatesMatch || !comparison.DataMatch {
				result.AllWgrib2Match = false
			}
		} else {
			result.GoGrib2Comparisons = append(result.GoGrib2Comparisons, comparison)
			if !comparison.MetadataMatch || !comparison.CoordinatesMatch || !comparison.DataMatch {
				result.AllGoGrib2Match = false
			}
		}

		result.FieldsCompared++
	}
}

// findMatchingField attempts to find a matching field by fuzzy matching on field name.
func findMatchingField(target *FieldData, candidates map[string]*FieldData) *FieldData {
	// Try exact level match first
	for _, candidate := range candidates {
		if candidate.Level == target.Level {
			// Check if field names are similar (case-insensitive partial match)
			targetField := strings.ToLower(target.Field)
			candidateField := strings.ToLower(candidate.Field)

			if targetField == candidateField ||
				strings.Contains(targetField, candidateField) ||
				strings.Contains(candidateField, targetField) {
				return candidate
			}
		}
	}

	return nil
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

	// Go-grib2 comparison
	if len(r.GoGrib2Comparisons) > 0 {
		b.WriteString("=== Comparison vs go-grib2 ===\n")
		if r.AllGoGrib2Match {
			b.WriteString("✓ All fields match within tolerance\n")
		} else {
			b.WriteString("✗ Some fields have differences\n")
		}

		for _, comp := range r.GoGrib2Comparisons {
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
func (r *IntegrationTestResult) Passed() bool {
	return r.AllWgrib2Match && r.AllGoGrib2Match && len(r.Errors) == 0
}
