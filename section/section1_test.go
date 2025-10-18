package section

import (
	"fmt"
	"testing"
	"time"
)

// Helper to create a valid Section 1
func makeSection1Data(year uint16, month, day, hour, minute, second uint8) []byte {
	data := make([]byte, 21)

	// Length (21 bytes)
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x15 // 21 in hex

	// Section number (1)
	data[4] = 1

	// Originating center (7 = NCEP)
	data[5] = 0x00
	data[6] = 0x07

	// Originating sub-center (0)
	data[7] = 0x00
	data[8] = 0x00

	// Master tables version (2)
	data[9] = 2

	// Local tables version (1)
	data[10] = 1

	// Significance of reference time (1 = start of forecast)
	data[11] = 1

	// Year (big-endian)
	data[12] = byte(year >> 8)
	data[13] = byte(year)

	// Month, day, hour, minute, second
	data[14] = month
	data[15] = day
	data[16] = hour
	data[17] = minute
	data[18] = second

	// Production status (0 = operational)
	data[19] = 0

	// Type of data (1 = forecast)
	data[20] = 1

	return data
}

func TestParseSection1Valid(t *testing.T) {
	data := makeSection1Data(2024, 10, 18, 12, 30, 0)

	sec1, err := ParseSection1(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec1.Length != 21 {
		t.Errorf("Length: got %d, want 21", sec1.Length)
	}

	if sec1.OriginatingCenter != 7 {
		t.Errorf("OriginatingCenter: got %d, want 7", sec1.OriginatingCenter)
	}

	if sec1.MasterTablesVersion != 2 {
		t.Errorf("MasterTablesVersion: got %d, want 2", sec1.MasterTablesVersion)
	}

	if sec1.LocalTablesVersion != 1 {
		t.Errorf("LocalTablesVersion: got %d, want 1", sec1.LocalTablesVersion)
	}

	if sec1.SignificanceOfRefTime != 1 {
		t.Errorf("SignificanceOfRefTime: got %d, want 1", sec1.SignificanceOfRefTime)
	}

	expectedTime := time.Date(2024, 10, 18, 12, 30, 0, 0, time.UTC)
	if !sec1.ReferenceTime.Equal(expectedTime) {
		t.Errorf("ReferenceTime: got %v, want %v", sec1.ReferenceTime, expectedTime)
	}

	if sec1.ProductionStatus != 0 {
		t.Errorf("ProductionStatus: got %d, want 0", sec1.ProductionStatus)
	}

	if sec1.TypeOfData != 1 {
		t.Errorf("TypeOfData: got %d, want 1", sec1.TypeOfData)
	}
}

func TestParseSection1TooShort(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"10 bytes", make([]byte, 10)},
		{"20 bytes", make([]byte, 20)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSection1(tt.data)
			if err == nil {
				t.Errorf("expected error for %d bytes, got nil", len(tt.data))
			}
		})
	}
}

func TestParseSection1WrongSectionNumber(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 0, 0, 0)
	data[4] = 2 // Change section number to 2

	_, err := ParseSection1(data)
	if err == nil {
		t.Fatal("expected error for wrong section number, got nil")
	}
}

func TestParseSection1LengthMismatch(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 0, 0, 0)
	// Change length to 100
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x64

	_, err := ParseSection1(data)
	if err == nil {
		t.Fatal("expected error for length mismatch, got nil")
	}
}

func TestParseSection1InvalidMonth(t *testing.T) {
	tests := []uint8{0, 13, 255}

	for _, month := range tests {
		t.Run(fmt.Sprintf("month=%d", month), func(t *testing.T) {
			data := makeSection1Data(2024, month, 1, 0, 0, 0)
			_, err := ParseSection1(data)
			if err == nil {
				t.Errorf("expected error for month %d, got nil", month)
			}
		})
	}
}

