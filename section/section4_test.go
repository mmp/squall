package section

import (
	"testing"
)

func makeSection4Template40Data(paramCategory, paramNumber, surfaceType uint8, surfaceValue uint32) []byte {
	// Create Section 4 with Template 4.0
	// Total: 9 (header) + 34 (template) = 43 bytes
	data := make([]byte, 43)

	// Section length (43 bytes)
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x2B // 43 in hex

	// Section number (4)
	data[4] = 4

	// Number of coordinate values (0)
	data[5] = 0x00
	data[6] = 0x00

	// Product definition template number (0)
	data[7] = 0x00
	data[8] = 0x00

	// Template 4.0 data starts at byte 9
	// Parameter category
	data[9] = paramCategory

	// Parameter number
	data[10] = paramNumber

	// Generating process type (0)
	data[11] = 0

	// Background process (0)
	data[12] = 0

	// Forecast process (0)
	data[13] = 0

	// Hours after cutoff (0)
	data[14] = 0x00
	data[15] = 0x00

	// Minutes after cutoff (0)
	data[16] = 0

	// Time range unit (1 = hour)
	data[17] = 1

	// Forecast time (0)
	data[18] = 0x00
	data[19] = 0x00
	data[20] = 0x00
	data[21] = 0x00

	// First surface type
	data[22] = surfaceType

	// First surface scale factor (0)
	data[23] = 0

	// First surface value
	data[24] = byte(surfaceValue >> 24)
	data[25] = byte(surfaceValue >> 16)
	data[26] = byte(surfaceValue >> 8)
	data[27] = byte(surfaceValue)

	// Second surface type (255 = missing)
	data[28] = 255

	// Second surface scale factor (0)
	data[29] = 0

	// Second surface value (0)
	data[30] = 0x00
	data[31] = 0x00
	data[32] = 0x00
	data[33] = 0x00

	return data
}

func TestParseSection4Template40(t *testing.T) {
	// Temperature (category 0, number 0) at 500 mb (type 100, value 50000)
	data := makeSection4Template40Data(0, 0, 100, 50000)

	sec4, err := ParseSection4(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec4.Length != 43 {
		t.Errorf("Length: got %d, want 43", sec4.Length)
	}

	if sec4.ProductDefinitionTemplate != 0 {
		t.Errorf("ProductDefinitionTemplate: got %d, want 0", sec4.ProductDefinitionTemplate)
	}

	if sec4.Product == nil {
		t.Fatal("Product should not be nil")
	}

	if sec4.Product.TemplateNumber() != 0 {
		t.Errorf("Product.TemplateNumber() = %d, want 0", sec4.Product.TemplateNumber())
	}

	if sec4.Product.GetParameterCategory() != 0 {
		t.Errorf("GetParameterCategory() = %d, want 0", sec4.Product.GetParameterCategory())
	}

	if sec4.Product.GetParameterNumber() != 0 {
		t.Errorf("GetParameterNumber() = %d, want 0", sec4.Product.GetParameterNumber())
	}
}

func TestParseSection4TooShort(t *testing.T) {
	data := make([]byte, 5)
	_, err := ParseSection4(data)
	if err == nil {
		t.Fatal("expected error for too short section, got nil")
	}
}

func TestParseSection4WrongSectionNumber(t *testing.T) {
	data := makeSection4Template40Data(0, 0, 100, 50000)
	data[4] = 5 // Change to section 5

	_, err := ParseSection4(data)
	if err == nil {
		t.Fatal("expected error for wrong section number, got nil")
	}
}

func TestParseSection4UnsupportedTemplate(t *testing.T) {
	data := makeSection4Template40Data(0, 0, 100, 50000)
	// Change template number to 999 (unsupported)
	data[7] = 0x03
	data[8] = 0xE7

	_, err := ParseSection4(data)
	if err == nil {
		t.Fatal("expected error for unsupported template, got nil")
	}
}

func TestParseSection4DifferentParameters(t *testing.T) {
	tests := []struct {
		name         string
		category     uint8
		number       uint8
		surfaceType  uint8
		surfaceValue uint32
	}{
		{"Temperature at 500mb", 0, 0, 100, 50000},
		{"Relative Humidity at surface", 1, 1, 1, 0},
		{"Wind Speed at 850mb", 2, 1, 100, 85000},
		{"Geopotential Height at 1000mb", 3, 5, 100, 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := makeSection4Template40Data(tt.category, tt.number, tt.surfaceType, tt.surfaceValue)

			sec4, err := ParseSection4(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if sec4.Product.GetParameterCategory() != tt.category {
				t.Errorf("GetParameterCategory() = %d, want %d", sec4.Product.GetParameterCategory(), tt.category)
			}

			if sec4.Product.GetParameterNumber() != tt.number {
				t.Errorf("GetParameterNumber() = %d, want %d", sec4.Product.GetParameterNumber(), tt.number)
			}
		})
	}
}
