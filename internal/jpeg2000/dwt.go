package jpeg2000

// Inverse discrete wavelet transforms (T.800 Annex F).
//
// One-dimensional transforms operate on a signal occupying the half-open
// interval [i0, i1) of absolute coordinates: samples at even coordinates
// are low-pass coefficients and those at odd coordinates are high-pass.
// The signal is extended symmetrically at both ends (F.3.7) by copying it
// into a buffer with margins on either side.

// Lifting parameters for the 9/7 irreversible filter (Table F.4).
const (
	alpha97 = -1.586134342059924
	beta97  = -0.052980118572961
	gamma97 = 0.882911075530934
	delta97 = 0.443506852043971
	k97     = 1.230174104914001
)

// mirror maps index i (relative to the start of a signal of length n > 1)
// into [0, n) by whole-sample symmetric extension.
func mirror(i, n int) int {
	period := 2 * (n - 1)
	i %= period
	if i < 0 {
		i += period
	}
	if i >= n {
		i = period - i
	}
	return i
}

// inverse53 applies the 1D reversible 5/3 synthesis (F.3.8.1) in place to x,
// which holds the signal at coordinates [i0, i0+len(x)). buf is scratch
// space of at least len(x)+4 elements.
func inverse53(x []int32, i0 int, buf []int64) {
	n := len(x)
	if n == 1 {
		if i0&1 != 0 {
			x[0] /= 2
		}
		return
	}
	const m = 2
	buf = buf[:n+2*m]
	for j := range buf {
		buf[j] = int64(x[mirror(j-m, n)])
	}
	// Coordinate i is at buf[i-i0+m]; parity is that of the absolute
	// coordinate.
	first := i0 - 1 + (i0-1)&1 // first even coordinate >= i0-1
	for i := first; i < i0+n+1; i += 2 {
		j := i - i0 + m
		buf[j] -= (buf[j-1] + buf[j+1] + 2) >> 2
	}
	first = i0 + 1 - i0&1 // first odd coordinate >= i0
	for i := first; i < i0+n; i += 2 {
		j := i - i0 + m
		buf[j] += (buf[j-1] + buf[j+1]) >> 1
	}
	for j := range x {
		x[j] = int32(buf[j+m])
	}
}

// inverse97 applies the 1D irreversible 9/7 synthesis (F.3.8.2) in place.
// buf must have at least len(x)+8 elements.
func inverse97(x []float32, i0 int, buf []float64) {
	n := len(x)
	if n == 1 {
		if i0&1 != 0 {
			x[0] /= 2
		}
		return
	}
	const m = 4
	buf = buf[:n+2*m]
	for j := range buf {
		buf[j] = float64(x[mirror(j-m, n)])
	}
	start := i0 - m // coordinate of buf[0]
	// Undo the scaling of the low- and high-pass coefficients.
	for j := range buf {
		if (start+j)&1 == 0 {
			buf[j] *= k97
		} else {
			buf[j] *= 1 / k97
		}
	}
	// lift updates the samples with coordinates of the given parity in
	// [lo, hi) using their two neighbors.
	lift := func(parity, lo, hi int, c float64) {
		i := lo + (lo-parity)&1
		for ; i < hi; i += 2 {
			j := i - start
			buf[j] -= c * (buf[j-1] + buf[j+1])
		}
	}
	i1 := i0 + n
	lift(0, i0-3, i1+3, delta97)
	lift(1, i0-2, i1+2, gamma97)
	lift(0, i0-1, i1+1, beta97)
	lift(1, i0, i1, alpha97)
	for j := range x {
		x[j] = float32(buf[j+m])
	}
}

// synthesize reconstructs resolution level r of a tile-component from the
// reconstructed resolution r-1 (low) and the three subbands of resolution
// r, using the 2D_SR procedure (F.3.2): interleave the coefficients, then
// transform each row, then each column.
func synthesize[T int32 | float32](res, lowRes *resolution, low []T, bandCoeffs func(*band) []T,
	inverse func([]T, int)) []T {
	w, h := res.x1-res.x0, res.y1-res.y0
	out := make([]T, w*h)
	if w == 0 || h == 0 {
		return out
	}
	lw := lowRes.x1 - lowRes.x0
	hl, lh, hh := res.bands[0], res.bands[1], res.bands[2]
	hlc, lhc, hhc := bandCoeffs(hl), bandCoeffs(lh), bandCoeffs(hh)
	for y := res.y0; y < res.y1; y++ {
		row := out[(y-res.y0)*w : (y-res.y0+1)*w]
		var evens, odds []T // low- and high-pass coefficients for this row
		if y&1 == 0 {
			ly := y/2 - lowRes.y0
			evens = low[ly*lw : (ly+1)*lw]
			hy := y/2 - hl.y0
			odds = hlc[hy*(hl.x1-hl.x0) : (hy+1)*(hl.x1-hl.x0)]
		} else {
			ly := y/2 - lh.y0
			evens = lhc[ly*(lh.x1-lh.x0) : (ly+1)*(lh.x1-lh.x0)]
			hy := y/2 - hh.y0
			odds = hhc[hy*(hh.x1-hh.x0) : (hy+1)*(hh.x1-hh.x0)]
		}
		// The low-pass band starts at coordinate ceil(x0/2), the high-pass
		// band at floor(x0/2).
		for x := res.x0; x < res.x1; x++ {
			if x&1 == 0 {
				row[x-res.x0] = evens[x/2-(res.x0+1)/2]
			} else {
				row[x-res.x0] = odds[x/2-res.x0/2]
			}
		}
		inverse(row, res.x0)
	}
	col := make([]T, h)
	for x := range w {
		for y := range h {
			col[y] = out[y*w+x]
		}
		inverse(col, res.y0)
		for y := range h {
			out[y*w+x] = col[y]
		}
	}
	return out
}
