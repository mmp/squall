// Package jpeg2000 decodes JPEG 2000 Part 1 codestreams (ITU-T T.800 |
// ISO/IEC 15444-1), as used by GRIB2 data representation template 5.40.
//
// The decoder supports the Part 1 codestream syntax: multiple tiles and
// tile-parts, multiple components (including the reversible and irreversible
// component transforms), all five progression orders and progression order
// changes, user-defined precincts, quality layers, all code-block coding
// styles, SOP and EPH markers, max-shift regions of interest, and both the
// 5/3 reversible and 9/7 irreversible wavelet transforms. Packed packet
// headers (PPM/PPT), Part 2 extensions, and HTJ2K (Part 15) are not
// supported. Codestreams wrapped in a JP2 file are accepted.
package jpeg2000

import (
	"errors"
	"fmt"
	"math"
)

var errTruncated = errors.New("jpeg2000: truncated codestream")

// Image is a decoded JPEG 2000 image.
type Image struct {
	Width, Height int // size of the image area on the reference grid
	Components    []Component
}

// Component is one decoded image component.
type Component struct {
	Width, Height int
	Precision     int  // bits per sample
	Signed        bool // whether samples are signed
	Data          []int32
}

// Config describes an image's dimensions without decoding it.
type Config struct {
	Width, Height int         // size of the image area on the reference grid
	Components    []Component // the components, without their Data
}

// DecodeConfig returns the image's dimensions and component formats after
// reading only the codestream's main header.
func DecodeConfig(data []byte) (*Config, error) {
	cs, err := unwrapJP2(data)
	if err != nil {
		return nil, fmt.Errorf("jpeg2000: %w", err)
	}
	d := &decoder{r: reader{data: cs}}
	if err := d.readMainHeader(); err != nil {
		return nil, fmt.Errorf("jpeg2000: %w", err)
	}
	img := d.newImage(false)
	return &Config{Width: img.Width, Height: img.Height, Components: img.Components}, nil
}

// Decode decodes a JPEG 2000 codestream (or JP2 file).
func Decode(data []byte) (*Image, error) {
	cs, err := unwrapJP2(data)
	if err != nil {
		return nil, fmt.Errorf("jpeg2000: %w", err)
	}
	d := &decoder{r: reader{data: cs}}
	if err := d.readMainHeader(); err != nil {
		return nil, fmt.Errorf("jpeg2000: %w", err)
	}
	if err := d.readTileParts(); err != nil {
		return nil, fmt.Errorf("jpeg2000: %w", err)
	}
	img := d.newImage(true)
	for i, t := range d.tiles {
		if t == nil {
			return nil, fmt.Errorf("jpeg2000: tile %d is missing", i)
		}
		if err := d.decodeTile(t, img); err != nil {
			return nil, fmt.Errorf("jpeg2000: tile %d: %w", i, err)
		}
	}
	return img, nil
}

type decoder struct {
	r     reader
	siz   *siz
	main  header
	tiles []*tile
}

func (d *decoder) readMainHeader() error {
	m, err := d.r.u16()
	if err != nil || m != markerSOC {
		return errors.New("missing SOC marker")
	}
	if m, err = d.r.u16(); err != nil || m != markerSIZ {
		return errors.New("missing SIZ marker")
	}
	seg, err := d.r.segment()
	if err != nil {
		return err
	}
	if d.siz, err = parseSIZ(seg); err != nil {
		return fmt.Errorf("SIZ: %w", err)
	}
	if d.siz.rsiz&0x8000 != 0 {
		return fmt.Errorf("unsupported Part 2 capabilities (Rsiz 0x%04X)", d.siz.rsiz)
	}
	d.tiles = make([]*tile, d.siz.numXTiles*d.siz.numYTiles)

	for {
		m, err := d.r.u16()
		if err != nil {
			return err
		}
		if m == markerSOT {
			d.r.pos -= 2
			break
		}
		if m < 0xFF00 {
			return fmt.Errorf("invalid marker 0x%04X in main header", m)
		}
		seg, err := d.r.segment()
		if err != nil {
			return err
		}
		if err := parseHeaderSegment(m, seg, &d.main, len(d.siz.comps)); err != nil {
			return err
		}
	}
	if d.main.cod == nil {
		return errors.New("missing COD marker")
	}
	if d.main.qcd == nil {
		return errors.New("missing QCD marker")
	}
	return nil
}

