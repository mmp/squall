package jpeg2000

import (
	"fmt"
	"math/bits"
)

// Tier-2 decoding: packet headers and the order of packets in a tile
// (T.800 Annex B).

// headerReader reads packet header bits. After a 0xFF byte, the most
// significant bit of the following byte is a stuffed 0 (B.10.1).
type headerReader struct {
	data    []byte
	pos     int
	buf     uint32
	ct      int
	overrun bool
}

func (r *headerReader) byteIn() {
	r.buf = (r.buf << 8) & 0xFFFF
	r.ct = 8
	if r.buf == 0xFF00 {
		r.ct = 7
	}
	if r.pos < len(r.data) {
		r.buf |= uint32(r.data[r.pos])
		r.pos++
	} else {
		r.overrun = true
	}
}

func (r *headerReader) bit() int {
	if r.ct == 0 {
		r.byteIn()
	}
	r.ct--
	return int(r.buf>>r.ct) & 1
}

func (r *headerReader) bits(n int) int {
	v := 0
	for range n {
		v = v<<1 | r.bit()
	}
	return v
}

// align skips to the end of the packet header, which includes a byte of
// stuffing bits if the last byte read was 0xFF.
func (r *headerReader) align() {
	if r.buf&0xFF == 0xFF {
		r.byteIn()
	}
	r.ct = 0
}

// tagTree is a tag tree (B.10.2) over a grid of code-blocks. Nodes are stored
// level by level, starting with the leaves.
type tagTree struct {
	parent []int32
	value  []int32
	low    []int32
}

func newTagTree(w, h int) *tagTree {
	n := 0
	for lw, lh := w, h; ; lw, lh = (lw+1)/2, (lh+1)/2 {
		n += lw * lh
		if lw*lh <= 1 {
			break
		}
	}
	t := &tagTree{
		parent: make([]int32, n),
		value:  make([]int32, n),
		low:    make([]int32, n),
	}
	for i := range t.value {
		t.value[i] = 1 << 30
	}
	// Link each level's nodes to their parents in the next level.
	start := 0
	for lw, lh := w, h; lw*lh > 1; lw, lh = (lw+1)/2, (lh+1)/2 {
		next := start + lw*lh
		pw := (lw + 1) / 2
		for y := range lh {
			for x := range lw {
				t.parent[start+y*lw+x] = int32(next + (y/2)*pw + x/2)
			}
		}
		start = next
	}
	t.parent[n-1] = -1
	return t
}

// decode reads bits until it is known whether the leaf's value is below
// threshold, returning true if so.
func (t *tagTree) decode(r *headerReader, leaf int, threshold int32) bool {
	var stack [32]int32
	sp := 0
	node := int32(leaf)
	for t.parent[node] >= 0 {
		stack[sp] = node
		sp++
		node = t.parent[node]
	}
	low := int32(0)
	for {
		if low > t.low[node] {
			t.low[node] = low
		} else {
			low = t.low[node]
		}
		for low < threshold && low < t.value[node] {
			if r.bit() != 0 {
				t.value[node] = low
			} else {
				low++
			}
			if r.overrun {
				return false
			}
		}
		t.low[node] = low
		if sp == 0 {
			break
		}
		sp--
		node = stack[sp]
	}
	return t.value[node] < threshold
}

// numPasses reads the number of new coding passes (Table B.4).
func numPasses(r *headerReader) int {
	if r.bit() == 0 {
		return 1
	}
	if r.bit() == 0 {
		return 2
	}
	if n := r.bits(2); n != 3 {
		return 3 + n
	}
	if n := r.bits(5); n != 31 {
		return 6 + n
	}
	return 37 + r.bits(7)
}

// newSegment starts a new codeword segment for a code-block, setting the
// maximum number of coding passes it can hold (B.10.7.1). With the
// arithmetic coder bypass, the first ten passes are arithmetic coded, and
// after that the significance and refinement passes of each bit-plane form a
// raw segment and the cleanup pass an arithmetic coded one.
func (cb *codeBlock) newSegment(style byte) *segment {
	maxPasses := 109 // more passes than can exist (3*37 - 2)
	if style&styleTermAll != 0 {
		maxPasses = 1
	} else if style&styleBypass != 0 {
		if len(cb.segs) == 0 {
			maxPasses = 10
		} else if prev := cb.segs[len(cb.segs)-1].maxPasses; prev == 1 || prev == 10 {
			maxPasses = 2
		} else {
			maxPasses = 1
		}
	}
	cb.segs = append(cb.segs, &segment{maxPasses: maxPasses})
	return cb.segs[len(cb.segs)-1]
}

