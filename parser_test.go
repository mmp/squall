package mgrib2

import (
	"testing"
)

// Helper function to create a minimal valid GRIB2 message
func makeTestMessage(discipline uint8, messageLength uint64) []byte {
	if messageLength < 20 {
		messageLength = 20 // Minimum: 16 (Section 0) + 4 (end marker)
	}

	msg := make([]byte, messageLength)

	// Section 0 (16 bytes)
	copy(msg[0:4], "GRIB")
	msg[4] = 0x00 // Reserved
	msg[5] = 0x00 // Reserved
	msg[6] = discipline
	msg[7] = 2 // Edition

	// Message length (big-endian uint64)
	msg[8] = byte(messageLength >> 56)
	msg[9] = byte(messageLength >> 48)
	msg[10] = byte(messageLength >> 40)
	msg[11] = byte(messageLength >> 32)
	msg[12] = byte(messageLength >> 24)
	msg[13] = byte(messageLength >> 16)
	msg[14] = byte(messageLength >> 8)
	msg[15] = byte(messageLength)

	// End marker "7777" (last 4 bytes)
	copy(msg[len(msg)-4:], "7777")

	return msg
}

func TestFindMessagesSingle(t *testing.T) {
	// Create a single 256-byte message
	msg := makeTestMessage(0, 256)

	boundaries, err := FindMessages(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(boundaries) != 1 {
		t.Fatalf("expected 1 message, got %d", len(boundaries))
	}

	b := boundaries[0]
	if b.Start != 0 {
		t.Errorf("Start: got %d, want 0", b.Start)
	}
	if b.Length != 256 {
		t.Errorf("Length: got %d, want 256", b.Length)
	}
	if b.Index != 0 {
		t.Errorf("Index: got %d, want 0", b.Index)
	}
}

func TestFindMessagesMultiple(t *testing.T) {
	// Create three messages of different sizes
	msg1 := makeTestMessage(0, 100)
	msg2 := makeTestMessage(1, 200)
	msg3 := makeTestMessage(2, 150)

	// Concatenate them
	data := append(append(msg1, msg2...), msg3...)

	boundaries, err := FindMessages(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(boundaries) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(boundaries))
	}

	// Check message 1
	if boundaries[0].Start != 0 {
		t.Errorf("msg1 Start: got %d, want 0", boundaries[0].Start)
	}
	if boundaries[0].Length != 100 {
		t.Errorf("msg1 Length: got %d, want 100", boundaries[0].Length)
	}
	if boundaries[0].Index != 0 {
		t.Errorf("msg1 Index: got %d, want 0", boundaries[0].Index)
	}

	// Check message 2
	if boundaries[1].Start != 100 {
		t.Errorf("msg2 Start: got %d, want 100", boundaries[1].Start)
	}
	if boundaries[1].Length != 200 {
		t.Errorf("msg2 Length: got %d, want 200", boundaries[1].Length)
	}
	if boundaries[1].Index != 1 {
		t.Errorf("msg2 Index: got %d, want 1", boundaries[1].Index)
	}

	// Check message 3
	if boundaries[2].Start != 300 {
		t.Errorf("msg3 Start: got %d, want 300", boundaries[2].Start)
	}
	if boundaries[2].Length != 150 {
		t.Errorf("msg3 Length: got %d, want 150", boundaries[2].Length)
	}
	if boundaries[2].Index != 2 {
		t.Errorf("msg3 Index: got %d, want 2", boundaries[2].Index)
	}
}

