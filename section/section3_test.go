package section

import (
	"testing"

	"github.com/mmp/squall/grid"
)

func makeSection3LatLonData(ni, nj uint32, la1, lo1, la2, lo2 int32) []byte {
	// Create a minimal Section 3 with Template 3.0 (Lat/Lon)
	// Total: 14 (header) + 58 (template) = 72 bytes
	data := make([]byte, 72)

	// Section length (72 bytes)
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x48 // 72 in hex

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

	// Basic angle and subdivisions (8 bytes) - set to 0, so angles are in
	// microdegrees
	// [38-45] = zeros

	// Signed values are stored in sign-magnitude form.
	put := func(i int, v int32) {
		u := uint32(v)
		if v < 0 {
			u = uint32(-v) | 0x80000000
		}
		data[i], data[i+1], data[i+2], data[i+3] = byte(u>>24), byte(u>>16), byte(u>>8), byte(u)
	}

	// La1 and Lo1 (first grid point)
	put(46, la1)
	put(50, lo1)

	// Resolution and component flags (1 byte)
	data[54] = 0x00

	// La2 and Lo2 (last grid point)
	put(55, la2)
	put(59, lo2)

	// Di and Dj (increments): 1 degree
	put(63, 1000000)
	put(67, 1000000)

	// Scanning mode (1 byte) - default: west to east, north to south
	data[71] = 0x00

	return data
}

func TestParseSection3LatLon(t *testing.T) {
	data := makeSection3LatLonData(
		144, 73, // 144x73 grid (2.5 degree global)
		90000000,  // La1 = 90°N
		0,         // Lo1 = 0°E
		-90000000, // La2 = 90°S
		357500000, // Lo2 = 357.5°E
	)

	sec3, err := ParseSection3(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec3.Length != 72 {
		t.Errorf("Length: got %d, want 72", sec3.Length)
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

	g, ok := sec3.Grid.(*grid.LatLonGrid)
	if !ok {
		t.Fatalf("Grid is %T, want *grid.LatLonGrid", sec3.Grid)
	}
	lat1, lon1 := g.FirstGridPoint()
	lat2, lon2 := g.LastGridPoint()
	if lat1 != 90 || lon1 != 0 || lat2 != -90 || lon2 != 357.5 {
		t.Errorf("corners (%v, %v) to (%v, %v), want (90, 0) to (-90, 357.5)", lat1, lon1, lat2, lon2)
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
	data := makeSection3LatLonData(10, 10, 0, 0, 9000000, 9000000)
	data[4] = 4 // Change to section 4

	_, err := ParseSection3(data)
	if err == nil {
		t.Fatal("expected error for wrong section number, got nil")
	}
}

func TestParseSection3UnsupportedTemplate(t *testing.T) {
	data := makeSection3LatLonData(10, 10, 0, 0, 9000000, 9000000)
	// Change template number to 999 (unsupported)
	data[12] = 0x03
	data[13] = 0xE7

	_, err := ParseSection3(data)
	if err == nil {
		t.Fatal("expected error for unsupported template, got nil")
	}
}
