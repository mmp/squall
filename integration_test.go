package grib_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/mmp/mgrib2/internal/testutil"
)

var (
	// Flag to allow testing very large files (e.g., full CONUS HRRR files)
	noSizeLimit = flag.Bool("no-size-limit", false, "Allow testing files of any size (default: 15MB limit)")
)

// TestIntegrationWithRealFiles tests mgrib2 against the wgrib2 reference implementation
// using real GRIB2 files from the testgribs/ directory.
//
// This test validates decoded values, coordinates, and grid data against wgrib2
// (NOAA's official reference implementation) with full float32 precision.
// By default, files larger than 15 MB are skipped. Use -no-size-limit to test all files.
//
// Examples:
//
//	go test -v -run TestIntegration                    # Test files up to 15 MB
//	go test -v -run TestIntegration -no-size-limit     # Test all files including large CONUS
func TestIntegrationWithRealFiles(t *testing.T) {
	// Look for GRIB2 files in testgribs directory
	testgribsDir := "testgribs"
	if _, err := os.Stat(testgribsDir); os.IsNotExist(err) {
		t.Skip("testgribs directory not found - skipping integration tests")
		return
	}

	// Find all .grib2 files
	matches, err := filepath.Glob(filepath.Join(testgribsDir, "*.grib2"))
	if err != nil {
		t.Fatalf("failed to search for GRIB2 files: %v", err)
	}

	// Also try .grb2 extension
	matches2, err := filepath.Glob(filepath.Join(testgribsDir, "*.grb2"))
	if err != nil {
		t.Fatalf("failed to search for .grb2 files: %v", err)
	}
	matches = append(matches, matches2...)

	if len(matches) == 0 {
		t.Skip("no GRIB2 files found in testgribs directory - skipping integration tests")
		return
	}

	t.Logf("Found %d GRIB2 files to test", len(matches))

	// Test each file
	for _, gribFile := range matches {
		t.Run(filepath.Base(gribFile), func(t *testing.T) {
			testGRIB2File(t, gribFile)
		})
	}
}

// testGRIB2File tests a single GRIB2 file against reference implementations.
func testGRIB2File(t *testing.T, gribFile string) {
	info, err := os.Stat(gribFile)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	// Size limit: 15 MB by default, or unlimited if flag is set
	const defaultSizeLimit = 15 * 1024 * 1024 // 15 MB
	if !*noSizeLimit && info.Size() > defaultSizeLimit {
		t.Skipf("Skipping large file %s (%.1f MB) - use -no-size-limit to test large files",
			filepath.Base(gribFile), float64(info.Size())/1024/1024)
		return
	}

	t.Logf("Testing %s (%.1f MB)", filepath.Base(gribFile), float64(info.Size())/1024/1024)

	// Maximum ULP difference allowed
	// Use reasonable tolerance for float32 precision comparison
	// Both mgrib2 and wgrib2 output float32 values, so ULP comparison is meaningful
	maxULP := int64(100) // Allow for minor rounding differences in float32 arithmetic

	// Run comparison
	result, err := testutil.CompareImplementations(gribFile, maxULP)
	if err != nil {
		t.Fatalf("comparison failed: %v", err)
	}

	// Log full results
	t.Log(result.String())

	// Check if test passed
	if !result.Passed() {
		t.Errorf("integration test failed for %s", filepath.Base(gribFile))

		// Print summary statistics
		if len(result.Wgrib2Comparisons) > 0 {
			printComparisonStats(t, "wgrib2", result.Wgrib2Comparisons)
		}
	} else {
		// Print success summary
		if len(result.Wgrib2Comparisons) > 0 {
			printComparisonStats(t, "wgrib2", result.Wgrib2Comparisons)
		}
	}
}

// printComparisonStats prints summary statistics for a set of comparisons.
func printComparisonStats(t *testing.T, refName string, comparisons []*testutil.ComparisonResult) {
	t.Logf("\n=== Statistics vs %s ===", refName)

	var totalMessages int
	var totalPoints int
	var totalExact int
	var maxULP int64
	var sumMeanULP float64
	var failedMessages int

	for _, comp := range comparisons {
		totalMessages++
		totalPoints += comp.TotalPoints
		totalExact += comp.ExactMatches
		if comp.MaxULPDiff > maxULP {
			maxULP = comp.MaxULPDiff
		}
		sumMeanULP += comp.MeanULPDiff

		// A comparison passes if coordinates and data match (ignore metadata since field names differ)
		if !comp.CoordinatesMatch || !comp.DataMatch {
			failedMessages++
		}
	}

	if len(comparisons) > 0 {
		avgMeanULP := sumMeanULP / float64(len(comparisons))
		exactPct := 100.0 * float64(totalExact) / float64(totalPoints)

		t.Logf("Messages compared: %d", totalMessages)
		t.Logf("Messages passed: %d (%.1f%%)", totalMessages-failedMessages,
			100.0*float64(totalMessages-failedMessages)/float64(totalMessages))
		t.Logf("Total grid points: %d", totalPoints)
		t.Logf("Exact matches: %d (%.1f%%)", totalExact, exactPct)
		t.Logf("Max ULP diff: %d", maxULP)
		t.Logf("Avg mean ULP: %.1f", avgMeanULP)
	}
}

// Example test showing manual comparison
func ExampleCompareImplementations() {
	// Compare mgrib2 against reference implementations
	result, err := testutil.CompareImplementations("testgribs/sample.grib2", 100)
	if err != nil {
		panic(err)
	}

	// Print results
	println(result.String())

	// Check if test passed
	if result.Passed() {
		println("All comparisons passed!")
	} else {
		println("Some comparisons failed")
	}
}
