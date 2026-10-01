package squall

import (
	"bytes"
	"math"
	"testing"
)

// makeCompleteGRIB2Message creates a complete GRIB2 message for testing.
//
// This creates a realistic message with:
// - Section 0: Meteorological discipline
// - Section 1: NCEP, 2023-01-15 12:00 UTC
// - Section 2: (optional, not included)
// - Section 3: 3x3 lat/lon grid, 90N-88N, 0E-2E
// - Section 4: Temperature at 500mb
// - Section 5: Simple packing, 8 bits/value
// - Section 6: No bitmap (all points valid)
// - Section 7: 9 data values
// - Section 8: "7777" end marker
func makeCompleteGRIB2Message() []byte {
	msg := []byte{}

	// Section 0: Indicator (16 bytes)
	sec0 := make([]byte, 16)
	copy(sec0[0:4], "GRIB")       // Magic number
	sec0[4], sec0[5] = 0x00, 0x00 // Reserved
	sec0[6] = 0                   // Discipline: Meteorological
	sec0[7] = 2                   // Edition 2
	// Message length will be filled in at the end (bytes 8-15)
	msg = append(msg, sec0...)

	// Section 1: Identification (21 bytes)
	sec1 := make([]byte, 21)
	// Length
	sec1[0], sec1[1], sec1[2], sec1[3] = 0x00, 0x00, 0x00, 0x15 // 21
	sec1[4] = 1                                                 // Section number
	// Center: 7 (NCEP)
	sec1[5], sec1[6] = 0x00, 0x07
	// Subcenter: 0
	sec1[7], sec1[8] = 0x00, 0x00
	// Master tables version: 2
	sec1[9] = 2
	// Local tables version: 1
	sec1[10] = 1
	// Significance of reference time: 1 (Start of forecast)
	sec1[11] = 1
	// Reference time: 2023-01-15 12:00:00
	sec1[12], sec1[13] = 0x07, 0xE7 // Year 2023
	sec1[14] = 1                    // Month
	sec1[15] = 15                   // Day
	sec1[16] = 12                   // Hour
	sec1[17] = 0                    // Minute
	sec1[18] = 0                    // Second
	// Production status: 0 (Operational)
	sec1[19] = 0
	// Type of data: 1 (Forecast)
	sec1[20] = 1
	msg = append(msg, sec1...)

	// Section 3: Grid Definition (72 bytes, Template 3.0)
	sec3 := make([]byte, 72)
	// Length
	sec3[0], sec3[1], sec3[2], sec3[3] = 0x00, 0x00, 0x00, 0x48 // 72
	sec3[4] = 3                                                 // Section number
	sec3[5] = 0                                                 // Source of grid definition
	// Number of data points: 9 (3x3)
	sec3[6], sec3[7], sec3[8], sec3[9] = 0x00, 0x00, 0x00, 0x09
	sec3[10] = 0 // Number of octets for optional list
	sec3[11] = 0 // Interpretation
	// Template number: 0 (Lat/Lon)
	sec3[12], sec3[13] = 0x00, 0x00
	// Shape of earth, etc. (16 bytes) - zeros for simplicity
	// Ni: 3
	sec3[30], sec3[31], sec3[32], sec3[33] = 0x00, 0x00, 0x00, 0x03
	// Nj: 3
	sec3[34], sec3[35], sec3[36], sec3[37] = 0x00, 0x00, 0x00, 0x03
	// Basic angle and subdivisions (8 bytes zeros): angles in microdegrees
	// La1: 90000000 microdegrees (90°N)
	sec3[46], sec3[47], sec3[48], sec3[49] = 0x05, 0x5D, 0x4A, 0x80
	// Lo1: 0 microdegrees (0°E)
	sec3[50], sec3[51], sec3[52], sec3[53] = 0x00, 0x00, 0x00, 0x00
	// Resolution flags
	sec3[54] = 0x00
	// La2: 88000000 microdegrees (88°N)
	sec3[55], sec3[56], sec3[57], sec3[58] = 0x05, 0x3E, 0xC6, 0x00
	// Lo2: 2000000 microdegrees (2°E)
	sec3[59], sec3[60], sec3[61], sec3[62] = 0x00, 0x1E, 0x84, 0x80
	// Di: 1000000 microdegrees (1°)
	sec3[63], sec3[64], sec3[65], sec3[66] = 0x00, 0x0F, 0x42, 0x40
	// Dj: 1000000 microdegrees (1°)
	sec3[67], sec3[68], sec3[69], sec3[70] = 0x00, 0x0F, 0x42, 0x40
	// Scanning mode: 0x00
	sec3[71] = 0x00
	msg = append(msg, sec3...)

	// Section 4: Product Definition (43 bytes, Template 4.0)
	sec4 := make([]byte, 43)
	// Length
	sec4[0], sec4[1], sec4[2], sec4[3] = 0x00, 0x00, 0x00, 0x2B // 43
	sec4[4] = 4                                                 // Section number
	// Coordinate values: 0
	sec4[5], sec4[6] = 0x00, 0x00
	// Template number: 0
	sec4[7], sec4[8] = 0x00, 0x00
	// Parameter category: 0 (Temperature)
	sec4[9] = 0
	// Parameter number: 0 (Temperature)
	sec4[10] = 0
	// Generating process type: 0
	sec4[11] = 0
	// Background process: 0
	sec4[12] = 0
	// Forecast process: 0
	sec4[13] = 0
	// Hours after cutoff: 0
	sec4[14], sec4[15] = 0x00, 0x00
	// Minutes after cutoff: 0
	sec4[16] = 0
	// Time range unit: 1 (hour)
	sec4[17] = 1
	// Forecast time: 0
	sec4[18], sec4[19], sec4[20], sec4[21] = 0x00, 0x00, 0x00, 0x00
	// First surface type: 100 (Isobaric surface)
	sec4[22] = 100
	// First surface scale factor: 0
	sec4[23] = 0
	// First surface value: 50000 (500 hPa)
	sec4[24], sec4[25], sec4[26], sec4[27] = 0x00, 0x00, 0xC3, 0x50
	// Second surface type: 255 (missing)
	sec4[28] = 255
	sec4[29] = 0
	sec4[30], sec4[31], sec4[32], sec4[33] = 0x00, 0x00, 0x00, 0x00
	msg = append(msg, sec4...)

	// Section 5: Data Representation (22 bytes, Template 5.0)
	sec5 := make([]byte, 22)
	// Length
	sec5[0], sec5[1], sec5[2], sec5[3] = 0x00, 0x00, 0x00, 0x16 // 22
	sec5[4] = 5                                                 // Section number
	// Number of data values: 9
	sec5[5], sec5[6], sec5[7], sec5[8] = 0x00, 0x00, 0x00, 0x09
	// Template number: 0 (Simple packing)
	sec5[9], sec5[10] = 0x00, 0x00
	// Reference value: 250.0 K (IEEE 754 float)
	refBits := uint32(0x437A0000) // 250.0 in IEEE 754
	sec5[11], sec5[12], sec5[13], sec5[14] = byte(refBits>>24), byte(refBits>>16), byte(refBits>>8), byte(refBits)
	// Binary scale factor: 0
	sec5[15], sec5[16] = 0x00, 0x00
	// Decimal scale factor: 0
	sec5[17], sec5[18] = 0x00, 0x00
	// Bits per value: 8
	sec5[19] = 8
	// Type of original values: 0 (floating point)
	sec5[20] = 0
	msg = append(msg, sec5...)

	// Section 6: Bitmap (6 bytes, no bitmap)
	sec6 := make([]byte, 6)
	// Length
	sec6[0], sec6[1], sec6[2], sec6[3] = 0x00, 0x00, 0x00, 0x06 // 6
	sec6[4] = 6                                                 // Section number
	sec6[5] = 255                                               // Bitmap indicator: 255 (no bitmap)
	msg = append(msg, sec6...)

	// Section 7: Data (14 bytes: 5 header + 9 data)
	sec7 := make([]byte, 14)
	// Length
	sec7[0], sec7[1], sec7[2], sec7[3] = 0x00, 0x00, 0x00, 0x0E // 14
	sec7[4] = 7                                                 // Section number
	// Data values: 0, 1, 2, 3, 4, 5, 6, 7, 8 (packed as 8-bit values)
	// These will decode to: 250.0, 251.0, 252.0, ..., 258.0
	for i := range 9 {
		sec7[5+i] = byte(i)
	}
	msg = append(msg, sec7...)

	// Section 8: End marker
	msg = append(msg, []byte("7777")...)

	// Fill in message length in Section 0
	msgLen := uint64(len(msg))
	msg[8] = byte(msgLen >> 56)
	msg[9] = byte(msgLen >> 48)
	msg[10] = byte(msgLen >> 40)
	msg[11] = byte(msgLen >> 32)
	msg[12] = byte(msgLen >> 24)
	msg[13] = byte(msgLen >> 16)
	msg[14] = byte(msgLen >> 8)
	msg[15] = byte(msgLen)

	return msg
}

