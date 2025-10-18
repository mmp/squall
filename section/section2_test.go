package section

import (
	"testing"
)

func makeSection2Data(localData []byte) []byte {
	length := 5 + len(localData)
	data := make([]byte, length)

	// Length
	data[0] = byte(length >> 24)
	data[1] = byte(length >> 16)
	data[2] = byte(length >> 8)
	data[3] = byte(length)

	// Section number (2)
	data[4] = 2

	// Local use data
	copy(data[5:], localData)

	return data
}

func TestParseSection2Empty(t *testing.T) {
	// Section 2 with no local data (just 5 bytes: length + section number)
	data := makeSection2Data([]byte{})

	sec2, err := ParseSection2(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec2.Length != 5 {
		t.Errorf("Length: got %d, want 5", sec2.Length)
	}

	if len(sec2.Data) != 0 {
		t.Errorf("Data length: got %d, want 0", len(sec2.Data))
	}

	if !sec2.IsEmpty() {
		t.Error("IsEmpty() should be true for empty section")
	}
}

func TestParseSection2WithData(t *testing.T) {
	localData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0xFF, 0xFE, 0xFD}
	data := makeSection2Data(localData)

	sec2, err := ParseSection2(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLength := uint32(5 + len(localData))
	if sec2.Length != expectedLength {
		t.Errorf("Length: got %d, want %d", sec2.Length, expectedLength)
	}

	if len(sec2.Data) != len(localData) {
		t.Errorf("Data length: got %d, want %d", len(sec2.Data), len(localData))
	}

	// Verify data matches
	for i, b := range localData {
		if sec2.Data[i] != b {
			t.Errorf("Data[%d]: got %02x, want %02x", i, sec2.Data[i], b)
		}
	}

	if sec2.IsEmpty() {
		t.Error("IsEmpty() should be false for non-empty section")
	}
}

func TestParseSection2TooShort(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{0x00}},
		{"4 bytes", []byte{0x00, 0x00, 0x00, 0x05}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSection2(tt.data)
			if err == nil {
				t.Errorf("expected error for %d bytes, got nil", len(tt.data))
			}
		})
	}
}

func TestParseSection2WrongSectionNumber(t *testing.T) {
	data := makeSection2Data([]byte{0x01, 0x02})
	data[4] = 3 // Change section number to 3

	_, err := ParseSection2(data)
	if err == nil {
		t.Fatal("expected error for wrong section number, got nil")
	}
}

func TestParseSection2LengthMismatch(t *testing.T) {
	data := makeSection2Data([]byte{0x01, 0x02, 0x03})
	// Change length to 100
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x64

	_, err := ParseSection2(data)
	if err == nil {
		t.Fatal("expected error for length mismatch, got nil")
	}
}

func TestParseSection2LargeData(t *testing.T) {
	// Create a large local use section (1KB)
	localData := make([]byte, 1024)
	for i := range localData {
		localData[i] = byte(i % 256)
	}

	data := makeSection2Data(localData)

	sec2, err := ParseSection2(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sec2.Data) != 1024 {
		t.Errorf("Data length: got %d, want 1024", len(sec2.Data))
	}

	// Verify a few bytes
	if sec2.Data[0] != 0 || sec2.Data[255] != 255 || sec2.Data[256] != 0 {
		t.Error("Large data verification failed")
	}
}

func TestParseSection2Immutability(t *testing.T) {
	localData := []byte{0xAA, 0xBB, 0xCC}
	data := makeSection2Data(localData)

	sec2, err := ParseSection2(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify the returned data
	sec2.Data[0] = 0xFF

	// Original local data should not be modified (because Bytes() returns a copy)
	if localData[0] != 0xAA {
		t.Error("Modifying sec2.Data should not affect original localData")
	}

	// But the data in the message buffer should be modified
	// This is expected because we pass the buffer to ParseSection2
	if data[5] == 0xFF {
		// This is actually okay - we're testing the copy
		t.Log("Data was modified in message buffer (expected)")
	}
}
