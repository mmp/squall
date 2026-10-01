package jpeg2000

import "fmt"

// Tier-1 decoding: the bit-plane coding of code-blocks (T.800 Annex D).

// Per-sample state flags. The low byte records which of the eight neighbors
// are significant, and is used directly to index the zero coding context
// tables.
const (
	fNW = 1 << iota
	fN
	fNE
	fW
	fE
	fSW
	fS
	fSE
	fSgnN // the north neighbor is significant and negative
	fSgnW
	fSgnE
	fSgnS
	fSig     // the sample is significant
	fVisit   // the sample was coded in this bit-plane's significance pass
	fRefined // the sample has had a magnitude refinement

	fNeighbors = fNW | fN | fNE | fW | fE | fSW | fS | fSE

	// With vertically causal context formation, samples in the stripe below
	// are treated as insignificant when coding a stripe's last row.
	vscMask = ^uint16(fSW | fS | fSE | fSgnS)
)

// Subband orientations.
const (
	bandLL = iota
	bandHL
	bandLH
	bandHH
)

var (
	// zcContext maps a band orientation and neighbor significance to a zero
	// coding context (Table D.1).
	zcContext [4][256]uint8
	// scContext maps the significance and signs of the four horizontal and
	// vertical neighbors to a sign coding context in the low bits and the
	// sign prediction ("XORbit") in bit 7 (Tables D.2 and D.3).
	scContext [256]uint8
)

func init() {
	for orient := range 4 {
		for f := range 256 {
			h := bit(f, fW) + bit(f, fE)
			v := bit(f, fN) + bit(f, fS)
			d := bit(f, fNW) + bit(f, fNE) + bit(f, fSW) + bit(f, fSE)
			zcContext[orient][f] = uint8(zeroCodingContext(orient, h, v, d))
		}
	}
	for i := range 256 {
		contrib := func(sig, neg int) int {
			if i&sig == 0 {
				return 0
			} else if i&neg != 0 {
				return -1
			}
			return 1
		}
		clamp := func(v int) int { return max(-1, min(1, v)) }
		h := clamp(contrib(2, 32) + contrib(4, 64))  // W, E
		v := clamp(contrib(1, 16) + contrib(8, 128)) // N, S
		xor := 0
		if h < 0 || (h == 0 && v < 0) {
			h, v, xor = -h, -v, 1
		}
		var ctx int
		switch {
		case h == 0 && v == 0:
			ctx = 0
		case h == 0:
			ctx = 1
		default:
			ctx = 3 + v
		}
		scContext[i] = uint8(ctxSC+ctx) | uint8(xor<<7)
	}
}

func bit(f, mask int) int {
	if f&mask != 0 {
		return 1
	}
	return 0
}

func zeroCodingContext(orient, h, v, d int) int {
	if orient == bandHH {
		hv := h + v
		switch {
		case d >= 3:
			return 8
		case d == 2:
			return 6 + min(hv, 1)
		case d == 1:
			return 3 + min(hv, 2)
		default:
			return min(hv, 2)
		}
	}
	if orient == bandHL {
		h, v = v, h
	}
	switch {
	case h == 2:
		return 8
	case h == 1:
		if v >= 1 {
			return 7
		} else if d >= 1 {
			return 6
		}
		return 5
	case v == 2:
		return 4
	case v == 1:
		return 3
	default:
		return min(d, 2)
	}
}

// scIndex extracts the scContext index from a sample's flags.
func scIndex(f uint16) int {
	return int(f>>1&1 | f>>2&2 | f>>2&4 | f>>3&8 | f>>4&0xF0)
}

// t1 holds the working state for decoding one code-block at a time.
type t1 struct {
	w, h   int
	stride int
	flags  []uint16 // (w+2)x(h+2), with a border so neighbors are always valid
	data   []int32  // decoded values, scaled by 2 to hold the reconstruction offset
	mq     mqDecoder
	raw    rawDecoder
}

