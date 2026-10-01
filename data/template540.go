package data

import (
	"fmt"

	"github.com/mmp/squall/internal"
	"github.com/mmp/squall/internal/jpeg2000"
)

// Template540 represents Data Representation Template 5.40: Grid point
// data - JPEG 2000 code stream format.
//
// The packed values X are the samples of a single-component JPEG 2000 image
// stored in Section 7, which are scaled as for simple packing:
// value = (R + X * 2^E) / 10^D. When a bitmap is present, the image holds
// only the values of the points the bitmap marks as present.
//
// Used by NCEP for products including the NAM nests (e.g., Hawaii) and by
// other centers such as CMC.
type Template540 struct {
	ReferenceValue         float32 // Reference value (R)
	BinaryScaleFactor      int16   // Binary scale factor (E)
	DecimalScaleFactor     int16   // Decimal scale factor (D)
	NumBitsPerValue        uint8   // Bit depth of the image (0 for a constant field)
	OriginalFieldType      uint8   // Type of original field values (Table 5.1)
	CompressionType        uint8   // Type of compression (Table 5.40): 0 lossless, 1 lossy
	TargetCompressionRatio uint8   // Target compression ratio M:1, if lossy
	NumberOfDataValues     uint32  // Number of data values to unpack
}

// ParseTemplate540 parses Data Representation Template 5.40.
//
// The template data (octets 12-23 of Section 5) should be 12 bytes.
func ParseTemplate540(numDataValues uint32, data []byte) (*Template540, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("template 5.40 requires at least 12 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)
	referenceValue, _ := r.Float32()
	binaryScaleFactor, _ := r.Int16()
	decimalScaleFactor, _ := r.Int16()
	bitsPerValue, _ := r.Uint8()
	originalFieldType, _ := r.Uint8()
	compressionType, _ := r.Uint8()
	compressionRatio, _ := r.Uint8()

	return &Template540{
		ReferenceValue:         referenceValue,
		BinaryScaleFactor:      binaryScaleFactor,
		DecimalScaleFactor:     decimalScaleFactor,
		NumBitsPerValue:        bitsPerValue,
		OriginalFieldType:      originalFieldType,
		CompressionType:        compressionType,
		TargetCompressionRatio: compressionRatio,
		NumberOfDataValues:     numDataValues,
	}, nil
}

// TemplateNumber returns 40 for Template 5.40.
func (t *Template540) TemplateNumber() int {
	return 40
}

// NumDataValues returns the number of data values.
func (t *Template540) NumDataValues() uint32 {
	return t.NumberOfDataValues
}

// BitsPerValue returns the bit depth of the packed values.
func (t *Template540) BitsPerValue() uint8 {
	return t.NumBitsPerValue
}

// Decode decodes the JPEG 2000 codestream and applies the scaling.
//
// If bitmap is provided, the output has one value per bitmap entry, with
// missingValue where the bitmap is false.
func (t *Template540) Decode(packedData []byte, bitmap []bool) ([]float32, error) {
	u := newUnpacker(t.ReferenceValue, t.BinaryScaleFactor, t.DecimalScaleFactor)

	// With 0 bits per value, all values are the reference value and there
	// may be no codestream at all.
	if t.NumBitsPerValue == 0 {
		return constantField(u.value(0), t.NumberOfDataValues, bitmap), nil
	}

	// Check the image size before decoding so that a corrupt codestream
	// can't cause a huge allocation.
	cfg, err := jpeg2000.DecodeConfig(packedData)
	if err != nil {
		return nil, fmt.Errorf("template 5.40: %w", err)
	}
	if len(cfg.Components) != 1 {
		return nil, fmt.Errorf("template 5.40: JPEG 2000 image has %d components, expected 1", len(cfg.Components))
	}
	if n := cfg.Components[0].Width * cfg.Components[0].Height; n != int(t.NumberOfDataValues) {
		return nil, fmt.Errorf("template 5.40: JPEG 2000 image has %d values, expected %d", n, t.NumberOfDataValues)
	}

	img, err := jpeg2000.Decode(packedData)
	if err != nil {
		return nil, fmt.Errorf("template 5.40: %w", err)
	}
	samples := img.Components[0].Data

	if bitmap != nil {
		return decodeWithBitmap(u, samples, bitmap)
	}
	values := make([]float32, len(samples))
	for i, x := range samples {
		values[i] = u.value(float64(x))
	}
	return values, nil
}

// String returns a human-readable description.
func (t *Template540) String() string {
	kind := "lossless"
	if t.CompressionType == 1 {
		kind = fmt.Sprintf("lossy, target ratio %d:1", t.TargetCompressionRatio)
	}
	return fmt.Sprintf("Template 5.40: JPEG 2000 (%s), %d values, %d bits/value, R=%g, E=%d, D=%d",
		kind, t.NumberOfDataValues, t.NumBitsPerValue, t.ReferenceValue,
		t.BinaryScaleFactor, t.DecimalScaleFactor)
}
