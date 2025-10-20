package data

import (
	"fmt"
	"math"

	"github.com/mmp/mgrib2/internal"
)

// Template53 represents Data Representation Template 5.3: Complex Packing with Spatial Differencing.
//
// This template is used for efficient compression of gridded meteorological data by:
// 1. Applying spatial differencing (first or second order) to reduce dynamic range
// 2. Dividing data into groups with varying bit widths
// 3. Packing each group with only the bits needed for its range
//
// Commonly used by regional forecast models like HRRR and NAM.
type Template53 struct {
	ReferenceValue          float32 // Reference value (R) - base value for all data
	BinaryScaleFactor       int16   // Binary scale factor (E)
	DecimalScaleFactor      int16   // Decimal scale factor (D)
	NumBitsPerValue         uint8   // Number of bits for each value (before grouping)
	OriginalFieldType       uint8   // Type of original field values (Table 5.1)
	GroupSplittingMethod    uint8   // Method used to split data into groups (Table 5.4)
	MissingValueManagement  uint8   // Missing value management (Table 5.5)
	PrimaryMissingValue     float32 // Primary missing value substitute
	SecondaryMissingValue   float32 // Secondary missing value substitute
	NumberOfGroups          uint32  // Number of groups
	ReferenceGroupWidth     uint8   // Reference for group widths
	NumBitsGroupWidth       uint8   // Number of bits for group widths
	ReferenceGroupLength    uint32  // Reference for group lengths
	GroupLengthIncrement    uint8   // Increment for group lengths
	TrueLengthLastGroup     uint32  // True length of last group
	NumBitsGroupLength      uint8   // Number of bits for scaled group lengths
	SpatialDiffOrder        uint8   // Order of spatial differencing (1 or 2)
	NumOctetsExtraDescriptors uint8 // Number of octets for extra descriptors
	NumberOfDataValues      uint32  // Total number of data values to unpack
}