// readTileParts reads each tile-part header and gathers the tile data.
func (d *decoder) readTileParts() error {
	data := d.r.data
	for d.r.pos+2 <= len(data) {
		start := d.r.pos
		m, _ := d.r.u16()
		if m == markerEOC {
			return nil
		}
		if m != markerSOT {
			return fmt.Errorf("expected SOT marker, found 0x%04X", m)
		}
		seg, err := d.r.segment()
		if err != nil {
			return err
		}
		isot, err1 := seg.u16()
		psot, err2 := seg.u32()
		if err1 != nil || err2 != nil {
			return errTruncated
		}
		if isot >= len(d.tiles) {
			return fmt.Errorf("invalid tile index %d", isot)
		}
		end := start + psot
		if psot == 0 {
			// The last tile-part extends to the EOC marker.
			end = len(data)
			if end >= 2 && data[end-2] == 0xFF && data[end-1] == 0xD9 {
				end -= 2
			}
		}
		if end > len(data) || end < d.r.pos {
			return fmt.Errorf("invalid tile-part length %d", psot)
		}

		t := d.tiles[isot]
		if t == nil {
			t = &tile{index: isot}
			d.tiles[isot] = t
		}
		for {
			m, err := d.r.u16()
			if err != nil {
				return err
			}
			if m == markerSOD {
				break
			}
			if m < 0xFF00 || d.r.pos >= end {
				return fmt.Errorf("invalid marker 0x%04X in tile-part header", m)
			}
			seg, err := d.r.segment()
			if err != nil {
				return err
			}
			if err := parseHeaderSegment(m, seg, &t.hdr, len(d.siz.comps)); err != nil {
				return err
			}
		}
		if d.r.pos > end {
			return errTruncated
		}
		t.data = append(t.data, data[d.r.pos:end]...)
		d.r.pos = end
	}
	return nil
}

func (d *decoder) newImage(alloc bool) *Image {
	s := d.siz
	img := &Image{Width: s.xsiz - s.xosiz, Height: s.ysiz - s.yosiz}
	for _, c := range s.comps {
		w := ceilDiv(s.xsiz, c.dx) - ceilDiv(s.xosiz, c.dx)
		h := ceilDiv(s.ysiz, c.dy) - ceilDiv(s.yosiz, c.dy)
		comp := Component{Width: w, Height: h, Precision: c.prec, Signed: c.signed}
		if alloc {
			comp.Data = make([]int32, w*h)
		}
		img.Components = append(img.Components, comp)
	}
	return img
}

// Tile decoding structures. Coordinates are absolute (on the tile-component,
// resolution, or subband grids) and ranges are half-open.

type tileDecoder struct {
	x0, y0, x1, y1 int // tile area on the reference grid
	comps          []*tileComp
	numLayers      int
	progression    int
	sop, eph       bool
	mct            byte
	pocs           []poc
	contribs       []contribution
}

type tileComp struct {
	x0, y0, x1, y1 int
	dx, dy         int // subsampling factors
	prec           int
	signed         bool
	coding         *compCoding
	roiShift       int
	res            []*resolution
}

type resolution struct {
	x0, y0, x1, y1 int
	pw, ph         int // number of precincts
	bands          []*band
}

type band struct {
	orient         int
	x0, y0, x1, y1 int
	mb             int     // number of magnitude bit-planes (M_b)
	step           float32 // quantization step size, for the 9/7 transform
	precincts      []precinct
	coeffs         []int32   // for the 5/3 transform
	fcoeffs        []float32 // for the 9/7 transform
}

type precinct struct {
	cblks     []*codeBlock
	incl, zbp *tagTree
}

type codeBlock struct {
	x0, y0, x1, y1 int
	included       bool // whether it has been included in a packet yet
	numbps         int  // number of bit-planes to decode
	lblock         int
	segs           []*segment
}

func (cb *codeBlock) lastSegment() *segment {
	if len(cb.segs) == 0 {
		return nil
	}
	return cb.segs[len(cb.segs)-1]
}

// segment is a codeword segment: a run of coding passes terminated
// together, possibly spread across several packets.
type segment struct {
	data      []byte
	passes    int
	maxPasses int
}

