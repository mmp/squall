package jpeg2000

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Marker codes (ITU-T T.800 Table A.2).
const (
	markerSOC = 0xFF4F // start of codestream
	markerCAP = 0xFF50 // extended capabilities (Part 15, HTJ2K)
	markerSIZ = 0xFF51 // image and tile size
	markerCOD = 0xFF52 // coding style default
	markerCOC = 0xFF53 // coding style component
	markerTLM = 0xFF55 // tile-part lengths
	markerPLM = 0xFF57 // packet length, main header
	markerPLT = 0xFF58 // packet length, tile-part header
	markerQCD = 0xFF5C // quantization default
	markerQCC = 0xFF5D // quantization component
	markerRGN = 0xFF5E // region of interest
	markerPOC = 0xFF5F // progression order change
	markerPPM = 0xFF60 // packed packet headers, main header
	markerPPT = 0xFF61 // packed packet headers, tile-part header
	markerCRG = 0xFF63 // component registration
	markerCOM = 0xFF64 // comment
	markerSOT = 0xFF90 // start of tile-part
	markerSOP = 0xFF91 // start of packet
	markerEPH = 0xFF92 // end of packet header
	markerSOD = 0xFF93 // start of data
	markerEOC = 0xFFD9 // end of codestream
)

// Code-block style flags (Table A.19).
const (
	styleBypass   = 0x01 // selective arithmetic coding bypass
	styleReset    = 0x02 // reset context probabilities after each pass
	styleTermAll  = 0x04 // terminate after each coding pass
	styleVSC      = 0x08 // vertically causal context formation
	stylePredTerm = 0x10 // predictable termination
	styleSegSym   = 0x20 // segmentation symbols
	styleHT       = 0x40 // HTJ2K block coder (Part 15)
)

// Progression orders (Table A.16).
const (
	progLRCP = iota
	progRLCP
	progRPCL
	progPCRL
	progCPRL
)

// Quantization styles (Table A.28).
const (
	quantNone     = 0
	quantDerived  = 1
	quantExpanded = 2
)

// Limits on what the decoder will accept; they bound memory use for
// malformed input while comfortably covering real images.
const (
	maxComponents = 16384
	maxLevels     = 32
	maxPrecision  = 31
	maxSamples    = 1 << 30 // per component
)

// siz holds the image and tile size parameters (SIZ marker, A.5.1).
type siz struct {
	rsiz           uint16
	xsiz, ysiz     int // reference grid size
	xosiz, yosiz   int // image area offset
	xtsiz, ytsiz   int // tile size
	xtosiz, ytosiz int // tile grid offset
	comps          []compSiz
	numXTiles      int
	numYTiles      int
}

type compSiz struct {
	prec   int  // bit depth
	signed bool // whether samples are signed
	dx, dy int  // horizontal and vertical subsampling (XRsiz, YRsiz)
}

// compCoding holds the per-component coding style parameters from the SPcod
// or SPcoc fields of a COD or COC marker (Table A.15).
type compCoding struct {
	numLevels  int  // number of decomposition levels (NL)
	xcb, ycb   int  // code-block width and height exponents
	style      byte // code-block style flags
	reversible bool // 5/3 reversible (true) or 9/7 irreversible wavelet
	precincts  []byte
}

// precinctSize returns the precinct width and height exponents (PPx, PPy)
// for resolution level r.
func (cc *compCoding) precinctSize(r int) (int, int) {
	if cc.precincts == nil {
		return 15, 15
	}
	return int(cc.precincts[r] & 0xF), int(cc.precincts[r] >> 4)
}

// cod holds the parameters of a COD marker (A.6.1).
type cod struct {
	sop, eph    bool
	progression int
	numLayers   int
	mct         byte
	comp        compCoding
}

// quant holds quantization parameters from a QCD or QCC marker (A.6.4).
type quant struct {
	style     int
	guardBits int
	exps      []int // exponent (epsilon_b) per subband
	mants     []int // mantissa (mu_b) per subband
}

// stepParams returns the quantization exponent and mantissa for a subband;
// index is the subband's position in the QCD ordering (0 for the LL band,
// then HL, LH, HH for each resolution from lowest to highest) and r is its
// resolution level.
func (q *quant) stepParams(index, r int) (exp, mant int, err error) {
	if q.style == quantDerived {
		// Equation E-5: epsilon_b = epsilon_0 - NL + n_b, where n_b is the
		// decomposition level of the subband.
		exp = q.exps[0]
		if r > 0 {
			exp -= r - 1
		}
		return max(exp, 0), q.mants[0], nil
	}
	if index >= len(q.exps) {
		return 0, 0, fmt.Errorf("quantization parameters missing for subband %d", index)
	}
	return q.exps[index], q.mants[index], nil
}

