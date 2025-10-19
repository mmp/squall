package section

import (
	"bytes"
	"testing"
)

func makeSection7Data(packedData []byte) []byte {
	// Total: 5 (header) + len(packedData)
	sectionLen := 5 + len(packedData)
	data := make([]byte, sectionLen)

	// Section length
	data[0] = byte(sectionLen >> 24)
	data[1] = byte(sectionLen >> 16)
	data[2] = byte(sectionLen >> 8)
	data[3] = byte(sectionLen)

	// Section number (7)
	data[4] = 7

	// Copy packed data
	copy(data[5:], packedData)

	return data
}

func TestParseSection7WithData(t *testing.T) {
	// Create some packed data
	packedData := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	data := makeSection7Data(packedData)

	sec7, err := ParseSection7(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sec7.Length != uint32(len(data)) {
		t.Errorf("Length: got %d, want %d", sec7.Length, len(data))
	}

	if !bytes.Equal(sec7.Data, packedData) {
		t.Errorf("Data mismatch: got %v, want %v", sec7.Data, packedData)
	}
}

func TestParseSection7Empty(t *testing.T) {
	// Section 7 with no data
	data := makeSection7Data([]byte{})

	sec7, err := ParseSection7(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sec7.Data) != 0 {
		t.Errorf("Data length: got %d, want 0", len(sec7.Data))
	}
}

func TestParseSection7LargeData(t *testing.T) {
	// Large packed data (10 KB)
	packedData := make([]byte, 10*1024)
	for i := range packedData {
		packedData[i] = byte(i % 256)
	}

	data := makeSection7Data(packedData)

	sec7, err := ParseSection7(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sec7.Data) != len(packedData) {
		t.Errorf("Data length: got %d, want %d", len(sec7.Data), len(packedData))
	}

	if !bytes.Equal(sec7.Data, packedData) {
		t.Error("Large data mismatch")
	}
}

func TestParseSection7TooShort(t *testing.T) {
	data := make([]byte, 3)
	_, err := ParseSection7(data)
	if err == nil {
		t.Fatal("expected error for too short section, got nil")
	}
}

func TestParseSection7WrongSectionNumber(t *testing.T) {
	data := makeSection7Data([]byte{1, 2, 3})
	data[4] = 6 // Change to section 6

	_, err := ParseSection7(data)
	if err == nil {
		t.Fatal("expected error for wrong section number, got nil")
	}
}

func TestParseSection7LengthMismatch(t *testing.T) {
	data := makeSection7Data([]byte{1, 2, 3})
	// Change length to incorrect value
	data[3] = 100

	_, err := ParseSection7(data)
	if err == nil {
		t.Fatal("expected error for length mismatch, got nil")
	}
}

func TestSection7String(t *testing.T) {
	sec7 := &Section7{
		Length: 100,
		Data:   make([]byte, 95),
	}

	str := sec7.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}
