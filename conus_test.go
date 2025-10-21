package mgrib2

import (
	"bytes"
	"os"
	"testing"
)

func TestParseHRRRCONUS(t *testing.T) {
	// Test with full CONUS HRRR file (much larger than Iowa subset)
	// Grid: 1799x1059 = 1,905,141 points per field (vs 22,632 for Iowa)
	// File: 357 MB (vs 12.5 MB for Iowa)
	data, err := os.ReadFile("testgribs/hrrr.20251015-conus-hrrr.t11z.wrfprsf00.grib2")
	if err != nil {
		t.Skip("CONUS HRRR file not found - place in testgribs/ or skip this test")
	}

	t.Logf("File size: %d bytes (%.1f MB)", len(data), float64(len(data))/1024/1024)

	// Parse all fields - Template 5.3 (complex packing) is now fully supported
	fields, err := ReadWithOptions(bytes.NewReader(data), WithSequential())
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	t.Logf("Parsed %d fields", len(fields))

	// Verify all 708 messages were parsed successfully
	if len(fields) != 708 {
		t.Errorf("Expected 708 fields, got %d", len(fields))
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