// poc is one progression order change (A.6.6). Layer, resolution, and
// component ranges are half-open.
type poc struct {
	resStart, compStart int
	layerEnd            int
	resEnd, compEnd     int
	progression         int
}

// header collects the coding parameters that can appear in either the main
// header or a tile-part header. Within a tile, tile-part header values take
// precedence over main header values, and within each header component
// specific values (COC, QCC) take precedence over defaults (COD, QCD).
type header struct {
	cod  *cod
	coc  map[int]*compCoding
	qcd  *quant
	qcc  map[int]*quant
	rgn  map[int]int
	pocs []poc
}

// tile accumulates the tile-part headers and data for one tile.
type tile struct {
	index int
	hdr   header
	data  []byte
}

// marker segment reader over the codestream.
type reader struct {
	data []byte
	pos  int
}

func (r *reader) u8() (int, error) {
	if r.pos+1 > len(r.data) {
		return 0, errTruncated
	}
	v := r.data[r.pos]
	r.pos++
	return int(v), nil
}

func (r *reader) u16() (int, error) {
	if r.pos+2 > len(r.data) {
		return 0, errTruncated
	}
	v := binary.BigEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return int(v), nil
}

func (r *reader) u32() (int, error) {
	if r.pos+4 > len(r.data) {
		return 0, errTruncated
	}
	v := binary.BigEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return int(v), nil
}

// segment reads the length field of a marker segment and returns a reader
// over its parameters.
func (r *reader) segment() (*reader, error) {
	n, err := r.u16()
	if err != nil {
		return nil, err
	}
	if n < 2 || r.pos+n-2 > len(r.data) {
		return nil, fmt.Errorf("invalid marker segment length %d", n)
	}
	s := &reader{data: r.data[r.pos : r.pos+n-2]}
	r.pos += n - 2
	return s, nil
}

// compIndex reads a component index, which is one byte when there are fewer
// than 257 components and two bytes otherwise.
func (r *reader) compIndex(numComps int) (int, error) {
	if numComps < 257 {
		return r.u8()
	}
	return r.u16()
}

// unwrapJP2 returns the contiguous codestream from a JP2 file, or the input
// unchanged if it is not a JP2 file. GRIB2 specifies a raw codestream, but
// accepting JP2 costs little.
func unwrapJP2(data []byte) ([]byte, error) {
	sig := []byte{0, 0, 0, 0x0C, 'j', 'P', ' ', ' ', 0x0D, 0x0A, 0x87, 0x0A}
	if !bytes.HasPrefix(data, sig) {
		return data, nil
	}
	pos := 0
	for pos+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[pos:]))
		boxType := string(data[pos+4 : pos+8])
		hdr := 8
		switch length {
		case 0:
			length = len(data) - pos
		case 1:
			if pos+16 > len(data) {
				return nil, errTruncated
			}
			l64 := binary.BigEndian.Uint64(data[pos+8:])
			if l64 > uint64(len(data)-pos) {
				return nil, errTruncated
			}
			length, hdr = int(l64), 16
		}
		if length < hdr || pos+length > len(data) {
			return nil, fmt.Errorf("invalid JP2 box length %d", length)
		}
		if boxType == "jp2c" {
			return data[pos+hdr : pos+length], nil
		}
		pos += length
	}
	return nil, fmt.Errorf("JP2 file has no codestream box")
}