func TestParseMessageComplete(t *testing.T) {
	data := makeCompleteGRIB2Message()

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	// Verify Section 0
	if msg.Section0 == nil {
		t.Fatal("Section0 is nil")
	}
	if msg.Section0.Discipline != 0 {
		t.Errorf("Discipline: got %d, want 0", msg.Section0.Discipline)
	}

	// Verify Section 1
	if msg.Section1 == nil {
		t.Fatal("Section1 is nil")
	}
	if msg.Section1.OriginatingCenter != 7 {
		t.Errorf("OriginatingCenter: got %d, want 7 (NCEP)", msg.Section1.OriginatingCenter)
	}

	// Verify Section 2 (should be nil - not included)
	if msg.Section2 != nil {
		t.Error("Section2 should be nil (not included in test message)")
	}

	// Verify Section 3
	if msg.Section3 == nil {
		t.Fatal("Section3 is nil")
	}
	if msg.Section3.NumDataPoints != 9 {
		t.Errorf("NumDataPoints: got %d, want 9", msg.Section3.NumDataPoints)
	}

	// Verify Section 4
	if msg.Section4 == nil {
		t.Fatal("Section4 is nil")
	}
	if msg.Section4.Product.GetParameterCategory() != 0 {
		t.Errorf("ParameterCategory: got %d, want 0", msg.Section4.Product.GetParameterCategory())
	}

	// Verify Section 5
	if msg.Section5 == nil {
		t.Fatal("Section5 is nil")
	}
	if msg.Section5.NumDataValues != 9 {
		t.Errorf("NumDataValues: got %d, want 9", msg.Section5.NumDataValues)
	}

	// Verify Section 6
	if msg.Section6 == nil {
		t.Fatal("Section6 is nil")
	}
	if msg.Section6.BitmapIndicator != 255 {
		t.Errorf("BitmapIndicator: got %d, want 255", msg.Section6.BitmapIndicator)
	}

	// Verify Section 7
	if msg.Section7 == nil {
		t.Fatal("Section7 is nil")
	}
	if len(msg.Section7.Data) != 9 {
		t.Errorf("Data length: got %d, want 9", len(msg.Section7.Data))
	}
}