func TestParseSection1InvalidDay(t *testing.T) {
	tests := []uint8{0, 32, 255}

	for _, day := range tests {
		t.Run(fmt.Sprintf("day=%d", day), func(t *testing.T) {
			data := makeSection1Data(2024, 1, day, 0, 0, 0)
			_, err := ParseSection1(data)
			if err == nil {
				t.Errorf("expected error for day %d, got nil", day)
			}
		})
	}
}

func TestParseSection1InvalidHour(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 24, 0, 0)
	_, err := ParseSection1(data)
	if err == nil {
		t.Fatal("expected error for hour 24, got nil")
	}
}

func TestParseSection1InvalidMinute(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 0, 60, 0)
	_, err := ParseSection1(data)
	if err == nil {
		t.Fatal("expected error for minute 60, got nil")
	}
}

func TestParseSection1InvalidSecond(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 0, 0, 60)
	_, err := ParseSection1(data)
	if err == nil {
		t.Fatal("expected error for second 60, got nil")
	}
}

func TestParseSection1EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		year   uint16
		month  uint8
		day    uint8
		hour   uint8
		minute uint8
		second uint8
	}{
		{"midnight", 2024, 1, 1, 0, 0, 0},
		{"end of day", 2024, 1, 1, 23, 59, 59},
		{"leap year", 2024, 2, 29, 12, 0, 0},
		{"year 1", 1, 1, 1, 0, 0, 0},
		{"year 9999", 9999, 12, 31, 23, 59, 59},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := makeSection1Data(tt.year, tt.month, tt.day, tt.hour, tt.minute, tt.second)
			sec1, err := ParseSection1(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedTime := time.Date(int(tt.year), time.Month(tt.month), int(tt.day),
				int(tt.hour), int(tt.minute), int(tt.second), 0, time.UTC)

			if !sec1.ReferenceTime.Equal(expectedTime) {
				t.Errorf("ReferenceTime: got %v, want %v", sec1.ReferenceTime, expectedTime)
			}
		})
	}
}

func TestSection1CenterName(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 0, 0, 0)
	// Set center to NCEP (7)
	data[5] = 0x00
	data[6] = 0x07

	sec1, err := ParseSection1(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name := sec1.CenterName(); name != "NCEP" {
		t.Errorf("CenterName() = %q, want \"NCEP\"", name)
	}
}

func TestSection1TimeSignificanceName(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 0, 0, 0)
	// Set significance to "Analysis" (0)
	data[11] = 0

	sec1, err := ParseSection1(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name := sec1.TimeSignificanceName(); name != "Analysis" {
		t.Errorf("TimeSignificanceName() = %q, want \"Analysis\"", name)
	}
}

func TestSection1ProductionStatusName(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 0, 0, 0)
	// Set production status to "Operational" (0)
	data[19] = 0

	sec1, err := ParseSection1(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name := sec1.ProductionStatusName(); name != "Operational" {
		t.Errorf("ProductionStatusName() = %q, want \"Operational\"", name)
	}
}

func TestSection1DataTypeName(t *testing.T) {
	data := makeSection1Data(2024, 1, 1, 0, 0, 0)
	// Set data type to "Forecast" (1)
	data[20] = 1

	sec1, err := ParseSection1(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name := sec1.DataTypeName(); name != "Forecast" {
		t.Errorf("DataTypeName() = %q, want \"Forecast\"", name)
	}
}

func TestParseSection1DifferentCenters(t *testing.T) {
	tests := []struct {
		center uint16
		name   string
	}{
		{7, "NCEP"},
		{98, "ECMWF"},
		{34, "JMA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := makeSection1Data(2024, 1, 1, 0, 0, 0)
			data[5] = byte(tt.center >> 8)
			data[6] = byte(tt.center)

			sec1, err := ParseSection1(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if sec1.OriginatingCenter != tt.center {
				t.Errorf("OriginatingCenter: got %d, want %d", sec1.OriginatingCenter, tt.center)
			}

			if name := sec1.CenterName(); name != tt.name {
				t.Errorf("CenterName() = %q, want %q", name, tt.name)
			}
		})
	}
}