func TestFindMessagesEmpty(t *testing.T) {
	boundaries, err := FindMessages([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(boundaries) != 0 {
		t.Errorf("expected 0 messages, got %d", len(boundaries))
	}
}

func TestFindMessagesInvalidMagic(t *testing.T) {
	data := []byte{
		'X', 'X', 'X', 'X', // Wrong magic
		0x00, 0x00, 0, 2,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14,
		0x00, 0x00, 0x00, 0x00,
	}

	_, err := FindMessages(data)
	if err == nil {
		t.Fatal("expected error for invalid magic, got nil")
	}

	// Should be InvalidFormatError
	var invErr *InvalidFormatError
	if !isErrorType(err, &invErr) {
		t.Errorf("expected InvalidFormatError, got %T", err)
	}
}

func TestFindMessagesTruncated(t *testing.T) {
	// Create a message but truncate it
	msg := makeTestMessage(0, 256)
	truncated := msg[:200] // Cut off before the end

	_, err := FindMessages(truncated)
	if err == nil {
		t.Fatal("expected error for truncated message, got nil")
	}
}

func TestFindMessagesIncompleteSection0(t *testing.T) {
	// Only 10 bytes (not enough for Section 0)
	data := []byte{'G', 'R', 'I', 'B', 0, 0, 0, 2, 0, 0}

	_, err := FindMessages(data)
	if err == nil {
		t.Fatal("expected error for incomplete Section 0, got nil")
	}
}

func TestFindMessagesMissingEndMarker(t *testing.T) {
	msg := makeTestMessage(0, 100)
	// Corrupt the end marker
	copy(msg[len(msg)-4:], "XXXX")

	_, err := FindMessages(msg)
	if err == nil {
		t.Fatal("expected error for missing end marker, got nil")
	}
}

func TestSplitMessagesSingle(t *testing.T) {
	msg := makeTestMessage(0, 256)

	messages, err := SplitMessages(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if len(messages[0]) != 256 {
		t.Errorf("message length: got %d, want 256", len(messages[0]))
	}

	// Verify it's the same data
	if string(messages[0][:4]) != "GRIB" {
		t.Error("message does not start with GRIB")
	}
	if string(messages[0][len(messages[0])-4:]) != "7777" {
		t.Error("message does not end with 7777")
	}
}

func TestSplitMessagesMultiple(t *testing.T) {
	msg1 := makeTestMessage(0, 100)
	msg2 := makeTestMessage(1, 200)
	msg3 := makeTestMessage(2, 150)

	data := append(append(msg1, msg2...), msg3...)

	messages, err := SplitMessages(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// Check lengths
	if len(messages[0]) != 100 {
		t.Errorf("msg1 length: got %d, want 100", len(messages[0]))
	}
	if len(messages[1]) != 200 {
		t.Errorf("msg2 length: got %d, want 200", len(messages[1]))
	}
	if len(messages[2]) != 150 {
		t.Errorf("msg3 length: got %d, want 150", len(messages[2]))
	}
}

func TestValidateMessageStructureValid(t *testing.T) {
	msg := makeTestMessage(0, 256)

	err := ValidateMessageStructure(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMessageStructureTooShort(t *testing.T) {
	data := []byte{'G', 'R', 'I', 'B'}

	err := ValidateMessageStructure(data)
	if err == nil {
		t.Fatal("expected error for too short message, got nil")
	}
}

func TestValidateMessageStructureInvalidSection0(t *testing.T) {
	data := []byte{
		'X', 'X', 'X', 'X', // Wrong magic
		0x00, 0x00, 0, 2,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14,
		0x00, 0x00, 0x00, 0x00,
	}

	err := ValidateMessageStructure(data)
	if err == nil {
		t.Fatal("expected error for invalid Section 0, got nil")
	}
}

func TestValidateMessageStructureLengthMismatch(t *testing.T) {
	msg := makeTestMessage(0, 256)
	// Truncate it
	truncated := msg[:200]

	err := ValidateMessageStructure(truncated)
	if err == nil {
		t.Fatal("expected error for length mismatch, got nil")
	}
}

func TestValidateMessageStructureMissingEndMarker(t *testing.T) {
	msg := makeTestMessage(0, 100)
	// Corrupt the end marker
	copy(msg[len(msg)-4:], "XXXX")

	err := ValidateMessageStructure(msg)
	if err == nil {
		t.Fatal("expected error for missing end marker, got nil")
	}
}

func TestFindMessagesLargeMessage(t *testing.T) {
	// Create a large message (1 MB)
	msg := makeTestMessage(0, 1024*1024)

	boundaries, err := FindMessages(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(boundaries) != 1 {
		t.Fatalf("expected 1 message, got %d", len(boundaries))
	}

	if boundaries[0].Length != 1024*1024 {
		t.Errorf("Length: got %d, want %d", boundaries[0].Length, 1024*1024)
	}
}

func TestFindMessagesMinimumSize(t *testing.T) {
	// Minimum valid message: 16 bytes (Section 0) + 4 bytes (end marker) = 20 bytes
	msg := makeTestMessage(0, 20)

	boundaries, err := FindMessages(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(boundaries) != 1 {
		t.Fatalf("expected 1 message, got %d", len(boundaries))
	}

	if boundaries[0].Length != 20 {
		t.Errorf("Length: got %d, want 20", boundaries[0].Length)
	}
}

// Helper function to check if err is of a specific type
func isErrorType(err error, target interface{}) bool {
	if err == nil {
		return false
	}

	switch target.(type) {
	case **InvalidFormatError:
		_, ok := err.(*InvalidFormatError)
		return ok
	case **ParseError:
		_, ok := err.(*ParseError)
		return ok
	default:
		return false
	}
}
