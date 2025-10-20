package mgrib2

import (
	"os"
	"testing"
)

func TestParseHRRRCONUS(t *testing.T) {
	// Test with full CONUS HRRR file (much larger than Iowa subset)
	// Grid: 1799x1059 = 1,905,141 points per field (vs 22,632 for Iowa)
	// File: 357 MB (vs 12.5 MB for Iowa)
	//
	// Note: This file primarily uses Data Representation Template 5.3
	// (complex packing) which is not yet implemented. We successfully
	// parse the ~44 fields that use Template 5.0 (simple packing).
	data, err := os.ReadFile("/Users/mmp/Downloads/hrrr.20251015-conus-hrrr.t11z.wrfprsf00.grib2")
	if err != nil {
		t.Skip("CONUS HRRR file not found in ~/Downloads")
	}

	t.Logf("File size: %d bytes (%.1f MB)", len(data), float64(len(data))/1024/1024)

	// Parse with skip errors (most fields use Template 5.3 complex packing)
	fields, err := ReadWithOptions(data, WithSequential(), WithSkipErrors())
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	t.Logf("Parsed %d fields", len(fields))

	// Verify Template 5.3 and 4.8 support is working
	// Without Template 5.3/4.8: 44 fields (only Template 5.0/4.0)
	// With Template 5.3 and 4.8: 276+ fields (39% of 708 total)
	// Remaining ~432 messages likely have edge cases in Template 5.3 complex packing
	if len(fields) < 250 {
		t.Errorf("Expected at least 250 fields (Template 5.0/4.0 + Template 5.3/4.8), got %d", len(fields))
	}

	t.Logf("Parsing coverage: %d/%d messages (%.1f%%)", len(fields), 708, 100.0*float64(len(fields))/708.0)

	if len(fields) > 0 {
		f := fields[0]
		t.Logf("First field: %s at %s", f.ParameterName, f.Level)
		t.Logf("  Center: %s", f.Center)
		t.Logf("  Grid type: %s", f.GridType)
		t.Logf("  Grid points: %d", f.NumPoints)
		t.Logf("  Valid values: %d", f.CountValid())

		// Verify CONUS has much larger grid than Iowa subset
		if f.NumPoints < 1_000_000 {
			t.Errorf("Expected CONUS grid to have >1M points, got %d", f.NumPoints)
		}
	}
}
