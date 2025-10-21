package section

import (
	"math"
	"testing"
)

func makeSection5Template50Data(numDataValues uint32, refValue float32, binaryScale, decimalScale int16, bitsPerValue uint8) []byte {
	// Create Section 5 with Template 5.0 (Simple Packing)
	// Total: 11 (header) + 11 (template) = 22 bytes
	data := make([]byte, 22)

	// Section length (22 bytes)
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x16 // 22 in hex

	// Section number (5)
	data[4] = 5

	// Number of data values
	data[5] = byte(numDataValues >> 24)
	data[6] = byte(numDataValues >> 16)
	data[7] = byte(numDataValues >> 8)
	data[8] = byte(numDataValues)

	// Data representation template number (0)
	data[9] = 0x00
	data[10] = 0x00

	// Template 5.0 data starts at byte 11

	// Reference value (IEEE 754 float32)
	refBits := math.Float32bits(refValue)
	data[11] = byte(refBits >> 24)
	data[12] = byte(refBits >> 16)
	data[13] = byte(refBits >> 8)
	data[14] = byte(refBits)

	// Binary scale factor (int16 sign-magnitude)
	// GRIB2 uses sign-magnitude: bit 15 = sign, bits 0-14 = magnitude
	var bsBytes uint16
	if binaryScale < 0 {
		bsBytes = 0x8000 | uint16(-binaryScale)
	} else {
		bsBytes = uint16(binaryScale)
	}
	data[15] = byte(bsBytes >> 8)
	data[16] = byte(bsBytes)

	// Decimal scale factor (int16 sign-magnitude)
	var dsBytes uint16
	if decimalScale < 0 {
		dsBytes = 0x8000 | uint16(-decimalScale)
	} else {
		dsBytes = uint16(decimalScale)
	}
	data[17] = byte(dsBytes >> 8)
	data[18] = byte(dsBytes)

	// Bits per value
	data[19] = bitsPerValue

	// Type of original field values (0 = floating point)
	data[20] = 0

	return data
}