// gains are the log2 subband gains (Table E.1) by orientation.
var gains = [4]int{bandLL: 0, bandHL: 1, bandLH: 1, bandHH: 2}

func (d *decoder) decodeTile(t *tile, img *Image) error {
	s := d.siz
	p, q := t.index%s.numXTiles, t.index/s.numXTiles
	td := &tileDecoder{
		x0: max(s.xtosiz+p*s.xtsiz, s.xosiz),
		y0: max(s.ytosiz+q*s.ytsiz, s.yosiz),
		x1: min(s.xtosiz+(p+1)*s.xtsiz, s.xsiz),
		y1: min(s.ytosiz+(q+1)*s.ytsiz, s.ysiz),
	}

	// Resolve the coding parameters: tile-part header values override main
	// header values, and component-specific values override defaults.
	c := d.main.cod
	if t.hdr.cod != nil {
		c = t.hdr.cod
	}
	td.numLayers, td.progression, td.sop, td.eph, td.mct = c.numLayers, c.progression, c.sop, c.eph, c.mct
	td.pocs = d.main.pocs
	if t.hdr.pocs != nil {
		td.pocs = t.hdr.pocs
	}

	for ci, cs := range s.comps {
		coding := &d.main.cod.comp
		if cc := d.main.coc[ci]; cc != nil {
			coding = cc
		}
		if t.hdr.cod != nil {
			coding = &t.hdr.cod.comp
		}
		if cc := t.hdr.coc[ci]; cc != nil {
			coding = cc
		}
		qu := d.main.qcd
		if qq := d.main.qcc[ci]; qq != nil {
			qu = qq
		}
		if t.hdr.qcd != nil {
			qu = t.hdr.qcd
		}
		if qq := t.hdr.qcc[ci]; qq != nil {
			qu = qq
		}
		roi := d.main.rgn[ci]
		if shift, ok := t.hdr.rgn[ci]; ok {
			roi = shift
		}

		tc := &tileComp{
			x0: ceilDiv(td.x0, cs.dx), y0: ceilDiv(td.y0, cs.dy),
			x1: ceilDiv(td.x1, cs.dx), y1: ceilDiv(td.y1, cs.dy),
			dx: cs.dx, dy: cs.dy,
			prec: cs.prec, signed: cs.signed,
			coding: coding, roiShift: roi,
		}
		if err := tc.build(qu); err != nil {
			return fmt.Errorf("component %d: %w", ci, err)
		}
		td.comps = append(td.comps, tc)
	}

	pos := 0
	err := td.forEachPacket(func(layer int, tc *tileComp, res *resolution, p int) error {
		n, err := td.readPacket(t.data[pos:], layer, tc, res, p)
		pos += n
		return err
	})
	if err != nil {
		return err
	}

	var dec t1
	for ci, tc := range td.comps {
		if err := tc.decodeCodeBlocks(&dec); err != nil {
			return fmt.Errorf("component %d: %w", ci, err)
		}
	}
	return td.reconstruct(d, img)
}