func (t *t1) reset(w, h int) {
	t.w, t.h, t.stride = w, h, w+2
	nf := (w + 2) * (h + 2)
	if cap(t.flags) < nf {
		t.flags = make([]uint16, nf)
	}
	t.flags = t.flags[:nf]
	clear(t.flags)
	if cap(t.data) < w*h {
		t.data = make([]int32, w*h)
	}
	t.data = t.data[:w*h]
	clear(t.data)
}

// decode decodes a code-block's coding passes into t.data. Each decoded
// value is twice the magnitude reconstructed from the decoded bit-planes
// (including the midpoint of the remaining uncertainty interval), so that
// the reconstruction offset can be represented in integers.
func (t *t1) decode(cb *codeBlock, orient int, style byte, roiShift int) error {
	t.reset(cb.x1-cb.x0, cb.y1-cb.y0)
	if cb.numbps <= 0 || len(cb.segs) == 0 {
		return nil
	}
	// bpno is one more than the index of the bit-plane being decoded.
	bpno := roiShift + cb.numbps
	if bpno > 30 {
		return fmt.Errorf("code-block has too many bit-planes (%d)", bpno)
	}
	vsc := style&styleVSC != 0
	t.mq.resetContexts()
	passType := 2 // the first pass is a cleanup pass
	for _, seg := range cb.segs {
		raw := style&styleBypass != 0 && bpno <= cb.numbps-4 && passType < 2
		if raw {
			t.raw.init(seg.data)
		} else {
			t.mq.init(seg.data)
		}
		for range seg.passes {
			if bpno < 1 {
				break
			}
			switch passType {
			case 0:
				t.significancePass(bpno, orient, vsc, raw)
			case 1:
				t.refinementPass(bpno, vsc, raw)
			case 2:
				t.cleanupPass(bpno, orient, vsc)
				if style&styleSegSym != 0 {
					for range 4 {
						t.mq.decode(ctxUniform)
					}
				}
			}
			if style&styleReset != 0 && !raw {
				t.mq.resetContexts()
			}
			if passType++; passType == 3 {
				passType = 0
				bpno--
			}
		}
	}

	if roiShift > 0 {
		// Max-shift ROI (Annex H): coefficients whose magnitude is at least
		// 2^roiShift belong to the ROI and were scaled up by the encoder.
		thresh := int32(1) << (roiShift + 1)
		for i, v := range t.data {
			mag := v
			if mag < 0 {
				mag = -mag
			}
			if mag >= thresh {
				mag >>= roiShift
				if v < 0 {
					mag = -mag
				}
				t.data[i] = mag
			}
		}
	}
	return nil
}

// setSignificant marks the sample at flag index fi as significant and
// records that in its neighbors' flags.
func (t *t1) setSignificant(fi int, negative bool) {
	f, s := t.flags, t.stride
	f[fi-s-1] |= fSE
	f[fi-s+1] |= fSW
	f[fi+s-1] |= fNE
	f[fi+s+1] |= fNW
	if negative {
		f[fi-s] |= fS | fSgnS
		f[fi+s] |= fN | fSgnN
		f[fi-1] |= fE | fSgnE
		f[fi+1] |= fW | fSgnW
	} else {
		f[fi-s] |= fS
		f[fi+s] |= fN
		f[fi-1] |= fE
		f[fi+1] |= fW
	}
	f[fi] |= fSig
}

// decodeSign decodes the sign of a newly significant sample and records it.
func (t *t1) decodeSign(fi, di int, f uint16, value int32) {
	sc := scContext[scIndex(f)]
	negative := t.mq.decode(int(sc&0x1F))^int(sc>>7) != 0
	if negative {
		value = -value
	}
	t.data[di] = value
	t.setSignificant(fi, negative)
}

// The coding passes visit samples in stripes of four rows; within a stripe,
// each column is scanned top to bottom, and the columns left to right
// (D.2).

