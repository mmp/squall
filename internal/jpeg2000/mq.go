package jpeg2000

// The MQ arithmetic decoder (ITU-T T.800 Annex C) and the raw decoder used
// for bypassed coding passes (D.6).

// qeEntry is a row of the probability estimation state table (Table C.2).
type qeEntry struct {
	qe         uint32
	nmps, nlps uint8
	switchMPS  bool
}

var qeTable = [47]qeEntry{
	{0x5601, 1, 1, true}, {0x3401, 2, 6, false}, {0x1801, 3, 9, false},
	{0x0AC1, 4, 12, false}, {0x0521, 5, 29, false}, {0x0221, 38, 33, false},
	{0x5601, 7, 6, true}, {0x5401, 8, 14, false}, {0x4801, 9, 14, false},
	{0x3801, 10, 14, false}, {0x3001, 11, 17, false}, {0x2401, 12, 18, false},
	{0x1C01, 13, 20, false}, {0x1601, 29, 21, false}, {0x5601, 15, 14, true},
	{0x5401, 16, 14, false}, {0x5101, 17, 15, false}, {0x4801, 18, 16, false},
	{0x3801, 19, 17, false}, {0x3401, 20, 18, false}, {0x3001, 21, 19, false},
	{0x2801, 22, 19, false}, {0x2401, 23, 20, false}, {0x2201, 24, 21, false},
	{0x1C01, 25, 22, false}, {0x1801, 26, 23, false}, {0x1601, 27, 24, false},
	{0x1401, 28, 25, false}, {0x1201, 29, 26, false}, {0x1101, 30, 27, false},
	{0x0AC1, 31, 28, false}, {0x09C1, 32, 29, false}, {0x08A1, 33, 30, false},
	{0x0521, 34, 31, false}, {0x0441, 35, 32, false}, {0x02A1, 36, 33, false},
	{0x0221, 37, 34, false}, {0x0141, 38, 35, false}, {0x0111, 39, 36, false},
	{0x0085, 40, 37, false}, {0x0049, 41, 38, false}, {0x0025, 42, 39, false},
	{0x0015, 43, 40, false}, {0x0009, 44, 41, false}, {0x0005, 45, 42, false},
	{0x0001, 45, 43, false}, {0x5601, 46, 46, false},
}

// Coding contexts used by the tier-1 decoder (Annex D).
const (
	ctxZC       = 0  // zero coding, 9 contexts
	ctxSC       = 9  // sign coding, 5 contexts
	ctxMR       = 14 // magnitude refinement, 3 contexts
	ctxRL       = 17 // run-length
	ctxUniform  = 18 // uniform
	numContexts = 19
)

type mqDecoder struct {
	data []byte
	bp   int    // index of the most recently read byte
	a    uint32 // interval size
	c    uint32 // code register: C_high in bits 16-31, C_low in bits 0-15
	ct   int    // bits available before the next byte must be read

	state [numContexts]uint8 // index into qeTable
	mps   [numContexts]uint8 // more probable symbol
}

// resetContexts sets the context states to their initial values (Table D.7).
func (m *mqDecoder) resetContexts() {
	m.state = [numContexts]uint8{}
	m.mps = [numContexts]uint8{}
	m.state[ctxUniform] = 46
	m.state[ctxRL] = 3
	m.state[ctxZC] = 4
}

// byteAt returns the i-th byte of the segment; past the end it returns 0xFF,
// which the decoder treats as a marker and so feeds 1 bits from then on.
func (m *mqDecoder) byteAt(i int) uint32 {
	if i < len(m.data) {
		return uint32(m.data[i])
	}
	return 0xFF
}

// init starts decoding a new terminated segment (INITDEC, C.3.5). Context
// states are preserved.
func (m *mqDecoder) init(data []byte) {
	m.data = data
	m.bp = 0
	m.c = m.byteAt(0) << 16
	m.byteIn()
	m.c <<= 7
	m.ct -= 7
	m.a = 0x8000
}

// byteIn reads the next byte into the code register (BYTEIN, C.3.4),
// handling the bit stuffing that follows a 0xFF byte.
func (m *mqDecoder) byteIn() {
	if m.byteAt(m.bp) == 0xFF {
		if m.byteAt(m.bp+1) > 0x8F {
			m.c += 0xFF00
			m.ct = 8
		} else {
			m.bp++
			m.c += m.byteAt(m.bp) << 9
			m.ct = 7
		}
	} else {
		m.bp++
		m.c += m.byteAt(m.bp) << 8
		m.ct = 8
	}
}

// decode returns the next decision decoded with context cx (DECODE, C.3.2).
func (m *mqDecoder) decode(cx int) int {
	e := &qeTable[m.state[cx]]
	mps := int(m.mps[cx])
	d := mps
	m.a -= e.qe
	if m.c>>16 < e.qe {
		// The lower subinterval: the LPS, unless a conditional exchange
		// applies (LPS_EXCHANGE).
		if m.a < e.qe {
			m.state[cx] = e.nmps
		} else {
			d = 1 - mps
			if e.switchMPS {
				m.mps[cx] = uint8(d)
			}
			m.state[cx] = e.nlps
		}
		m.a = e.qe
	} else {
		m.c -= e.qe << 16
		if m.a&0x8000 != 0 {
			return d
		}
		// MPS_EXCHANGE.
		if m.a < e.qe {
			d = 1 - mps
			if e.switchMPS {
				m.mps[cx] = uint8(d)
			}
			m.state[cx] = e.nlps
		} else {
			m.state[cx] = e.nmps
		}
	}
	// RENORMD.
	for {
		if m.ct == 0 {
			m.byteIn()
		}
		m.a <<= 1
		m.c <<= 1
		m.ct--
		if m.a&0x8000 != 0 {
			break
		}
	}
	return d
}

// rawDecoder reads the uncoded bits of bypassed coding passes.
type rawDecoder struct {
	data []byte
	pos  int
	c    uint32
	ct   int
}

func (r *rawDecoder) init(data []byte) {
	*r = rawDecoder{data: data}
}

func (r *rawDecoder) bit() int {
	if r.ct == 0 {
		next := uint32(0xFF)
		if r.pos < len(r.data) {
			next = uint32(r.data[r.pos])
		}
		if r.c == 0xFF {
			// A 0 bit is stuffed after each 0xFF byte, unless the 0xFF is
			// followed by a marker, in which case 1s are fed in.
			if next > 0x8F {
				r.ct = 8
			} else {
				r.c = next
				r.pos++
				r.ct = 7
			}
		} else {
			r.c = next
			r.pos++
			r.ct = 8
		}
	}
	r.ct--
	return int(r.c>>r.ct) & 1
}