func TestMessageDecodeData(t *testing.T) {
	data := makeCompleteGRIB2Message()

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	values, err := msg.DecodeData()
	if err != nil {
		t.Fatalf("DecodeData failed: %v", err)
	}

	// Should have 9 values
	if len(values) != 9 {
		t.Fatalf("got %d values, want 9", len(values))
	}

	// Values should be 250.0, 251.0, ..., 258.0
	for i, val := range values {
		expected := float32(250.0 + float64(i))
		if math.Abs(float64(val-expected)) > 0.001 {
			t.Errorf("value[%d]: got %.3f, want %.3f", i, val, expected)
		}
	}
}

func TestMessageCoordinates(t *testing.T) {
	data := makeCompleteGRIB2Message()

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	lats, lons, err := msg.Coordinates()
	if err != nil {
		t.Fatalf("Coordinates failed: %v", err)
	}

	// Should have 9 coordinates
	if len(lats) != 9 {
		t.Fatalf("got %d latitudes, want 9", len(lats))
	}
	if len(lons) != 9 {
		t.Fatalf("got %d longitudes, want 9", len(lons))
	}

	// First point should be 90°N, 0°E
	if math.Abs(float64(lats[0]-90.0)) > 0.001 {
		t.Errorf("first lat: got %.3f, want 90.0", lats[0])
	}
	if math.Abs(float64(lons[0]-0.0)) > 0.001 {
		t.Errorf("first lon: got %.3f, want 0.0", lons[0])
	}

	// Last point should be 88°N, 2°E
	if math.Abs(float64(lats[8]-88.0)) > 0.001 {
		t.Errorf("last lat: got %.3f, want 88.0", lats[8])
	}
	if math.Abs(float64(lons[8]-2.0)) > 0.001 {
		t.Errorf("last lon: got %.3f, want 2.0", lons[8])
	}
}

func TestMessageString(t *testing.T) {
	data := makeCompleteGRIB2Message()

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	str := msg.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Should contain discipline, grid, and product info
	if len(str) < 20 {
		t.Errorf("String() too short: %q", str)
	}
}

func TestParseMessageInvalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"Too short", []byte("GRIB")},
		{"No end marker", makeCompleteGRIB2Message()[:100]},
		{"Wrong magic", []byte("XXXX" + string(makeCompleteGRIB2Message()[4:]))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMessage(tt.data)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// makeMultiFieldMessage returns a message holding two fields on the
// 3x3 grid of makeCompleteGRIB2Message: the first defines a bitmap with 7
// points present, and the second reuses it with bitmap indicator 254 if
// reuseBitmap is true or has no bitmap section of its own otherwise.
func makeMultiFieldMessage(reuseBitmap bool) []byte {
	base := makeCompleteGRIB2Message()
	head := base[:109] // Sections 0, 1, and 3
	sec4 := base[109:152]
	sec5 := append([]byte(nil), base[152:174]...)
	sec5[8] = 7 // 7 data values
	sec5b := append([]byte(nil), sec5...)
	sec5b[11], sec5b[12], sec5b[13], sec5b[14] = 0x42, 0xC8, 0x00, 0x00 // reference value 100
	sec7 := func(start byte) []byte {
		s := []byte{0, 0, 0, 12, 7}
		for i := range byte(7) {
			s = append(s, start+i)
		}
		return s
	}

	var msg []byte
	msg = append(msg, head...)
	msg = append(msg, sec4...)
	msg = append(msg, sec5...)
	msg = append(msg, 0, 0, 0, 8, 6, 0, 0b10110111, 0b10000000) // bitmap
	msg = append(msg, sec7(0)...)
	msg = append(msg, sec4...)
	msg = append(msg, sec5b...)
	if reuseBitmap {
		msg = append(msg, 0, 0, 0, 6, 6, 254)
	} else {
		msg = append(msg, 0, 0, 0, 6, 6, 255)
	}
	msg = append(msg, sec7(10)...)
	msg = append(msg, "7777"...)
	for i := range 8 {
		msg[8+i] = byte(uint64(len(msg)) >> (56 - 8*i))
	}
	return msg
}

func TestParseMessageFields(t *testing.T) {
	data := makeMultiFieldMessage(true)
	fields, err := ParseMessageFields(data)
	if err != nil {
		t.Fatalf("ParseMessageFields failed: %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("got %d fields, want 2", len(fields))
	}

	present := []bool{true, false, true, true, false, true, true, true, true}
	for i, ref := range []float32{250, 110} {
		m := fields[i]
		if m.Section3 != fields[0].Section3 {
			t.Errorf("field %d does not share Section 3", i)
		}
		values, err := m.DecodeData()
		if err != nil {
			t.Fatalf("field %d: DecodeData failed: %v", i, err)
		}
		if len(values) != 9 {
			t.Fatalf("field %d: got %d values, want 9", i, len(values))
		}
		next := ref
		for j, v := range values {
			if !present[j] {
				if !IsMissing(v) {
					t.Errorf("field %d: value %d = %v, want missing", i, j, v)
				}
				continue
			}
			if v != next {
				t.Errorf("field %d: value %d = %v, want %v", i, j, v, next)
			}
			next++
		}
	}

	first, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if values, err := first.DecodeData(); err != nil || values[0] != 250 {
		t.Errorf("ParseMessage did not return the first field (values %v, error %v)", values, err)
	}

	// Read returns both fields.
	records, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Read returned %d records, want 2", len(records))
	}
}

func TestParseMessageFieldsBitmapErrors(t *testing.T) {
	// Bitmap indicator 254 with no earlier bitmap in the message.
	data := makeMultiFieldMessage(true)
	bitmapSec6 := []byte{0, 0, 0, 8, 6, 0, 0b10110111, 0b10000000}
	i := bytes.Index(data, bitmapSec6)
	if i < 0 {
		t.Fatal("bitmap section not found")
	}
	data[i+5] = 254
	if _, err := ParseMessageFields(data); err == nil {
		t.Error("expected error for bitmap indicator 254 with no previous bitmap")
	}

	// Without a bitmap, the second field has 7 values for 9 grid points.
	fields, err := ParseMessageFields(makeMultiFieldMessage(false))
	if err != nil {
		t.Fatalf("ParseMessageFields failed: %v", err)
	}
	if fields[1].Section6.HasBitmap() {
		t.Error("second field should not have a bitmap")
	}
}
