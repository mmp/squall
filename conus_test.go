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

	t.Logf("Parsed %d fields (Template 5.0 simple packing)", len(fields))
	t.Logf("Note: ~664 fields skipped (use Template 5.3 complex packing, not yet implemented)")

	// Verify we parsed some fields successfully
	if len(fields) < 10 {
		t.Errorf("Expected at least 10 fields with simple packing, got %d", len(fields))
	}

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