// build computes the geometry of the tile-component's resolutions,
// subbands, precincts, and code-blocks (B.5-B.7).
func (tc *tileComp) build(q *quant) error {
	cc := tc.coding
	nl := cc.numLevels
	for r := 0; r <= nl; r++ {
		level := nl - r
		res := &resolution{
			x0: ceilDivPow2(tc.x0, level), y0: ceilDivPow2(tc.y0, level),
			x1: ceilDivPow2(tc.x1, level), y1: ceilDivPow2(tc.y1, level),
		}
		ppx, ppy := cc.precinctSize(r)
		if res.x1 > res.x0 && res.y1 > res.y0 {
			res.pw = ceilDivPow2(res.x1, ppx) - res.x0>>ppx
			res.ph = ceilDivPow2(res.y1, ppy) - res.y0>>ppy
		}
		if res.pw*res.ph > maxSamples {
			return fmt.Errorf("too many precincts")
		}

		// Precinct and code-block sizes, in subband coordinates.
		cbgw, cbgh := ppx, ppy
		orients := []int{bandLL}
		if r > 0 {
			cbgw, cbgh = ppx-1, ppy-1
			orients = []int{bandHL, bandLH, bandHH}
		}
		xcb, ycb := min(cc.xcb, cbgw), min(cc.ycb, cbgh)

		for i, orient := range orients {
			b := &band{orient: orient}
			if r == 0 {
				b.x0, b.y0, b.x1, b.y1 = res.x0, res.y0, res.x1, res.y1
			} else {
				// Equation B-15, for decomposition level nb.
				nb := nl - r + 1
				xob, yob := orient&1, orient>>1
				b.x0 = ceilDivPow2(tc.x0-xob<<(nb-1), nb)
				b.y0 = ceilDivPow2(tc.y0-yob<<(nb-1), nb)
				b.x1 = ceilDivPow2(tc.x1-xob<<(nb-1), nb)
				b.y1 = ceilDivPow2(tc.y1-yob<<(nb-1), nb)
			}

			index := 0
			if r > 0 {
				index = 3*(r-1) + 1 + i
			}
			exp, mant, err := q.stepParams(index, r)
			if err != nil {
				return err
			}
			b.mb = q.guardBits + exp - 1
			if b.mb+tc.roiShift > 30 {
				return fmt.Errorf("too many bit-planes (%d)", b.mb+tc.roiShift)
			}
			if !cc.reversible {
				rb := tc.prec + gains[orient]
				b.step = float32(math.Ldexp(1+float64(mant)/2048, rb-exp))
			}

			w, h := b.x1-b.x0, b.y1-b.y0
			if w > 0 && h > 0 {
				if cc.reversible {
					b.coeffs = make([]int32, w*h)
				} else {
					b.fcoeffs = make([]float32, w*h)
				}
				b.precincts = make([]precinct, res.pw*res.ph)
				for p := range b.precincts {
					b.buildPrecinct(&b.precincts[p], res, p, r, cbgw, cbgh, xcb, ycb)
				}
			}
			res.bands = append(res.bands, b)
		}
		tc.res = append(tc.res, res)
	}
	return nil
}

// buildPrecinct sets up the code-blocks of the band that fall within
// precinct p of its resolution.
func (b *band) buildPrecinct(prc *precinct, res *resolution, p, r, cbgw, cbgh, xcb, ycb int) {
	ppx, ppy := cbgw, cbgh
	if r > 0 {
		ppx, ppy = cbgw+1, cbgh+1
	}
	// The precinct's origin in resolution coordinates, mapped to the band.
	px0 := (res.x0>>ppx + p%res.pw) << ppx
	py0 := (res.y0>>ppy + p/res.pw) << ppy
	if r > 0 {
		px0, py0 = px0>>1, py0>>1
	}
	x0, y0 := max(px0, b.x0), max(py0, b.y0)
	x1, y1 := min(px0+1<<cbgw, b.x1), min(py0+1<<cbgh, b.y1)
	if x0 >= x1 || y0 >= y1 {
		return
	}
	cx0, cy0 := x0>>xcb, y0>>ycb
	cw, ch := ceilDivPow2(x1, xcb)-cx0, ceilDivPow2(y1, ycb)-cy0
	prc.cblks = make([]*codeBlock, 0, cw*ch)
	for j := range ch {
		for i := range cw {
			prc.cblks = append(prc.cblks, &codeBlock{
				x0: max((cx0+i)<<xcb, x0), y0: max((cy0+j)<<ycb, y0),
				x1: min((cx0+i+1)<<xcb, x1), y1: min((cy0+j+1)<<ycb, y1),
			})
		}
	}
	prc.incl = newTagTree(cw, ch)
	prc.zbp = newTagTree(cw, ch)
}