func TestParseSection5Template50(t *testing.T) {
	// Temperature data: 100 values, reference 250.0 K, no scaling, 12 bits per value
	data := makeSection5Template50Data(100, 250.0, 0, 0, 12)

	sec5, err := ParseSection5(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec5.Length != 22 {
		t.Errorf("Length: got %d, want 22", sec5.Length)
	}

	if sec5.NumDataValues != 100 {
		t.Errorf("NumDataValues: got %d, want 100", sec5.NumDataValues)
	}

	if sec5.DataRepresentationTemplate != 0 {
		t.Errorf("DataRepresentationTemplate: got %d, want 0", sec5.DataRepresentationTemplate)
	}

	if sec5.Representation == nil {
		t.Fatal("Representation should not be nil")
	}

	if sec5.Representation.TemplateNumber() != 0 {
		t.Errorf("Representation.TemplateNumber() = %d, want 0", sec5.Representation.TemplateNumber())
	}

	if sec5.Representation.NumDataValues() != 100 {
		t.Errorf("Representation.NumDataValues() = %d, want 100", sec5.Representation.NumDataValues())
	}

	if sec5.Representation.BitsPerValue() != 12 {
		t.Errorf("Representation.BitsPerValue() = %d, want 12", sec5.Representation.BitsPerValue())
	}
}

func TestParseSection5TooShort(t *testing.T) {
	data := make([]byte, 5)
	_, err := ParseSection5(data)
	if err == nil {
		t.Fatal("expected error for too short section, got nil")
	}
}

func TestParseSection5WrongSectionNumber(t *testing.T) {
	data := makeSection5Template50Data(100, 250.0, 0, 0, 12)
	data[4] = 6 // Change to section 6

	_, err := ParseSection5(data)
	if err == nil {
		t.Fatal("expected error for wrong section number, got nil")
	}
}

func TestParseSection5UnsupportedTemplate(t *testing.T) {
	data := makeSection5Template50Data(100, 250.0, 0, 0, 12)
	// Change template number to 999 (unsupported)
	data[9] = 0x03
	data[10] = 0xE7

	_, err := ParseSection5(data)
	if err == nil {
		t.Fatal("expected error for unsupported template, got nil")
	}
}

func TestTemplate50Decode(t *testing.T) {
	tests := []struct {
		name           string
		refValue       float32
		binaryScale    int16
		decimalScale   int16
		bitsPerValue   uint8
		packedValues   []uint32
		expectedValues []float64
	}{
		{
			name:           "No scaling",
			refValue:       100.0,
			binaryScale:    0,
			decimalScale:   0,
			bitsPerValue:   8,
			packedValues:   []uint32{0, 1, 2, 10, 255},
			expectedValues: []float64{100.0, 101.0, 102.0, 110.0, 355.0},
		},
		{
			name:           "Binary scaling only",
			refValue:       0.0,
			binaryScale:    -2, // divide by 4
			decimalScale:   0,
			bitsPerValue:   8,
			packedValues:   []uint32{0, 4, 8, 16},
			expectedValues: []float64{0.0, 1.0, 2.0, 4.0},
		},
		{
			name:           "Decimal scaling only",
			refValue:       1000.0,
			binaryScale:    0,
			decimalScale:   1, // divide by 10
			bitsPerValue:   8,
			packedValues:   []uint32{0, 10, 20},
			expectedValues: []float64{100.0, 101.0, 102.0},
		},
		{
			name:           "Both scaling factors",
			refValue:       500.0,
			binaryScale:    -1, // divide by 2
			decimalScale:   1,  // divide by 10
			bitsPerValue:   8,
			packedValues:   []uint32{0, 2, 4, 6},
			expectedValues: []float64{50.0, 50.1, 50.2, 50.3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create template
			data := makeSection5Template50Data(uint32(len(tt.packedValues)),
				tt.refValue, tt.binaryScale, tt.decimalScale, tt.bitsPerValue)

			sec5, err := ParseSection5(data)
			if err != nil {
				t.Fatalf("ParseSection5 failed: %v", err)
			}

			// Pack the test values into bytes
			packedData := packValues(tt.packedValues, int(tt.bitsPerValue))

			// Decode
			values, err := sec5.Representation.Decode(packedData, nil)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			// Verify
			if len(values) != len(tt.expectedValues) {
				t.Fatalf("got %d values, want %d", len(values), len(tt.expectedValues))
			}

			for i, expected := range tt.expectedValues {
				if math.Abs(float64(values[i])-expected) > 0.001 {
					t.Errorf("value[%d]: got %g, want %g", i, values[i], expected)
				}
			}
		})
	}
}

func TestTemplate50DecodeWithBitmap(t *testing.T) {
	// Create template: reference 100.0, no scaling, 8 bits per value
	data := makeSection5Template50Data(3, 100.0, 0, 0, 8)

	sec5, err := ParseSection5(data)
	if err != nil {
		t.Fatalf("ParseSection5 failed: %v", err)
	}

	// Pack 3 values: 0, 5, 10
	packedData := packValues([]uint32{0, 5, 10}, 8)

	// Bitmap: 5 grid points, valid at positions 0, 2, 4
	bitmap := []bool{true, false, true, false, true}

	// Decode
	values, err := sec5.Representation.Decode(packedData, bitmap)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Should have 5 values (same as bitmap length)
	if len(values) != 5 {
		t.Fatalf("got %d values, want 5", len(values))
	}

	// Check values
	expected := []float64{100.0, 9.999e20, 105.0, 9.999e20, 110.0}
	for i, exp := range expected {
		if math.Abs(float64(values[i])-exp) > 1e15 {
			t.Errorf("value[%d]: got %g, want %g", i, values[i], exp)
		}
	}
}

func TestTemplate50DecodeZeroBitsPerValue(t *testing.T) {
	// All values are the reference value
	data := makeSection5Template50Data(5, 273.15, 0, 0, 0)

	sec5, err := ParseSection5(data)
	if err != nil {
		t.Fatalf("ParseSection5 failed: %v", err)
	}

	// No packed data needed
	values, err := sec5.Representation.Decode([]byte{}, nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(values) != 5 {
		t.Fatalf("got %d values, want 5", len(values))
	}

	for i, val := range values {
		if math.Abs(float64(val)-273.15) > 0.001 {
			t.Errorf("value[%d]: got %g, want 273.15", i, val)
		}
	}
}

// Helper function to pack values into bytes at specified bit width
func packValues(values []uint32, bitsPerValue int) []byte {
	if bitsPerValue == 0 {
		return []byte{}
	}

	totalBits := len(values) * bitsPerValue
	numBytes := (totalBits + 7) / 8
	data := make([]byte, numBytes)

	bitOffset := 0
	for _, value := range values {
		// Pack value into bytes at current bit offset
		for bit := bitsPerValue - 1; bit >= 0; bit-- {
			if (value & (1 << uint(bit))) != 0 {
				byteIdx := bitOffset / 8
				bitIdx := 7 - (bitOffset % 8)
				data[byteIdx] |= 1 << uint(bitIdx)
			}
			bitOffset++
		}
	}

	return data
}
