package mgrib2

import (
	"os"
	"testing"
)

func TestParseHRRRFile(t *testing.T) {
	data, err := os.ReadFile("testdata/hrrr-iowa-subset.grib2")
	if err != nil {
		t.Skip("Test file not found")
	}

	t.Logf("File size: %d bytes", len(data))

	// Parse with skip errors (some templates not yet supported)
	// Use sequential for now (parallel + skipErrors not yet implemented)
	fields, err := ReadWithOptions(data, WithSequential(), WithSkipErrors())
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	t.Logf("Parsed %d fields", len(fields))

	// Verify we parsed a significant number of fields (96% of 708 total)
	if len(fields) < 650 {
		t.Errorf("Expected at least 650 fields, got %d", len(fields))
	}

	if len(fields) > 0 {
		f := fields[0]
		t.Logf("First field: %s at %s", f.ParameterName, f.Level)
		t.Logf("  Center: %s", f.Center)
		t.Logf("  Grid type: %s", f.GridType)
		t.Logf("  Grid points: %d", f.NumPoints)
		t.Logf("  Valid values: %d", f.CountValid())
	}
}
