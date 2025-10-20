package data

import (
	"fmt"
	"math"

	"github.com/mmp/mgrib2/internal"
)

// Template50 represents Data Representation Template 5.0: Simple Packing.
//
// This is the most common data representation template (used in 80%+ of GRIB2 files).
// Data values are linearly scaled and packed as n-bit unsigned integers.
//
// Decoding formula: value = (R + X * 2^E) / 10^D
// where:
//   R = reference value (IEEE 754 float)
//   X = packed value (n-bit unsigned integer)
//   E = binary scale factor (signed int)
//   D = decimal scale factor (signed int)
type Template50 struct {
	ReferenceValue       float32 // Reference value (R) - minimum value in field
	BinaryScaleFactor    int16   // Binary scale factor (E)
	DecimalScaleFactor   int16   // Decimal scale factor (D)
	NumBitsPerValue      uint8   // Number of bits per packed value (n)
	OriginalFieldType    uint8   // Type of original field values (Table 5.1)
	NumberOfDataValues   uint32  // Number of data values to unpack
}

// ParseTemplate50 parses Data Representation Template 5.0.
//
// The template data should be 10 bytes for Template 5.0.
func ParseTemplate50(numDataValues uint32, data []byte) (*Template50, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("template 5.0 requires at least 10 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	referenceValue, _ := r.Float32()
	binaryScaleFactor, _ := r.Int16()
	decimalScaleFactor, _ := r.Int16()
	bitsPerValue, _ := r.Uint8()
	originalFieldType, _ := r.Uint8()

	return &Template50{
		ReferenceValue:      referenceValue,
		BinaryScaleFactor:   binaryScaleFactor,
		DecimalScaleFactor:  decimalScaleFactor,
		NumBitsPerValue:     bitsPerValue,
		OriginalFieldType:   originalFieldType,
		NumberOfDataValues:  numDataValues,
	}, nil
}

// TemplateNumber returns 0 for Template 5.0.
func (t *Template50) TemplateNumber() int {
	return 0
}

// NumDataValues returns the number of data values.
func (t *Template50) NumDataValues() uint32 {
	return t.NumberOfDataValues
}

// BitsPerValue returns the number of bits per value.
func (t *Template50) BitsPerValue() uint8 {
	return t.NumBitsPerValue
}

// Decode unpacks data using simple packing algorithm.
//
// Formula: value = (R + X * 2^E) / 10^D
//
// If bitmap is provided, it must have length equal to the number of grid points.
// The output will have the same length as the bitmap, with undefined values
// set to 9.999e20 where bitmap is false.
//
// If bitmap is nil, all values are assumed to be valid.
func (t *Template50) Decode(packedData []byte, bitmap []bool) ([]float64, error) {
	// Handle special case: 0 bits per value means all values are the reference value
	if t.NumBitsPerValue == 0 {
		count := t.NumberOfDataValues
		if bitmap != nil {
			count = uint32(len(bitmap))
		}

		values := make([]float64, count)
		refValue := t.applyScaling(0)

		if bitmap != nil {
			for i := range values {
				if bitmap[i] {
					values[i] = refValue
				} else {
					values[i] = 9.999e20 // Missing value
				}
			}
		} else {
			for i := range values {
				values[i] = refValue
			}
		}

		return values, nil
	}

	// Create bit reader for packed data
	bitReader := internal.NewBitReader(packedData)

	// Read packed values
	packedValues := make([]uint32, t.NumberOfDataValues)
	for i := uint32(0); i < t.NumberOfDataValues; i++ {
		val, err := bitReader.ReadBits(int(t.NumBitsPerValue))
		if err != nil {
			return nil, fmt.Errorf("failed to read packed value %d: %w", i, err)
		}
		packedValues[i] = uint32(val)
	}

	// Apply scaling and bitmap
	if bitmap != nil {
		return t.decodeWithBitmap(packedValues, bitmap)
	}

	return t.decodeWithoutBitmap(packedValues), nil
}

// decodeWithoutBitmap decodes when all values are valid.
func (t *Template50) decodeWithoutBitmap(packedValues []uint32) []float64 {
	values := make([]float64, len(packedValues))
	for i, packed := range packedValues {
		values[i] = t.applyScaling(packed)
	}
	return values
}

// decodeWithBitmap decodes and applies bitmap.
func (t *Template50) decodeWithBitmap(packedValues []uint32, bitmap []bool) ([]float64, error) {
	if len(packedValues) > len(bitmap) {
		return nil, fmt.Errorf("more packed values (%d) than bitmap entries (%d)",
			len(packedValues), len(bitmap))
	}

	values := make([]float64, len(bitmap))
	packedIdx := 0

	for i := range bitmap {
		if bitmap[i] {
			if packedIdx >= len(packedValues) {
				return nil, fmt.Errorf("bitmap indicates more valid points than packed values available")
			}
			values[i] = t.applyScaling(packedValues[packedIdx])
			packedIdx++
		} else {
			values[i] = 9.999e20 // Missing value
		}
	}

	if packedIdx != len(packedValues) {
		return nil, fmt.Errorf("bitmap mismatch: used %d packed values, have %d",
			packedIdx, len(packedValues))
	}

	return values, nil
}

// applyScaling applies the scaling formula to a packed value.
//
// Formula: value = (R + X * 2^E) / 10^D
func (t *Template50) applyScaling(packedValue uint32) float64 {
	// Start with reference value
	value := float64(t.ReferenceValue)

	// Add scaled packed value: X * 2^E
	if packedValue != 0 {
		binaryScale := math.Pow(2.0, float64(t.BinaryScaleFactor))
		value += float64(packedValue) * binaryScale
	}

	// Apply decimal scaling: / 10^D
	if t.DecimalScaleFactor != 0 {
		decimalScale := math.Pow(10.0, float64(t.DecimalScaleFactor))
		value /= decimalScale
	}

	return value
}

// String returns a human-readable description.
func (t *Template50) String() string {
	return fmt.Sprintf("Template 5.0: Simple Packing, %d values, %d bits/value, R=%g, E=%d, D=%d",
		t.NumberOfDataValues, t.NumBitsPerValue, t.ReferenceValue,
		t.BinaryScaleFactor, t.DecimalScaleFactor)
}