// ParseTemplate53 parses Data Representation Template 5.3.
//
// The template data should be at least 38 bytes for Template 5.3.
func ParseTemplate53(numDataValues uint32, data []byte) (*Template53, error) {
	if len(data) < 38 {
		return nil, fmt.Errorf("template 5.3 requires at least 38 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Bytes 0-3: Reference value (IEEE 32-bit float)
	referenceValue, _ := r.Float32()

	// Bytes 4-5: Binary scale factor (signed 16-bit)
	binaryScaleFactor, _ := r.Int16()

	// Bytes 6-7: Decimal scale factor (signed 16-bit)
	decimalScaleFactor, _ := r.Int16()

	// Byte 8: Number of bits per value
	bitsPerValue, _ := r.Uint8()

	// Byte 9: Original field type
	originalFieldType, _ := r.Uint8()

	// Byte 10: Group splitting method
	groupSplittingMethod, _ := r.Uint8()

	// Byte 11: Missing value management
	missingValueManagement, _ := r.Uint8()

	// Bytes 12-15: Primary missing value substitute
	primaryMissingValue, _ := r.Float32()

	// Bytes 16-19: Secondary missing value substitute
	secondaryMissingValue, _ := r.Float32()

	// Bytes 20-23: Number of groups
	numberOfGroups, _ := r.Uint32()

	// Byte 24: Reference for group widths
	referenceGroupWidth, _ := r.Uint8()

	// Byte 25: Number of bits for group widths
	numBitsGroupWidth, _ := r.Uint8()

	// Bytes 26-29: Reference for group lengths
	referenceGroupLength, _ := r.Uint32()

	// Byte 30: Increment for group lengths
	groupLengthIncrement, _ := r.Uint8()

	// Bytes 31-34: True length of last group
	trueLengthLastGroup, _ := r.Uint32()

	// Byte 35: Number of bits for scaled group lengths
	numBitsGroupLength, _ := r.Uint8()

	// Byte 36: Order of spatial differencing (Template 5.3 specific)
	spatialDiffOrder, _ := r.Uint8()

	// Byte 37: Number of octets for extra descriptors
	numOctetsExtraDescriptors, _ := r.Uint8()

	return &Template53{
		ReferenceValue:            referenceValue,
		BinaryScaleFactor:         binaryScaleFactor,
		DecimalScaleFactor:        decimalScaleFactor,
		NumBitsPerValue:           bitsPerValue,
		OriginalFieldType:         originalFieldType,
		GroupSplittingMethod:      groupSplittingMethod,
		MissingValueManagement:    missingValueManagement,
		PrimaryMissingValue:       primaryMissingValue,
		SecondaryMissingValue:     secondaryMissingValue,
		NumberOfGroups:            numberOfGroups,
		ReferenceGroupWidth:       referenceGroupWidth,
		NumBitsGroupWidth:         numBitsGroupWidth,
		ReferenceGroupLength:      referenceGroupLength,
		GroupLengthIncrement:      groupLengthIncrement,
		TrueLengthLastGroup:       trueLengthLastGroup,
		NumBitsGroupLength:        numBitsGroupLength,
		SpatialDiffOrder:          spatialDiffOrder,
		NumOctetsExtraDescriptors: numOctetsExtraDescriptors,
		NumberOfDataValues:        numDataValues,
	}, nil
}

// TemplateNumber returns 3 for Template 5.3.
func (t *Template53) TemplateNumber() int {
	return 3
}

// NumDataValues returns the number of data values.
func (t *Template53) NumDataValues() uint32 {
	return t.NumberOfDataValues
}

// BitsPerValue returns the number of bits per value.
func (t *Template53) BitsPerValue() uint8 {
	return t.NumBitsPerValue
}

// Decode unpacks data using complex packing with spatial differencing.
//
// Algorithm:
// 1. Read first values (spatial difference references)
// 2. Read minimum values for each group
// 3. Unpack group widths and lengths
// 4. Unpack data values for each group
// 5. Reverse spatial differencing
// 6. Apply scaling
//
// If bitmap is provided, it must have length equal to the number of grid points.
// The output will have the same length as the bitmap, with undefined values
// set to 9.999e20 where bitmap is false.
func (t *Template53) Decode(packedData []byte, bitmap []bool) ([]float64, error) {
	if len(packedData) == 0 {
		return nil, fmt.Errorf("no packed data to decode")
	}

	bitReader := internal.NewBitReader(packedData)

	// Calculate number of values including spatial difference references
	ndata := t.NumberOfDataValues
	if bitmap != nil {
		ndata = uint32(len(bitmap))
	}

	// Read spatial difference reference values and min_val
	// For Template 5.3 with spatial differencing:
	// - First come the reference values (1 or 2 depending on order)
	// - Then comes min_val (used as offset in spatial differencing)
	// These values are stored as bytes (octets), not bit-packed like regular data values.
	// The number of bytes per value is given by NumOctetsExtraDescriptors.
	var firstVals []int32
	var minVal int32
	if t.SpatialDiffOrder == 1 || t.SpatialDiffOrder == 2 {
		if t.NumOctetsExtraDescriptors == 0 {
			// No extra descriptors, so no first values or min_val in data section
			// This shouldn't happen for proper spatial differencing, but handle gracefully
			return nil, fmt.Errorf("spatial differencing order %d requires NumOctetsExtraDescriptors > 0, got 0",
				t.SpatialDiffOrder)
		}

		numFirstVals := int(t.SpatialDiffOrder)
		firstVals = make([]int32, numFirstVals)
		numOctets := int(t.NumOctetsExtraDescriptors)

		// Read first reference values (stored as bytes, not bit-packed)
		for i := 0; i < numFirstVals; i++ {
			val, err := bitReader.ReadBytes(numOctets)
			if err != nil {
				return nil, fmt.Errorf("failed to read first value %d: %w", i, err)
			}
			firstVals[i] = int32(val)
		}

		// Read min_val (minimum value offset, stored as signed bytes)
		val, err := bitReader.ReadSignedBytes(numOctets)
		if err != nil {
			return nil, fmt.Errorf("failed to read min_val: %w", err)
		}
		minVal = int32(val)
	}

	// Read minimum values for each group (group reference values)
	groupMinVals := make([]int32, t.NumberOfGroups)
	for i := uint32(0); i < t.NumberOfGroups; i++ {
		val, err := bitReader.ReadBits(int(t.NumBitsPerValue))
		if err != nil {
			return nil, fmt.Errorf("failed to read group min value %d: %w", i, err)
		}
		groupMinVals[i] = int32(val)
	}

	// Unpack group widths
	groupWidths := make([]uint8, t.NumberOfGroups)
	if t.NumBitsGroupWidth > 0 {
		for i := uint32(0); i < t.NumberOfGroups; i++ {
			val, err := bitReader.ReadBits(int(t.NumBitsGroupWidth))
			if err != nil {
				return nil, fmt.Errorf("failed to read group width %d: %w", i, err)
			}
			groupWidths[i] = uint8(val) + t.ReferenceGroupWidth
		}
	} else {
		// All groups use reference width
		for i := uint32(0); i < t.NumberOfGroups; i++ {
			groupWidths[i] = t.ReferenceGroupWidth
		}
	}

	// Unpack group lengths
	groupLengths := make([]uint32, t.NumberOfGroups)
	if t.NumBitsGroupLength > 0 {
		for i := uint32(0); i < t.NumberOfGroups; i++ {
			val, err := bitReader.ReadBits(int(t.NumBitsGroupLength))
			if err != nil {
				return nil, fmt.Errorf("failed to read group length %d: %w", i, err)
			}
			groupLengths[i] = t.ReferenceGroupLength + uint32(val)*uint32(t.GroupLengthIncrement)
		}
		// Last group uses true length
		if t.NumberOfGroups > 0 {
			groupLengths[t.NumberOfGroups-1] = t.TrueLengthLastGroup
		}
	} else {
		// All groups use reference length, except last group
		for i := uint32(0); i < t.NumberOfGroups; i++ {
			groupLengths[i] = t.ReferenceGroupLength
		}
		if t.NumberOfGroups > 0 {
			groupLengths[t.NumberOfGroups-1] = t.TrueLengthLastGroup
		}
	}

	// Unpack data values for each group
	// Total values = ndata - number of first values
	numUnpackedVals := int(ndata) - len(firstVals)
	unpackedVals := make([]int32, numUnpackedVals)

	idx := 0
	for i := uint32(0); i < t.NumberOfGroups; i++ {
		groupWidth := groupWidths[i]
		groupLength := groupLengths[i]
		groupMin := groupMinVals[i]

		for j := uint32(0); j < groupLength; j++ {
			if idx >= numUnpackedVals {
				break
			}

			if groupWidth == 0 {
				// All values in group are the minimum
				unpackedVals[idx] = groupMin
			} else {
				val, err := bitReader.ReadBits(int(groupWidth))
				if err != nil {
					return nil, fmt.Errorf("failed to read value in group %d: %w", i, err)
				}
				unpackedVals[idx] = groupMin + int32(val)
			}
			idx++
		}
	}

	// Combine first values and unpacked values
	allVals := make([]int32, len(firstVals)+len(unpackedVals))
	copy(allVals, firstVals)
	copy(allVals[len(firstVals):], unpackedVals)

	// Reverse spatial differencing
	var finalVals []int32
	if t.SpatialDiffOrder == 1 {
		finalVals = t.reverseSpatialDifferencing1(allVals, minVal)
	} else if t.SpatialDiffOrder == 2 {
		finalVals = t.reverseSpatialDifferencing2(allVals, minVal)
	} else {
		finalVals = allVals
	}

	// Apply scaling and convert to float64
	if bitmap != nil {
		return t.applyScalingWithBitmap(finalVals, bitmap)
	}
	return t.applyScalingWithoutBitmap(finalVals), nil
}

// reverseSpatialDifferencing1 reverses first-order spatial differencing.
//
// First-order differencing: Y[n] = X[n] - X[n-1]
// Reversal: X[n] = X[n-1] + Y[n] + min_val
//
// The min_val offset is added at each step per the GRIB2 specification
// and reference implementations (wgrib2, go-grib2).
func (t *Template53) reverseSpatialDifferencing1(diffVals []int32, minVal int32) []int32 {
	if len(diffVals) == 0 {
		return diffVals
	}

	vals := make([]int32, len(diffVals))
	vals[0] = diffVals[0] // First value is the reference, unchanged

	for i := 1; i < len(diffVals); i++ {
		vals[i] = vals[i-1] + diffVals[i] + minVal
	}

	return vals
}

// reverseSpatialDifferencing2 reverses second-order spatial differencing.
//
// Second-order differencing: Z[n] = (X[n] - X[n-1]) - (X[n-1] - X[n-2])
//                                  = X[n] - 2*X[n-1] + X[n-2]
// Reversal: X[n] = Z[n] + 2*X[n-1] - X[n-2] + min_val
//
// The min_val offset is added at each step per the GRIB2 specification
// and reference implementations (wgrib2, go-grib2).
func (t *Template53) reverseSpatialDifferencing2(diffVals []int32, minVal int32) []int32 {
	if len(diffVals) < 2 {
		return diffVals
	}

	vals := make([]int32, len(diffVals))
	vals[0] = diffVals[0] // First value is the reference, unchanged
	vals[1] = diffVals[1] // Second value is the reference, unchanged

	for i := 2; i < len(diffVals); i++ {
		vals[i] = diffVals[i] + 2*vals[i-1] - vals[i-2] + minVal
	}

	return vals
}

// applyScalingWithoutBitmap applies scaling when all values are valid.
func (t *Template53) applyScalingWithoutBitmap(packedValues []int32) []float64 {
	values := make([]float64, len(packedValues))
	for i, packed := range packedValues {
		values[i] = t.applyScaling(packed)
	}
	return values
}

// applyScalingWithBitmap applies scaling and bitmap.
func (t *Template53) applyScalingWithBitmap(packedValues []int32, bitmap []bool) ([]float64, error) {
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
func (t *Template53) applyScaling(packedValue int32) float64 {
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
func (t *Template53) String() string {
	return fmt.Sprintf("Template 5.3: Complex Packing (Spatial Diff Order %d), %d values, %d groups, R=%g, E=%d, D=%d",
		t.SpatialDiffOrder, t.NumberOfDataValues, t.NumberOfGroups, t.ReferenceValue,
		t.BinaryScaleFactor, t.DecimalScaleFactor)
}