// decodeCodeBlocks runs the tier-1 decoder on each code-block and stores
// the dequantized coefficients in the subbands.
func (tc *tileComp) decodeCodeBlocks(dec *t1) error {
	for _, res := range tc.res {
		for _, b := range res.bands {
			bw := b.x1 - b.x0
			for _, prc := range b.precincts {
				for _, cb := range prc.cblks {
					if err := dec.decode(cb, b.orient, tc.coding.style, tc.roiShift); err != nil {
						return err
					}
					w := cb.x1 - cb.x0
					for y := cb.y0; y < cb.y1; y++ {
						src := dec.data[(y-cb.y0)*w : (y-cb.y0+1)*w]
						off := (y-b.y0)*bw + cb.x0 - b.x0
						if b.coeffs != nil {
							dst := b.coeffs[off : off+w]
							for i, v := range src {
								dst[i] = v / 2
							}
						} else {
							dst := b.fcoeffs[off : off+w]
							scale := b.step / 2
							for i, v := range src {
								dst[i] = float32(v) * scale
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// reconstruct applies the inverse wavelet and component transforms and the
// DC level shift, and stores the tile's samples in the image.
func (td *tileDecoder) reconstruct(d *decoder, img *Image) error {
	ints := make([][]int32, len(td.comps))
	floats := make([][]float32, len(td.comps))
	for ci, tc := range td.comps {
		if tc.coding.reversible {
			buf := make([]int64, max(tc.x1-tc.x0, tc.y1-tc.y0)+4)
			cur := tc.res[0].bands[0].coeffs
			for r := 1; r < len(tc.res); r++ {
				cur = synthesize(tc.res[r], tc.res[r-1], cur,
					func(b *band) []int32 { return b.coeffs },
					func(x []int32, i0 int) { inverse53(x, i0, buf) })
			}
			ints[ci] = cur
		} else {
			buf := make([]float64, max(tc.x1-tc.x0, tc.y1-tc.y0)+8)
			cur := tc.res[0].bands[0].fcoeffs
			for r := 1; r < len(tc.res); r++ {
				cur = synthesize(tc.res[r], tc.res[r-1], cur,
					func(b *band) []float32 { return b.fcoeffs },
					func(x []float32, i0 int) { inverse97(x, i0, buf) })
			}
			floats[ci] = cur
		}
	}

	if td.mct != 0 && len(td.comps) >= 3 {
		if err := td.inverseMCT(ints, floats); err != nil {
			return err
		}
	}

	s := d.siz
	for ci, tc := range td.comps {
		comp := &img.Components[ci]
		w := tc.x1 - tc.x0
		ox := tc.x0 - ceilDiv(s.xosiz, tc.dx)
		oy := tc.y0 - ceilDiv(s.yosiz, tc.dy)
		lo, hi := int64(0), int64(1)<<tc.prec-1
		shift := int64(1) << (tc.prec - 1)
		if tc.signed {
			lo, hi, shift = -shift, shift-1, 0
		}
		for y := tc.y0; y < tc.y1; y++ {
			dst := comp.Data[(oy+y-tc.y0)*comp.Width+ox:]
			for x := range w {
				var v int64
				if ints[ci] != nil {
					v = int64(ints[ci][(y-tc.y0)*w+x])
				} else if floats[ci] != nil {
					v = int64(math.RoundToEven(float64(floats[ci][(y-tc.y0)*w+x])))
				}
				dst[x] = int32(max(lo, min(hi, v+shift)))
			}
		}
	}
	return nil
}

// inverseMCT applies the inverse reversible (RCT) or irreversible (ICT)
// component transform to the first three components (Annex G).
func (td *tileDecoder) inverseMCT(ints [][]int32, floats [][]float32) error {
	c0, c1, c2 := td.comps[0], td.comps[1], td.comps[2]
	for _, c := range []*tileComp{c1, c2} {
		if c.x0 != c0.x0 || c.y0 != c0.y0 || c.x1 != c0.x1 || c.y1 != c0.y1 ||
			c.coding.reversible != c0.coding.reversible {
			return errors.New("multiple component transform requires identical components")
		}
	}
	if c0.coding.reversible {
		y0, y1, y2 := ints[0], ints[1], ints[2]
		for i := range y0 {
			g := y0[i] - (y1[i]+y2[i])>>2
			y0[i], y1[i], y2[i] = y2[i]+g, g, y1[i]+g
		}
	} else {
		y0, y1, y2 := floats[0], floats[1], floats[2]
		for i := range y0 {
			y, cb, cr := y0[i], y1[i], y2[i]
			y0[i] = y + 1.402*cr
			y1[i] = y - 0.34413*cb - 0.71414*cr
			y2[i] = y + 1.772*cb
		}
	}
	return nil
}

func ceilDiv(a, b int) int {
	if a >= 0 {
		return (a + b - 1) / b
	}
	return -(-a / b)
}

func ceilDivPow2(a, e int) int {
	return -(-a >> e)
}