func parseSIZ(r *reader) (*siz, error) {
	var s siz
	var err error
	rsiz, err := r.u16()
	if err != nil {
		return nil, err
	}
	s.rsiz = uint16(rsiz)
	vals := make([]int, 8)
	for i := range vals {
		if vals[i], err = r.u32(); err != nil {
			return nil, err
		}
	}
	s.xsiz, s.ysiz, s.xosiz, s.yosiz = vals[0], vals[1], vals[2], vals[3]
	s.xtsiz, s.ytsiz, s.xtosiz, s.ytosiz = vals[4], vals[5], vals[6], vals[7]

	if s.xsiz <= s.xosiz || s.ysiz <= s.yosiz {
		return nil, fmt.Errorf("empty image area (%d,%d)-(%d,%d)", s.xosiz, s.yosiz, s.xsiz, s.ysiz)
	}
	if s.xtsiz == 0 || s.ytsiz == 0 || s.xtosiz > s.xosiz || s.ytosiz > s.yosiz ||
		s.xtosiz+s.xtsiz <= s.xosiz || s.ytosiz+s.ytsiz <= s.yosiz {
		return nil, fmt.Errorf("invalid tile grid: size %dx%d, offset (%d,%d)", s.xtsiz, s.ytsiz, s.xtosiz, s.ytosiz)
	}
	s.numXTiles = ceilDiv(s.xsiz-s.xtosiz, s.xtsiz)
	s.numYTiles = ceilDiv(s.ysiz-s.ytosiz, s.ytsiz)
	if s.numXTiles*s.numYTiles > 65535 {
		return nil, fmt.Errorf("too many tiles (%d)", s.numXTiles*s.numYTiles)
	}

	nc, err := r.u16()
	if err != nil {
		return nil, err
	}
	if nc == 0 || nc > maxComponents {
		return nil, fmt.Errorf("invalid number of components %d", nc)
	}
	s.comps = make([]compSiz, nc)
	for i := range s.comps {
		ssiz, err1 := r.u8()
		dx, err2 := r.u8()
		dy, err3 := r.u8()
		if err1 != nil || err2 != nil || err3 != nil {
			return nil, errTruncated
		}
		c := compSiz{prec: ssiz&0x7F + 1, signed: ssiz&0x80 != 0, dx: dx, dy: dy}
		if c.prec > maxPrecision {
			return nil, fmt.Errorf("component %d: unsupported precision %d", i, c.prec)
		}
		if dx == 0 || dy == 0 {
			return nil, fmt.Errorf("component %d: invalid subsampling %dx%d", i, dx, dy)
		}
		w := ceilDiv(s.xsiz, dx) - ceilDiv(s.xosiz, dx)
		h := ceilDiv(s.ysiz, dy) - ceilDiv(s.yosiz, dy)
		if w*h > maxSamples {
			return nil, fmt.Errorf("component %d: image too large (%dx%d)", i, w, h)
		}
		s.comps[i] = c
	}
	return &s, nil
}

// parseCompCoding parses the SPcod/SPcoc parameters; hasPrecincts is the
// "precincts defined" bit of Scod/Scoc.
func parseCompCoding(r *reader, hasPrecincts bool) (*compCoding, error) {
	var vals [5]int
	for i := range vals {
		v, err := r.u8()
		if err != nil {
			return nil, err
		}
		vals[i] = v
	}
	cc := &compCoding{
		numLevels:  vals[0],
		xcb:        vals[1] + 2,
		ycb:        vals[2] + 2,
		style:      byte(vals[3]),
		reversible: vals[4] == 1,
	}
	if cc.numLevels > maxLevels {
		return nil, fmt.Errorf("too many decomposition levels (%d)", cc.numLevels)
	}
	if cc.xcb > 10 || cc.ycb > 10 || cc.xcb+cc.ycb > 12 {
		return nil, fmt.Errorf("invalid code-block size 2^%d x 2^%d", cc.xcb, cc.ycb)
	}
	if cc.style&styleHT != 0 {
		return nil, fmt.Errorf("HTJ2K (Part 15) code-blocks are not supported")
	}
	if vals[4] > 1 {
		return nil, fmt.Errorf("unsupported wavelet transform %d", vals[4])
	}
	if hasPrecincts {
		cc.precincts = make([]byte, cc.numLevels+1)
		for i := range cc.precincts {
			v, err := r.u8()
			if err != nil {
				return nil, err
			}
			cc.precincts[i] = byte(v)
			if i > 0 && (v&0xF == 0 || v>>4 == 0) {
				return nil, fmt.Errorf("invalid precinct size for resolution %d", i)
			}
		}
	}
	return cc, nil
}

func parseCOD(r *reader) (*cod, error) {
	scod, err := r.u8()
	if err != nil {
		return nil, err
	}
	prog, err1 := r.u8()
	layers, err2 := r.u16()
	mct, err3 := r.u8()
	if err1 != nil || err2 != nil || err3 != nil {
		return nil, errTruncated
	}
	if prog > progCPRL {
		return nil, fmt.Errorf("invalid progression order %d", prog)
	}
	if layers == 0 {
		return nil, fmt.Errorf("invalid number of layers 0")
	}
	cc, err := parseCompCoding(r, scod&1 != 0)
	if err != nil {
		return nil, err
	}
	return &cod{
		sop:         scod&2 != 0,
		eph:         scod&4 != 0,
		progression: prog,
		numLayers:   layers,
		mct:         byte(mct),
		comp:        *cc,
	}, nil
}

func parseCOC(r *reader, numComps int) (int, *compCoding, error) {
	c, err := r.compIndex(numComps)
	if err != nil {
		return 0, nil, err
	}
	if c >= numComps {
		return 0, nil, fmt.Errorf("COC for nonexistent component %d", c)
	}
	scoc, err := r.u8()
	if err != nil {
		return 0, nil, err
	}
	cc, err := parseCompCoding(r, scoc&1 != 0)
	return c, cc, err
}

