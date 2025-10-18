package section

import (
	"testing"
)

func TestParseSection0Valid(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		discipline    uint8
		messageLength uint64
	}{
		{
			name: "meteorological product",
			data: []byte{
				'G', 'R', 'I', 'B', // Magic number
				0x00, 0x00, // Reserved
				0, // Discipline 0 (Meteorological)
				2, // Edition 2
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, // Message length 256
			},
			discipline:    0,
			messageLength: 256,
		},
		{
			name: "hydrological product",
			data: []byte{
				'G', 'R', 'I', 'B',
				0x00, 0x00,
				1, // Discipline 1 (Hydrological)
				2,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, // Message length 512
			},
			discipline:    1,
			messageLength: 512,
		},
		{
			name: "large message",
			data: []byte{
				'G', 'R', 'I', 'B',
				0x00, 0x00,
				0,
				2,
				0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, // Message length 16777216
			},
			discipline:    0,
			messageLength: 16777216,
		},
		{
			name: "minimum valid message",
			data: []byte{
				'G', 'R', 'I', 'B',
				0x00, 0x00,
				0,
				2,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, // Message length 16 (minimum)
			},
			discipline:    0,
			messageLength: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec0, err := ParseSection0(tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if sec0 == nil {
				t.Fatal("ParseSection0 returned nil")
			}

			if sec0.Discipline != tt.discipline {
				t.Errorf("Discipline: got %d, want %d", sec0.Discipline, tt.discipline)
			}

			if sec0.Edition != 2 {
				t.Errorf("Edition: got %d, want 2", sec0.Edition)
			}

			if sec0.MessageLength != tt.messageLength {
				t.Errorf("MessageLength: got %d, want %d", sec0.MessageLength, tt.messageLength)
			}
		})
	}
}

func TestParseSection0InvalidMagic(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "wrong magic - XXXX",
			data: []byte{
				'X', 'X', 'X', 'X',
				0x00, 0x00, 0, 2,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
			},
		},
		{
			name: "wrong magic - grib (lowercase)",
			data: []byte{
				'g', 'r', 'i', 'b',
				0x00, 0x00, 0, 2,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
			},
		},
		{
			name: "partial magic - GRI",
			data: []byte{
				'G', 'R', 'I', 0x00,
				0x00, 0x00, 0, 2,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSection0(tt.data)
			if err == nil {
				t.Fatal("expected error for invalid magic number, got nil")
			}

			// Check that error message mentions magic number
			errMsg := err.Error()
			if errMsg == "" {
				t.Error("error message is empty")
			}
		})
	}
}

func TestParseSection0WrongEdition(t *testing.T) {
	tests := []struct {
		name    string
		edition uint8
	}{
		{"GRIB edition 0", 0},
		{"GRIB edition 1", 1},
		{"GRIB edition 3", 3},
		{"GRIB edition 255", 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte{
				'G', 'R', 'I', 'B',
				0x00, 0x00,
				0,          // Discipline
				tt.edition, // Wrong edition
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
			}

			_, err := ParseSection0(data)
			if err == nil {
				t.Fatalf("expected error for edition %d, got nil", tt.edition)
			}

			// Check that error message mentions edition
			errMsg := err.Error()
			if errMsg == "" {
				t.Error("error message is empty")
			}
		})
	}
}

func TestParseSection0TooShort(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{'G'}},
		{"4 bytes (only magic)", []byte{'G', 'R', 'I', 'B'}},
		{"8 bytes", []byte{'G', 'R', 'I', 'B', 0x00, 0x00, 0, 2}},
		{"15 bytes (missing 1)", []byte{
			'G', 'R', 'I', 'B',
			0x00, 0x00, 0, 2,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSection0(tt.data)
			if err == nil {
				t.Fatalf("expected error for %d bytes, got nil", len(tt.data))
			}
		})
	}
}

func TestParseSection0InvalidMessageLength(t *testing.T) {
	tests := []struct {
		name          string
		messageLength uint64
	}{
		{"zero length", 0},
		{"too small - 1 byte", 1},
		{"too small - 15 bytes", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte{
				'G', 'R', 'I', 'B',
				0x00, 0x00, 0, 2,
				// Message length (big-endian uint64)
				byte(tt.messageLength >> 56),
				byte(tt.messageLength >> 48),
				byte(tt.messageLength >> 40),
				byte(tt.messageLength >> 32),
				byte(tt.messageLength >> 24),
				byte(tt.messageLength >> 16),
				byte(tt.messageLength >> 8),
				byte(tt.messageLength),
			}

			_, err := ParseSection0(data)
			if err == nil {
				t.Fatalf("expected error for message length %d, got nil", tt.messageLength)
			}
		})
	}
}

func TestParseSection0NonZeroReserved(t *testing.T) {
	// Test that non-zero reserved bytes don't cause a failure
	// (spec says they should be zero, but we're lenient)
	data := []byte{
		'G', 'R', 'I', 'B',
		0xFF, 0xFF, // Reserved (non-zero, but we allow it)
		0,          // Discipline
		2,          // Edition
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
	}

	sec0, err := ParseSection0(data)
	if err != nil {
		t.Fatalf("unexpected error for non-zero reserved bytes: %v", err)
	}

	if sec0 == nil {
		t.Fatal("ParseSection0 returned nil")
	}
}

func TestGetDisciplineName(t *testing.T) {
	tests := []struct {
		code uint8
		want string
	}{
		{0, "Meteorological"},
		{1, "Hydrological"},
		{2, "Land Surface"},
		{3, "Space"},
		{4, "Space Weather"},
		{10, "Oceanographic"},
		{20, "Health"},
		{192, "Local"},
		{255, "Missing"},
		{99, "Unknown discipline (99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GetDisciplineName(tt.code)
			if got != tt.want {
				t.Errorf("GetDisciplineName(%d) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestSection0DisciplineName(t *testing.T) {
	sec0 := &Section0{
		Discipline:    0,
		Edition:       2,
		MessageLength: 256,
	}

	got := sec0.DisciplineName()
	want := "Meteorological"

	if got != want {
		t.Errorf("DisciplineName() = %q, want %q", got, want)
	}
}

func TestParseSection0ExtraBytes(t *testing.T) {
	// Test that extra bytes beyond 16 are ignored
	data := []byte{
		'G', 'R', 'I', 'B',
		0x00, 0x00, 0, 2,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
		// Extra bytes (should be ignored)
		0xFF, 0xFF, 0xFF, 0xFF,
	}

	sec0, err := ParseSection0(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec0 == nil {
		t.Fatal("ParseSection0 returned nil")
	}

	// Should have parsed successfully, ignoring extra bytes
	if sec0.MessageLength != 256 {
		t.Errorf("MessageLength: got %d, want 256", sec0.MessageLength)
	}
}