// contribution records the data a packet contributes to a codeword segment.
type contribution struct {
	seg    *segment
	passes int
	length int
}

// readPacket decodes the packet for the given layer and precinct of a
// resolution, returning the number of bytes it occupies in data.
func (td *tileDecoder) readPacket(data []byte, layer int, tc *tileComp, res *resolution, p int) (int, error) {
	off := 0
	if td.sop && len(data) >= 6 && data[0] == 0xFF && data[1] == 0x91 {
		off = 6
	}

	r := headerReader{data: data[off:]}
	contribs := td.contribs[:0]
	if r.bit() != 0 {
		for _, b := range res.bands {
			if len(b.precincts) == 0 {
				continue
			}
			prc := &b.precincts[p]
			for k, cb := range prc.cblks {
				var included bool
				if !cb.included {
					included = prc.incl.decode(&r, k, int32(layer+1))
				} else {
					included = r.bit() != 0
				}
				if r.overrun {
					return 0, errTruncated
				}
				if !included {
					continue
				}

				if !cb.included {
					// The number of missing most significant bit-planes.
					zbp := int32(0)
					for !prc.zbp.decode(&r, k, zbp+1) {
						if r.overrun {
							return 0, errTruncated
						}
						zbp++
					}
					cb.numbps = b.mb - int(zbp)
					cb.lblock = 3
					cb.included = true
				}

				n := numPasses(&r)
				for r.bit() != 0 && !r.overrun {
					cb.lblock++
				}

				seg := cb.lastSegment()
				if seg == nil || seg.passes == seg.maxPasses {
					seg = cb.newSegment(tc.coding.style)
				}
				for {
					passes := min(seg.maxPasses-seg.passes, n)
					nbits := cb.lblock + bits.Len(uint(passes)) - 1
					if nbits > 32 {
						return 0, fmt.Errorf("invalid codeword segment length size (%d bits)", nbits)
					}
					contribs = append(contribs, contribution{seg: seg, passes: passes, length: r.bits(nbits)})
					if n -= passes; n == 0 {
						break
					}
					seg = cb.newSegment(tc.coding.style)
				}
				if r.overrun {
					return 0, errTruncated
				}
			}
		}
	}
	r.align()
	if r.overrun {
		return 0, errTruncated
	}
	off += r.pos

	if td.eph && off+2 <= len(data) && data[off] == 0xFF && data[off+1] == 0x92 {
		off += 2
	}

	for _, c := range contribs {
		if c.length > len(data)-off {
			return 0, errTruncated
		}
		c.seg.data = append(c.seg.data, data[off:off+c.length]...)
		c.seg.passes += c.passes
		off += c.length
	}
	td.contribs = contribs
	return off, nil
}

// progression describes one progression through a tile's packets: either
// the default from the COD marker or one from a POC marker.
type progression struct {
	order              int
	layerEnd           int
	resStart, resEnd   int
	compStart, compEnd int
}

