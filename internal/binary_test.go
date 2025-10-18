package internal

import (
	"io"
	"math"
	"testing"
)

func TestReaderUint8(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    []uint8
		wantErr bool
	}{
		{
			name: "single byte",
			data: []byte{0x42},
			want: []uint8{0x42},
		},
		{
			name: "multiple bytes",
			data: []byte{0x00, 0xFF, 0x7F},
			want: []uint8{0x00, 0xFF, 0x7F},
		},
		{
			name:    "EOF",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(tt.data)

			for i, want := range tt.want {
				got, err := r.Uint8()
				if err != nil {
					t.Fatalf("read %d: unexpected error: %v", i, err)
				}
				if got != want {
					t.Errorf("read %d: got %d, want %d", i, got, want)
				}
			}

			// Next read should fail
			if !tt.wantErr {
				_, err := r.Uint8()
				if err != io.ErrUnexpectedEOF {
					t.Errorf("expected EOF, got %v", err)
				}
			} else {
				_, err := r.Uint8()
				if err == nil {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}

func TestReaderInt8(t *testing.T) {
	data := []byte{0x00, 0x7F, 0x80, 0xFF}
	want := []int8{0, 127, -128, -1}

	r := NewReader(data)
	for i, w := range want {
		got, err := r.Int8()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got != w {
			t.Errorf("read %d: got %d, want %d", i, got, w)
		}
	}
}

func TestReaderUint16(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint16
		wantErr bool
	}{
		{
			name: "big endian",
			data: []byte{0x12, 0x34},
			want: 0x1234,
		},
		{
			name: "zero",
			data: []byte{0x00, 0x00},
			want: 0,
		},
		{
			name: "max",
			data: []byte{0xFF, 0xFF},
			want: 0xFFFF,
		},
		{
			name:    "truncated",
			data:    []byte{0x12},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(tt.data)
			got, err := r.Uint16()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestReaderInt16(t *testing.T) {
	data := []byte{0x00, 0x00, 0x7F, 0xFF, 0x80, 0x00, 0xFF, 0xFF}
	want := []int16{0, 32767, -32768, -1}

	r := NewReader(data)
	for i, w := range want {
		got, err := r.Int16()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got != w {
			t.Errorf("read %d: got %d, want %d", i, got, w)
		}
	}
}

func TestReaderUint32(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56, 0x78}
	r := NewReader(data)

	got, err := r.Uint32()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := uint32(0x12345678)
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestReaderInt32(t *testing.T) {
	data := []byte{
		0x00, 0x00, 0x00, 0x00, // 0
		0x7F, 0xFF, 0xFF, 0xFF, // max positive
		0x80, 0x00, 0x00, 0x00, // min negative
		0xFF, 0xFF, 0xFF, 0xFF, // -1
	}
	want := []int32{0, 2147483647, -2147483648, -1}

	r := NewReader(data)
	for i, w := range want {
		got, err := r.Int32()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got != w {
			t.Errorf("read %d: got %d, want %d", i, got, w)
		}
	}
}

func TestReaderUint64(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	r := NewReader(data)

	got, err := r.Uint64()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := uint64(0x123456789ABCDEF0)
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestReaderFloat32(t *testing.T) {
	// IEEE 754 representation of 3.14159
	data := []byte{0x40, 0x49, 0x0F, 0xD0}
	r := NewReader(data)

	got, err := r.Float32()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := float32(3.14159)
	if math.Abs(float64(got-want)) > 0.00001 {
		t.Errorf("got %f, want %f", got, want)
	}
}

func TestReaderFloat64(t *testing.T) {
	// IEEE 754 representation of 3.141592653589793
	data := []byte{0x40, 0x09, 0x21, 0xFB, 0x54, 0x44, 0x2D, 0x18}
	r := NewReader(data)

	got, err := r.Float64()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := 3.141592653589793
	if math.Abs(got-want) > 0.00000000000001 {
		t.Errorf("got %f, want %f", got, want)
	}
}

func TestReaderBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	r := NewReader(data)

	got, err := r.Bytes(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []byte{0x01, 0x02, 0x03}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("byte %d: got %02x, want %02x", i, got[i], want[i])
		}
	}

	// Verify it's a copy by modifying the returned slice
	got[0] = 0xFF
	if r.data[0] == 0xFF {
		t.Error("Bytes() did not return a copy")
	}
}

func TestReaderBytesNoCopy(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	r := NewReader(data)

	got, err := r.BytesNoCopy(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's NOT a copy
	got[0] = 0xFF
	if r.data[0] != 0xFF {
		t.Error("BytesNoCopy() returned a copy instead of a reference")
	}
}

func TestReaderString(t *testing.T) {
	data := []byte("GRIB")
	r := NewReader(data)

	got, err := r.String(4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "GRIB"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReaderSkip(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	r := NewReader(data)

	if err := r.Skip(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := r.Uint8()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 0x03 {
		t.Errorf("got %02x, want 03", got)
	}
}

func TestReaderPeek(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	r := NewReader(data)

	got, err := r.Peek(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 || got[0] != 0x01 || got[1] != 0x02 {
		t.Errorf("Peek returned wrong data")
	}

	// Offset should not have changed
	if r.Offset() != 0 {
		t.Errorf("Peek changed offset to %d", r.Offset())
	}
}

func TestReaderRemaining(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	r := NewReader(data)

	if r.Remaining() != 5 {
		t.Errorf("initial: got %d, want 5", r.Remaining())
	}

	r.Uint8()
	if r.Remaining() != 4 {
		t.Errorf("after 1 byte: got %d, want 4", r.Remaining())
	}

	r.Uint16()
	if r.Remaining() != 2 {
		t.Errorf("after 3 bytes: got %d, want 2", r.Remaining())
	}
}

func TestReaderSetOffset(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	r := NewReader(data)

	if err := r.SetOffset(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := r.Uint8()
	if got != 0x04 {
		t.Errorf("got %02x, want 04", got)
	}

	// Test invalid offset
	if err := r.SetOffset(100); err == nil {
		t.Error("expected error for out of bounds offset")
	}
}

func TestBitReaderReadBits(t *testing.T) {
	// Binary: 10110011 01010101
	// Bits:   1011 0011 0101 0101
	data := []byte{0xB3, 0x55}
	br := NewBitReader(data)

	tests := []struct {
		nbits int
		want  uint64
		desc  string
	}{
		{1, 1, "bit 0: 1"},                    // 1
		{2, 1, "bits 1-2: 01"},                // 01
		{3, 4, "bits 3-5: 100"},               // 100
		{4, 13, "bits 6-9: 1101"},             // 1101 (crosses byte boundary)
		{6, 21, "bits 10-15: 010101"},         // 010101
	}

	for i, tt := range tests {
		got, err := br.ReadBits(tt.nbits)
		if err != nil {
			t.Fatalf("test %d: unexpected error: %v", i, err)
		}
		if got != tt.want {
			t.Errorf("test %d (%s): got %d, want %d", i, tt.desc, got, tt.want)
		}
	}
}

func TestBitReaderReadSignedBits(t *testing.T) {
	// Binary: 11111111 (all ones)
	data := []byte{0xFF}
	br := NewBitReader(data)

	// Read 4 bits: 1111 in 4-bit two's complement is -1
	got, err := br.ReadSignedBits(4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := int64(-1)
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestBitReaderAlign(t *testing.T) {
	data := []byte{0xFF, 0xFF}
	br := NewBitReader(data)

	// Read 3 bits
	br.ReadBits(3)
	if br.Offset() != 3 {
		t.Errorf("offset before align: got %d, want 3", br.Offset())
	}

	// Align to next byte boundary
	br.Align()
	if br.Offset() != 8 {
		t.Errorf("offset after align: got %d, want 8", br.Offset())
	}

	// Align when already aligned should do nothing
	br.Align()
	if br.Offset() != 8 {
		t.Errorf("offset after second align: got %d, want 8", br.Offset())
	}
}

func TestBitReaderEOF(t *testing.T) {
	data := []byte{0xFF}
	br := NewBitReader(data)

	// Read all 8 bits
	_, err := br.ReadBits(8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Next read should fail
	_, err = br.ReadBits(1)
	if err != io.ErrUnexpectedEOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestBitReaderInvalidBits(t *testing.T) {
	data := []byte{0xFF}
	br := NewBitReader(data)

	// Try to read 0 bits
	_, err := br.ReadBits(0)
	if err == nil {
		t.Error("expected error for 0 bits")
	}

	// Try to read > 64 bits
	_, err = br.ReadBits(65)
	if err == nil {
		t.Error("expected error for > 64 bits")
	}
}
