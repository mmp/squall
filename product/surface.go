package product

import "math"

// SurfaceValue returns the value of a fixed surface from its scale factor
// and scaled value (value = scaledValue * 10^-scaleFactor). Both are stored
// as sign-magnitude integers, with the most significant bit indicating a
// negative value. If either has all bits set, the value is missing and
// SurfaceValue returns 0.
func SurfaceValue(scaleFactor uint8, scaledValue uint32) float64 {
	if SurfaceMissing(scaleFactor, scaledValue) {
		return 0
	}
	v := float64(scaledValue & 0x7FFFFFFF)
	if scaledValue&0x80000000 != 0 {
		v = -v
	}
	sf := int(scaleFactor & 0x7F)
	if scaleFactor&0x80 != 0 {
		return v * math.Pow10(sf)
	}
	return v / math.Pow10(sf)
}

// SurfaceMissing reports whether a fixed surface's value is missing: its
// scale factor or scaled value has all bits set.
func SurfaceMissing(scaleFactor uint8, scaledValue uint32) bool {
	return scaleFactor == 0xFF || scaledValue == 0xFFFFFFFF
}

// FixedSurface is a fixed surface of a level or layer.
type FixedSurface struct {
	Type    uint8   // Type of surface (Code Table 4.5); 255 if missing
	Value   float64 // Value, with the scale factor applied; 0 if missing
	Missing bool    // Whether the value is missing
}

func fixedSurface(typ, scaleFactor uint8, scaledValue uint32) FixedSurface {
	return FixedSurface{
		Type:    typ,
		Value:   SurfaceValue(scaleFactor, scaledValue),
		Missing: SurfaceMissing(scaleFactor, scaledValue),
	}
}