func parseQuant(r *reader) (*quant, error) {
	sq, err := r.u8()
	if err != nil {
		return nil, err
	}
	q := &quant{style: sq & 0x1F, guardBits: sq >> 5}
	switch q.style {
	case quantNone:
		for len(r.data)-r.pos > 0 {
			v, _ := r.u8()
			q.exps = append(q.exps, v>>3)
			q.mants = append(q.mants, 0)
		}
	case quantDerived, quantExpanded:
		for len(r.data)-r.pos >= 2 {
			v, _ := r.u16()
			q.exps = append(q.exps, v>>11)
			q.mants = append(q.mants, v&0x7FF)
			if q.style == quantDerived {
				break
			}
		}
	default:
		return nil, fmt.Errorf("invalid quantization style %d", q.style)
	}
	if len(q.exps) == 0 {
		return nil, fmt.Errorf("quantization marker has no step sizes")
	}
	return q, nil
}

func parseQCC(r *reader, numComps int) (int, *quant, error) {
	c, err := r.compIndex(numComps)
	if err != nil {
		return 0, nil, err
	}
	if c >= numComps {
		return 0, nil, fmt.Errorf("QCC for nonexistent component %d", c)
	}
	q, err := parseQuant(r)
	return c, q, err
}

func parseRGN(r *reader, numComps int) (int, int, error) {
	c, err := r.compIndex(numComps)
	if err != nil {
		return 0, 0, err
	}
	if c >= numComps {
		return 0, 0, fmt.Errorf("RGN for nonexistent component %d", c)
	}
	style, err1 := r.u8()
	shift, err2 := r.u8()
	if err1 != nil || err2 != nil {
		return 0, 0, errTruncated
	}
	if style != 0 {
		return 0, 0, fmt.Errorf("unsupported ROI style %d", style)
	}
	return c, shift, nil
}

func parsePOC(r *reader, numComps int) ([]poc, error) {
	var pocs []poc
	for len(r.data)-r.pos > 0 {
		var p poc
		var err error
		var vals [6]int
		for i := range vals {
			switch i {
			case 1, 4:
				vals[i], err = r.compIndex(numComps)
			case 2:
				vals[i], err = r.u16()
			default:
				vals[i], err = r.u8()
			}
			if err != nil {
				return nil, err
			}
		}
		p.resStart, p.compStart, p.layerEnd = vals[0], vals[1], vals[2]
		p.resEnd, p.compEnd, p.progression = vals[3], vals[4], vals[5]
		if p.compEnd == 0 && numComps < 257 {
			p.compEnd = 256
		}
		if p.progression > progCPRL {
			return nil, fmt.Errorf("invalid progression order %d in POC", p.progression)
		}
		pocs = append(pocs, p)
	}
	return pocs, nil
}

// parseHeaderSegment parses a marker segment that may appear in the main
// header or a tile-part header, storing its contents in h. Unrecognized
// marker segments are ignored.
func parseHeaderSegment(marker int, seg *reader, h *header, numComps int) error {
	switch marker {
	case markerCOD:
		c, err := parseCOD(seg)
		if err != nil {
			return fmt.Errorf("COD: %w", err)
		}
		h.cod = c
	case markerCOC:
		c, cc, err := parseCOC(seg, numComps)
		if err != nil {
			return fmt.Errorf("COC: %w", err)
		}
		if h.coc == nil {
			h.coc = make(map[int]*compCoding)
		}
		h.coc[c] = cc
	case markerQCD:
		q, err := parseQuant(seg)
		if err != nil {
			return fmt.Errorf("QCD: %w", err)
		}
		h.qcd = q
	case markerQCC:
		c, q, err := parseQCC(seg, numComps)
		if err != nil {
			return fmt.Errorf("QCC: %w", err)
		}
		if h.qcc == nil {
			h.qcc = make(map[int]*quant)
		}
		h.qcc[c] = q
	case markerRGN:
		c, shift, err := parseRGN(seg, numComps)
		if err != nil {
			return fmt.Errorf("RGN: %w", err)
		}
		if h.rgn == nil {
			h.rgn = make(map[int]int)
		}
		h.rgn[c] = shift
	case markerPOC:
		p, err := parsePOC(seg, numComps)
		if err != nil {
			return fmt.Errorf("POC: %w", err)
		}
		h.pocs = append(h.pocs, p...)
	case markerPPM, markerPPT:
		return fmt.Errorf("packed packet headers (PPM/PPT) are not supported")
	case markerCAP:
		return fmt.Errorf("HTJ2K (Part 15) codestreams are not supported")
	}
	return nil
}