func (t *t1) significancePass(bpno, orient int, vsc, raw bool) {
	one := int32(1) << bpno
	value := one | one>>1
	zc := &zcContext[orient]
	for k := 0; k < t.h; k += 4 {
		kend := min(k+4, t.h)
		for x := range t.w {
			fi := (k+1)*t.stride + x + 1
			di := k*t.w + x
			for y := k; y < kend; y, fi, di = y+1, fi+t.stride, di+t.w {
				f := t.flags[fi]
				if vsc && y == k+3 {
					f &= vscMask
				}
				if f&(fSig|fVisit) != 0 || f&fNeighbors == 0 {
					continue
				}
				if raw {
					if t.raw.bit() != 0 {
						negative := t.raw.bit() != 0
						t.data[di] = value
						if negative {
							t.data[di] = -value
						}
						t.setSignificant(fi, negative)
					}
				} else if t.mq.decode(int(zc[f&0xFF])) != 0 {
					t.decodeSign(fi, di, f, value)
				}
				t.flags[fi] |= fVisit
			}
		}
	}
}

func (t *t1) refinementPass(bpno int, vsc, raw bool) {
	half := (int32(1) << bpno) >> 1
	for k := 0; k < t.h; k += 4 {
		kend := min(k+4, t.h)
		for x := range t.w {
			fi := (k+1)*t.stride + x + 1
			di := k*t.w + x
			for y := k; y < kend; y, fi, di = y+1, fi+t.stride, di+t.w {
				f := t.flags[fi]
				if f&(fSig|fVisit) != fSig {
					continue
				}
				if vsc && y == k+3 {
					f &= vscMask
				}
				var b int
				if raw {
					b = t.raw.bit()
				} else {
					ctx := ctxMR
					if f&fRefined != 0 {
						ctx += 2
					} else if f&fNeighbors != 0 {
						ctx++
					}
					b = t.mq.decode(ctx)
				}
				// Move the reconstruction to the midpoint of the upper or
				// lower half of the current interval.
				if (b != 0) == (t.data[di] >= 0) {
					t.data[di] += half
				} else {
					t.data[di] -= half
				}
				t.flags[fi] |= fRefined
			}
		}
	}
}

func (t *t1) cleanupPass(bpno, orient int, vsc bool) {
	one := int32(1) << bpno
	value := one | one>>1
	zc := &zcContext[orient]
	s := t.stride
	for k := 0; k < t.h; k += 4 {
		kend := min(k+4, t.h)
		for x := range t.w {
			fi := (k+1)*s + x + 1
			di := k*t.w + x
			start := k
			if kend-k == 4 {
				// Run-length mode applies when all four samples in the
				// column are insignificant, have no significant neighbors,
				// and were not coded in the significance pass.
				f3 := t.flags[fi+3*s]
				if vsc {
					f3 &= vscMask
				}
				const busy = fSig | fVisit | fNeighbors
				if (t.flags[fi]|t.flags[fi+s]|t.flags[fi+2*s]|f3)&busy == 0 {
					if t.mq.decode(ctxRL) == 0 {
						continue // all four remain insignificant
					}
					run := t.mq.decode(ctxUniform) << 1
					run |= t.mq.decode(ctxUniform)
					rfi, rdi := fi+run*s, di+run*t.w
					f := t.flags[rfi]
					if vsc && run == 3 {
						f &= vscMask
					}
					t.decodeSign(rfi, rdi, f, value)
					start = k + run + 1
				}
			}
			fi += (start - k) * s
			di += (start - k) * t.w
			for y := start; y < kend; y, fi, di = y+1, fi+s, di+t.w {
				f := t.flags[fi]
				if f&(fSig|fVisit) == 0 {
					if vsc && y == k+3 {
						f &= vscMask
					}
					if t.mq.decode(int(zc[f&0xFF])) != 0 {
						t.decodeSign(fi, di, f, value)
					}
				}
				t.flags[fi] &^= fVisit
			}
		}
	}
}
