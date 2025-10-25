package squall

import (
	"bytes"
	"os"
	"testing"
)

func TestParseHRRRFile(t *testing.T) {
	data, err := os.ReadFile("testgribs/hrrr-iowa-subset.grib2")
	if err != nil {
		t.Skip("Test file not found - run from repository root or place test file in testgribs/")
	}

	t.Logf("File size: %d bytes", len(data))

	// Parse all fields - Template 5.3 (complex packing) is now fully supported
	fields, err := ReadWithOptions(bytes.NewReader(data), WithSequential())
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	t.Logf("Parsed %d fields", len(fields))

	// Verify we parsed all 708 fields
	if len(fields) != 708 {
		t.Errorf("Expected 708 fields, got %d", len(fields))
	}

	if len(fields) > 0 {
		f := fields[0]
		t.Logf("First field: %s at %s", f.Parameter.String(), f.Level)
		t.Logf("  Center: %s", f.Center)
		t.Logf("  Grid type: %s", f.GridType)
		t.Logf("  Grid points: %d", f.NumPoints)
		t.Logf("  Valid values: %d", f.CountValid())
	}
}