// forEachPacket calls fn for each packet of the tile in codestream order.
func (td *tileDecoder) forEachPacket(fn func(layer int, tc *tileComp, res *resolution, p int) error) error {
	// next[c][r][p] is the next layer expected for each precinct; it
	// prevents a packet from being visited twice when progression order
	// changes overlap.
	next := make([][][]int32, len(td.comps))
	maxRes := 0
	for c, tc := range td.comps {
		next[c] = make([][]int32, len(tc.res))
		for r, res := range tc.res {
			next[c][r] = make([]int32, res.pw*res.ph)
		}
		maxRes = max(maxRes, len(tc.res))
	}
	emit := func(l int, c, r, p int) error {
		if l != int(next[c][r][p]) {
			return nil
		}
		next[c][r][p]++
		tc := td.comps[c]
		return fn(l, tc, tc.res[r], p)
	}

	progs := make([]progression, 0, len(td.pocs)+1)
	for _, p := range td.pocs {
		progs = append(progs, progression{
			order:     p.progression,
			layerEnd:  min(p.layerEnd, td.numLayers),
			resStart:  p.resStart,
			resEnd:    min(p.resEnd, maxRes),
			compStart: p.compStart,
			compEnd:   min(p.compEnd, len(td.comps)),
		})
	}
	// Any packets not covered by the progression order changes follow in
	// the default order.
	progs = append(progs, progression{
		order:    td.progression,
		layerEnd: td.numLayers,
		resEnd:   maxRes,
		compEnd:  len(td.comps),
	})

	for _, pg := range progs {
		var err error
		switch pg.order {
		case progLRCP:
			err = td.layerResolution(pg, true, emit)
		case progRLCP:
			err = td.layerResolution(pg, false, emit)
		default:
			err = td.positional(pg, emit)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// layerResolution iterates in LRCP (layerFirst) or RLCP order.
func (td *tileDecoder) layerResolution(pg progression, layerFirst bool, emit func(l, c, r, p int) error) error {
	outer, inner := pg.layerEnd, pg.resEnd
	if !layerFirst {
		outer, inner = pg.resEnd, pg.layerEnd
	}
	for i := 0; i < outer; i++ {
		start := 0
		if layerFirst {
			start = pg.resStart
		} else if i < pg.resStart {
			continue
		}
		for j := start; j < inner; j++ {
			l, r := i, j
			if !layerFirst {
				l, r = j, i
			}
			for c := pg.compStart; c < pg.compEnd; c++ {
				tc := td.comps[c]
				if r >= len(tc.res) {
					continue
				}
				res := tc.res[r]
				for p := range res.pw * res.ph {
					if err := emit(l, c, r, p); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// positional iterates in one of the progressions that step through precinct
// positions on the reference grid: RPCL, PCRL, or CPRL (B.12.1.3-5).
func (td *tileDecoder) positional(pg progression, emit func(l, c, r, p int) error) error {
	// step returns the smallest precinct spacing on the reference grid over
	// the given components and all of their resolutions.
	step := func(c0, c1 int) (int, int) {
		dx, dy := 0, 0
		for c := c0; c < c1; c++ {
			tc := td.comps[c]
			for r := range tc.res {
				ppx, ppy := tc.coding.precinctSize(r)
				level := len(tc.res) - 1 - r
				sx := tc.dx << (ppx + level)
				sy := tc.dy << (ppy + level)
				if dx == 0 || sx < dx {
					dx = sx
				}
				if dy == 0 || sy < dy {
					dy = sy
				}
			}
		}
		return dx, dy
	}

	// visit emits the packets for the precinct of component c at resolution
	// r that starts at reference grid position (x, y), if one does.
	visit := func(x, y, c, r int) error {
		tc := td.comps[c]
		if r < pg.resStart || r >= pg.resEnd || r >= len(tc.res) {
			return nil
		}
		res := tc.res[r]
		if res.pw == 0 || res.ph == 0 {
			return nil
		}
		ppx, ppy := tc.coding.precinctSize(r)
		level := len(tc.res) - 1 - r
		rpx, rpy := ppx+level, ppy+level
		// A precinct starts at a multiple of its size, or at the tile's
		// origin if that falls inside a precinct (B.12.1.3).
		if y%(tc.dy<<rpy) != 0 && (y != td.y0 || (res.y0<<level)%(1<<rpy) == 0) {
			return nil
		}
		if x%(tc.dx<<rpx) != 0 && (x != td.x0 || (res.x0<<level)%(1<<rpx) == 0) {
			return nil
		}
		px := ceilDiv(x, tc.dx<<level)>>ppx - res.x0>>ppx
		py := ceilDiv(y, tc.dy<<level)>>ppy - res.y0>>ppy
		p := px + py*res.pw
		for l := 0; l < pg.layerEnd; l++ {
			if err := emit(l, c, r, p); err != nil {
				return err
			}
		}
		return nil
	}

	// positions calls f for each reference grid position in the tile that
	// is a multiple of the step, along with the tile's origin.
	positions := func(dx, dy int, f func(x, y int) error) error {
		if dx == 0 || dy == 0 {
			return nil
		}
		for y := td.y0; y < td.y1; y += dy - y%dy {
			for x := td.x0; x < td.x1; x += dx - x%dx {
				if err := f(x, y); err != nil {
					return err
				}
			}
		}
		return nil
	}

	switch pg.order {
	case progRPCL:
		dx, dy := step(0, len(td.comps))
		for r := pg.resStart; r < pg.resEnd; r++ {
			err := positions(dx, dy, func(x, y int) error {
				for c := pg.compStart; c < pg.compEnd; c++ {
					if err := visit(x, y, c, r); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	case progPCRL:
		dx, dy := step(0, len(td.comps))
		return positions(dx, dy, func(x, y int) error {
			for c := pg.compStart; c < pg.compEnd; c++ {
				for r := pg.resStart; r < pg.resEnd; r++ {
					if err := visit(x, y, c, r); err != nil {
						return err
					}
				}
			}
			return nil
		})
	case progCPRL:
		for c := pg.compStart; c < pg.compEnd; c++ {
			dx, dy := step(c, c+1)
			err := positions(dx, dy, func(x, y int) error {
				for r := pg.resStart; r < pg.resEnd; r++ {
					if err := visit(x, y, c, r); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
