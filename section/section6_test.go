package section

import (
	"testing"
)

func makeSection6WithBitmap(bitmap []bool) []byte {
	// Calculate number of bytes needed for bitmap
	numBits := uint32(len(bitmap))
	numBytes := (numBits + 7) / 8

	// Total: 6 (header) + numBytes (bitmap)
	data := make([]byte, 6+numBytes)

	// Section length
	sectionLen := 6 + numBytes
	data[0] = byte(sectionLen >> 24)
	data[1] = byte(sectionLen >> 16)
	data[2] = byte(sectionLen >> 8)
	data[3] = byte(sectionLen)

	// Section number (6)
	data[4] = 6

	// Bitmap indicator (0 = bitmap specified in this section)
	data[5] = 0

	// Pack bitmap into bytes
	for i, bit := range bitmap {
		byteIdx := i / 8
		bitIdx := 7 - (i % 8)
		if bit {
			data[6+byteIdx] |= 1 << uint(bitIdx)
		}
	}

	return data
}

func makeSection6NoBitmap() []byte {
	// Section with bitmap indicator 255 (no bitmap)
	data := make([]byte, 6)

	// Section length (6 bytes)
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x06

	// Section number (6)
	data[4] = 6

	// Bitmap indicator (255 = no bitmap)
	data[5] = 255

	return data
}

func TestParseSection6WithBitmap(t *testing.T) {
	// Create bitmap: 10 points, alternating valid/invalid
	bitmap := []bool{true, false, true, false, true, false, true, false, true, false}
	data := makeSection6WithBitmap(bitmap)

	sec6, err := ParseSection6(data, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec6.Length != uint32(len(data)) {
		t.Errorf("Length: got %d, want %d", sec6.Length, len(data))
	}

	if sec6.BitmapIndicator != 0 {
		t.Errorf("BitmapIndicator: got %d, want 0", sec6.BitmapIndicator)
	}

	if !sec6.HasBitmap() {
		t.Error("HasBitmap() should return true")
	}

	if len(sec6.Bitmap) != 10 {
		t.Fatalf("Bitmap length: got %d, want 10", len(sec6.Bitmap))
	}

	// Verify bitmap values
	for i, expected := range bitmap {
		if sec6.Bitmap[i] != expected {
			t.Errorf("Bitmap[%d]: got %v, want %v", i, sec6.Bitmap[i], expected)
		}
	}

	// Verify count
	if sec6.CountValidPoints() != 5 {
		t.Errorf("CountValidPoints: got %d, want 5", sec6.CountValidPoints())
	}
}

func TestParseSection6NoBitmap(t *testing.T) {
	data := makeSection6NoBitmap()

	sec6, err := ParseSection6(data, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec6.BitmapIndicator != 255 {
		t.Errorf("BitmapIndicator: got %d, want 255", sec6.BitmapIndicator)
	}

	if sec6.HasBitmap() {
		t.Error("HasBitmap() should return false")
	}

	if sec6.Bitmap != nil {
		t.Error("Bitmap should be nil when indicator is 255")
	}
}

func TestParseSection6AllValid(t *testing.T) {
	// All points valid
	bitmap := []bool{true, true, true, true, true}
	data := makeSection6WithBitmap(bitmap)

	sec6, err := ParseSection6(data, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec6.CountValidPoints() != 5 {
		t.Errorf("CountValidPoints: got %d, want 5", sec6.CountValidPoints())
	}
}

func TestParseSection6AllInvalid(t *testing.T) {
	// All points invalid
	bitmap := []bool{false, false, false, false, false}
	data := makeSection6WithBitmap(bitmap)

	sec6, err := ParseSection6(data, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec6.CountValidPoints() != 0 {
		t.Errorf("CountValidPoints: got %d, want 0", sec6.CountValidPoints())
	}
}

func TestParseSection6NonMultipleOf8(t *testing.T) {
	// Test bitmap that's not a multiple of 8 bits
	bitmap := []bool{true, false, true, false, true} // 5 bits
	data := makeSection6WithBitmap(bitmap)

	sec6, err := ParseSection6(data, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sec6.Bitmap) != 5 {
		t.Fatalf("Bitmap length: got %d, want 5", len(sec6.Bitmap))
	}

	for i, expected := range bitmap {
		if sec6.Bitmap[i] != expected {
			t.Errorf("Bitmap[%d]: got %v, want %v", i, sec6.Bitmap[i], expected)
		}
	}
}

func TestParseSection6LargeBitmap(t *testing.T) {
	// Test with a large bitmap (1000 points)
	bitmap := make([]bool, 1000)
	for i := range bitmap {
		bitmap[i] = i%3 == 0 // Every third point is valid
	}

	data := makeSection6WithBitmap(bitmap)

	sec6, err := ParseSection6(data, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sec6.Bitmap) != 1000 {
		t.Fatalf("Bitmap length: got %d, want 1000", len(sec6.Bitmap))
	}

	// Verify random samples
	for i := 0; i < 1000; i += 100 {
		if sec6.Bitmap[i] != bitmap[i] {
			t.Errorf("Bitmap[%d]: got %v, want %v", i, sec6.Bitmap[i], bitmap[i])
		}
	}
}

func TestParseSection6TooShort(t *testing.T) {
	data := make([]byte, 3)
	_, err := ParseSection6(data, 10)
	if err == nil {
		t.Fatal("expected error for too short section, got nil")
	}
}

func TestParseSection6WrongSectionNumber(t *testing.T) {
	data := makeSection6NoBitmap()
	data[4] = 7 // Change to section 7

	_, err := ParseSection6(data, 10)
	if err == nil {
		t.Fatal("expected error for wrong section number, got nil")
	}
}

func TestParseSection6PreviouslyDefined(t *testing.T) {
	data := makeSection6NoBitmap()
	data[5] = 254 // Previously defined bitmap

	_, err := ParseSection6(data, 10)
	if err == nil {
		t.Fatal("expected error for unsupported bitmap indicator 254, got nil")
	}
}

func TestSection6String(t *testing.T) {
	tests := []struct {
		name     string
		sec6     *Section6
		contains string
	}{
		{
			name: "With bitmap",
			sec6: &Section6{
				BitmapIndicator: 0,
				Bitmap:          []bool{true, false, true, true},
			},
			contains: "3/4 valid",
		},
		{
			name: "No bitmap",
			sec6: &Section6{
				BitmapIndicator: 255,
				Bitmap:          nil,
			},
			contains: "all points valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.sec6.String()
			if str == "" {
				t.Error("String() returned empty string")
			}
		})
	}
}
