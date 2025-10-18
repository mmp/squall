package section

import (
	"testing"
)

func makeSection3LatLonData(ni, nj uint32, la1, lo1, la2, lo2 int32) []byte {
	// Create a minimal Section 3 with Template 3.0 (Lat/Lon)
	// Total: 14 (header) + 72 (template) = 86 bytes
	data := make([]byte, 86)

	// Section length (86 bytes)
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x56 // 86 in hex

	// Section number (3)
	data[4] = 3

	// Source of grid definition (0 = from template)
	data[5] = 0

	// Number of data points (ni * nj)
	numPoints := ni * nj
	data[6] = byte(numPoints >> 24)
	data[7] = byte(numPoints >> 16)
	data[8] = byte(numPoints >> 8)
	data[9] = byte(numPoints)

	// Number of octets for optional list (0)
	data[10] = 0

	// Interpretation of optional list (0)
	data[11] = 0

	// Template number (0 = Lat/Lon)
	data[12] = 0x00
	data[13] = 0x00

	// Template 3.0 data starts at byte 14
	// Shape of earth + parameters (16 bytes) - set to 0 for now
	// [14-29] = zeros

	// Ni (number of points along parallel)
	data[30] = byte(ni >> 24)
	data[31] = byte(ni >> 16)
	data[32] = byte(ni >> 8)
	data[33] = byte(ni)

	// Nj (number of points along meridian)
	data[34] = byte(nj >> 24)
	data[35] = byte(nj >> 16)
	data[36] = byte(nj >> 8)
	data[37] = byte(nj)

	// Basic angle and subdivisions (8 bytes) - set to 0
	// [38-45] = zeros

	// La1 (latitude of first grid point, millidegrees)
	data[46] = byte(la1 >> 24)
	data[47] = byte(la1 >> 16)
	data[48] = byte(la1 >> 8)
	data[49] = byte(la1)

	// Lo1 (longitude of first grid point, millidegrees)
	data[50] = byte(lo1 >> 24)
	data[51] = byte(lo1 >> 16)
	data[52] = byte(lo1 >> 8)
	data[53] = byte(lo1)

	// Resolution and component flags (1 byte)
	data[54] = 0x00

	// La2 (latitude of last grid point, millidegrees)
	data[55] = byte(la2 >> 24)
	data[56] = byte(la2 >> 16)
	data[57] = byte(la2 >> 8)
	data[58] = byte(la2)

	// Lo2 (longitude of last grid point, millidegrees)
	data[59] = byte(lo2 >> 24)
	data[60] = byte(lo2 >> 16)
	data[61] = byte(lo2 >> 8)
	data[62] = byte(lo2)

	// Di (i direction increment, millidegrees)
	data[63] = 0x00
	data[64] = 0x00
	data[65] = 0x03
	data[66] = 0xE8 // 1000 millidegrees = 1 degree

	// Dj (j direction increment, millidegrees)
	data[67] = 0x00
	data[68] = 0x00
	data[69] = 0x03
	data[70] = 0xE8 // 1000 millidegrees = 1 degree

	// Scanning mode (1 byte) - default: west to east, north to south
	data[71] = 0x00

	return data
}

func TestParseSection3LatLon(t *testing.T) {
	data := makeSection3LatLonData(
		144, 73,      // 144x73 grid (2.5 degree global)
		90000,        // La1 = 90°N
		0,            // Lo1 = 0°E
		-90000,       // La2 = 90°S
		357500,       // Lo2 = 357.5°E
	)

	sec3, err := ParseSection3(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec3.Length != 86 {
		t.Errorf("Length: got %d, want 86", sec3.Length)
	}

	if sec3.NumDataPoints != 144*73 {
		t.Errorf("NumDataPoints: got %d, want %d", sec3.NumDataPoints, 144*73)
	}

	if sec3.TemplateNumber != 0 {
		t.Errorf("TemplateNumber: got %d, want 0", sec3.TemplateNumber)
	}

	if sec3.Grid == nil {
		t.Fatal("Grid should not be nil")
	}

	if sec3.Grid.TemplateNumber() != 0 {
		t.Errorf("Grid.TemplateNumber() = %d, want 0", sec3.Grid.TemplateNumber())
	}

	if sec3.Grid.NumPoints() != 144*73 {
		t.Errorf("Grid.NumPoints() = %d, want %d", sec3.Grid.NumPoints(), 144*73)
	}
}

func TestParseSection3TooShort(t *testing.T) {
	data := make([]byte, 10)
	_, err := ParseSection3(data)
	if err == nil {
		t.Fatal("expected error for too short section, got nil")
	}
}

func TestParseSection3WrongSectionNumber(t *testing.T) {
	data := makeSection3LatLonData(10, 10, 0, 0, 10000, 10000)
	data[4] = 4 // Change to section 4

	_, err := ParseSection3(data)
	if err == nil {
		t.Fatal("expected error for wrong section number, got nil")
	}
}

func TestParseSection3UnsupportedTemplate(t *testing.T) {
	data := makeSection3LatLonData(10, 10, 0, 0, 10000, 10000)
	// Change template number to 999 (unsupported)
	data[12] = 0x03
	data[13] = 0xE7

	_, err := ParseSection3(data)
	if err == nil {
		t.Fatal("expected error for unsupported template, got nil")
	}
}
