package data

// unpacker converts packed integers to data values using the GRIB2 formula
// Y = (R + X * 2^E) / 10^D.
//
// The arithmetic is done in float64 and rounded to float32 once at the end,
// exactly as wgrib2 does it: ((X * 2^E) + R) * 10^-D, with 10^-D computed by
// wgrib2's Int_Power. Doing the intermediate steps in float32 instead loses
// precision whenever R + X*2^E needs more than 24 significant bits.
type unpacker struct {
	ref, bin, dec float64
}

func newUnpacker(ref float32, binaryScale, decimalScale int16) unpacker {
	return unpacker{
		ref: float64(ref),
		bin: intPower(2, int(binaryScale)),
		dec: intPower(10, -int(decimalScale)),
	}
}

// value returns the data value for packed integer x.
func (u unpacker) value(x float64) float32 {
	return float32((x*u.bin + u.ref) * u.dec)
}

// intPower returns x^y by repeated squaring, matching wgrib2's Int_Power so
// that, for example, 10^-2 is 0.1*0.1 rather than the closest double to 0.01.
func intPower(x float64, y int) float64 {
	var p uint
	if y < 0 {
		p = uint(-y)
		x = 1 / x
	} else {
		p = uint(y)
	}
	v := 1.0
	for ; p != 0; p >>= 1 {
		if p&1 != 0 {
			v *= x
		}
		x *= x
	}
	return v
}
