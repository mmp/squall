package jpeg2000

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type compInfo struct {
	w, h, prec int
	signed     bool
}

// The testdata codestreams were produced by OpenJPEG's opj_compress from
// synthetic images, covering a wide range of coding options (the name of
// each indicates what it exercises), plus two from a NAM Hawaii GRIB2 file
// (encoded by JasPer via NCEP's g2c library); one of the latter is a
// 2677x1 image from a field with a bitmap. The expected values are the
// SHA-256 of the samples decoded by OpenJPEG, as little-endian int32s with
// the components in order. Irreversible (9/7) codestreams instead have the
// OpenJPEG decoding stored in testdata/<name>.ref.gz; floating-point
// differences make it reasonable to differ from it by 1.
var conformanceTests = []struct {
	name   string
	comps  []compInfo
	sha256 string
}{
	{"cb128x32", []compInfo{{150, 40, 12, false}}, "0f3cfa9dfdde389a4b0e140c8eb9e1240e382a4869ac3e22f19670d13a3007d2"},
	{"cb16x64", []compInfo{{70, 40, 12, false}}, "5f7019e809f032f1a8ae7af0d1be25f0126f903783c7b7b1de0f407b7c5749a2"},
	{"cb4x4", []compInfo{{37, 33, 12, false}}, "b12f4526520d42213bae577e325efbb9e7a6ea902b210d7b581eb172da20ca13"},
	{"col", []compInfo{{1, 61, 12, false}}, "a59176a8958022b77dd04ef347cfbbb22f098288f88d12347babc1a07aca37bf"},
	{"const", []compInfo{{33, 21, 9, false}}, "99aa566eb04c6c36f4d0829c7880b2a2cc89e63ec0b3102cab051479502b7f46"},
	{"cprl", []compInfo{{45, 37, 12, false}}, "a875de40a9710a77490c86938a218625ed4e68603ecaee1d99ec8a278070b7ae"},
	{"default", []compInfo{{45, 37, 12, false}}, "c604af4d4877d49222acb46a5b21477e46aad15a9724bc1ab3373bbd5565263f"},
	{"imgoff_odd", []compInfo{{60, 54, 12, false}}, "49341af786368b435f5fea6c23028ce702c359472a0f52915bc8a5dd7596840f"},
	{"irrev", []compInfo{{45, 37, 12, false}}, ""},
	{"irrev_layers", []compInfo{{45, 37, 16, false}}, ""},
	{"irrev_off_tiles", []compInfo{{65, 50, 12, false}}, ""},
	{"irrev_signed", []compInfo{{45, 37, 11, true}}, ""},
	{"m_all_layers", []compInfo{{45, 37, 16, false}}, "998c0f998edf5eea895c1131ea08bfc3009e840fa80274ef262c31ecd357cb10"},
	{"m_bypass", []compInfo{{45, 37, 16, false}}, "1104f4fb68788d520509044d3cba18d55e622217949b6fc37ed0942ffe9d3bba"},
	{"m_bypass_layers", []compInfo{{45, 37, 16, false}}, "f89b5fff5c15a58b93e0546985561c3764e246ba0b12b7cdb948c6f38d11e38e"},
	{"m_reset", []compInfo{{45, 37, 12, false}}, "06a81e122990cd9eb95d1d369fb903e2be7cebad43fa4df16367c37c95d92805"},
	{"m_segsym", []compInfo{{45, 37, 12, false}}, "be286192369c6972213d573c51bf34ef1cb66393c29c5ae7bc8640dec0982130"},
	{"m_termall", []compInfo{{45, 37, 12, false}}, "de602da026e81603e20a0ea42a5b2ddddfe5a4fc5b195f4a3560f7dc841414ba"},
	{"m_vsc", []compInfo{{45, 37, 12, false}}, "33a7693f83c546eee2b07a71b2469ecca7ee1f360ae5f264d656a580a1c4278c"},
	{"n1", []compInfo{{30, 22, 12, false}}, "0de80f4635e646cb872ad3fdeba34e50860df77afa2dfb40995f94594458bc1c"},
	{"n8", []compInfo{{140, 130, 12, false}}, "26f5202d1b9deed0596ec5c3f60b625091204702bcbc40468dcd7094ac9e710c"},
	{"nam_hawaii_bitmap", []compInfo{{2677, 1, 9, false}}, "35e19402d07542368ed8319c29ee9cefc69c2c50c57a6fbd4abb8b3a17679ce1"},
	{"nam_hawaii_msg3", []compInfo{{321, 225, 10, false}}, "88c806f8003f4580ee730131797d3a8fcacc486bd2a17f04b29c816d6a5c678f"},
	{"pcrl", []compInfo{{45, 37, 12, false}}, "5bb09cd3e3eadd6733da34603f56754ed1540cced636afb57fda8af4fc9fe3a8"},
	{"poc", []compInfo{{45, 37, 12, false}}, "97429359678fdfb5583620d94299065fbde60321eae1f001c2e0af8805979d7b"},
	{"prec1", []compInfo{{40, 33, 1, false}}, "569cfdcf139f915b2f1dabdfbc86320ec1c7d54e28c12a93132dae8ef4f91c83"},
	{"prec16", []compInfo{{41, 35, 16, false}}, "9dd3feea97124d1dba26de1527d6e8da2b292b8d7d3ceb0db846e829ae9af8c4"},
	{"prec24", []compInfo{{35, 33, 24, false}}, "6c33afaf22ec59e2633adb17a154f11b17e0b17e8f38560fd6dcb2a61c40a759"},
	{"prec8noise", []compInfo{{36, 34, 8, false}}, "27f6711814d2efcbe36876410888e090304f38dd42a396791e53ff574f3f24b5"},
	{"prec_cprl", []compInfo{{75, 55, 12, false}}, "d5534f84e43881b5bb350ea5d0f22373afd9c06a5ddf1cd526e36ae1e391fc86"},
	{"prec_lrcp", []compInfo{{75, 55, 12, false}}, "2e414e58db9d1aad9f7fb6df4763ed68c62a36fd9be5383418e7301f63ab2f5b"},
	{"prec_pcrl", []compInfo{{75, 55, 12, false}}, "62b68caf1ee77a87ced8ecfd0c9336a33499cce822e5d6a861d9edb0160dbb6f"},
	{"prec_rpcl", []compInfo{{75, 55, 12, false}}, "21a5b42488ddef70b07b61128531c9074685b08a7a105332ee7c3e5f73ec04e8"},
	{"rgb_cprl_tiles", []compInfo{{41, 35, 8, false}, {41, 35, 8, false}, {41, 35, 8, false}}, "e30aca2b7ddfb3ef7d501f5a82a7f2dbc3db2578d8972edc5a5761f62553fc2c"},
	{"rgb_ict", []compInfo{{41, 35, 8, false}, {41, 35, 8, false}, {41, 35, 8, false}}, ""},
	{"rgb_poc", []compInfo{{41, 35, 8, false}, {41, 35, 8, false}, {41, 35, 8, false}}, "83b62adef50448738f733ab4000ad42da0b4420016cfec64974eb52c2fcecb1e"},
	{"rgb_rct", []compInfo{{41, 35, 8, false}, {41, 35, 8, false}, {41, 35, 8, false}}, "f9c793d9c05d375f85b30dcb20d145ed5ea33472452bc9056f46c4a60d140cf1"},
	{"rlcp", []compInfo{{45, 37, 12, false}}, "aa4d3aa8f6b8dfc7836dab620316b5b224f76a00e8f843f417aa353958a73542"},
	{"roi", []compInfo{{45, 37, 12, false}}, "08d871b06889aade21572822496fce0e1bcf62ce74354cf20704b4ae89f33de6"},
	{"row", []compInfo{{77, 1, 12, false}}, "487217a840817304aefc38ce362d8174698c02ba2387ea556f67bf098e35f313"},
	{"rpcl", []compInfo{{45, 37, 12, false}}, "d76305a94a67e40a3e8e25128cf8b805d117138b0241e00b476e46e37ab384d4"},
	{"signed12", []compInfo{{38, 33, 11, true}}, "50057d2972d05ad1f72ed113d3d0c0be5a22a08e0faa6d8ae217f456621f4e32"},
	{"signed16noise", []compInfo{{33, 38, 15, true}}, "6293e2e560cd02eea6d4314298ffcaccccfd1b9617edc88fb26711eb7ab2281f"},
	{"sop_eph", []compInfo{{45, 37, 12, false}}, "3cfbeb1d572a80234bcde7a6fb8e1760bb438695054fa54db2cf01a1acf82492"},
	{"sparse", []compInfo{{64, 48, 14, false}}, "1e5d74ade748b63d26e522c12cb381321f35d47cfb445ebf5752530685e403da"},
	{"subsample_off", []compInfo{{59, 49, 12, false}}, "16ba487f90099e96ff952c24de176c4f51e8044b1ff5a6657695223b5ba8ef67"},
	{"thin", []compInfo{{3, 50, 9, false}}, "39e4b51d0f581bb6a5b507aab83e7f30cdf1c37947f855524326d8035849e615"},
	{"tiles_off", []compInfo{{75, 44, 12, false}}, "d4033fed3db38fd332071e686aaa15877306861f63cdbe267ea6d40136e5bf20"},
	{"tiny1x1", []compInfo{{1, 1, 6, false}}, "d2d27d69fc0a2c6cc0aabec462ce665aa8a92766844f081b672588acdf8a2c71"},
	{"tp_layer", []compInfo{{45, 37, 12, false}}, "d4597da8ff530568c957ecefab2bc0e4470510eb465482f69a820760b274eb82"},
	{"tp_res", []compInfo{{70, 60, 12, false}}, "b48ca55792db78fb8d294e04578b1d5f74d286d077d8653663ae692788ce60a0"},
}

func TestDecodeConformance(t *testing.T) {
	for _, tc := range conformanceTests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", tc.name+".j2k"))
			if err != nil {
				t.Fatal(err)
			}
			img, err := Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(img.Components) != len(tc.comps) {
				t.Fatalf("got %d components, want %d", len(img.Components), len(tc.comps))
			}
			cfg, err := DecodeConfig(data)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Width != img.Width || cfg.Height != img.Height || len(cfg.Components) != len(img.Components) {
				t.Errorf("DecodeConfig %dx%d/%d components; Decode %dx%d/%d", cfg.Width, cfg.Height,
					len(cfg.Components), img.Width, img.Height, len(img.Components))
			}
			var samples []byte
			for i, c := range img.Components {
				want := tc.comps[i]
				got := compInfo{c.Width, c.Height, c.Precision, c.Signed}
				if got != want {
					t.Fatalf("component %d: got %+v, want %+v", i, got, want)
				}
				if len(c.Data) != c.Width*c.Height {
					t.Fatalf("component %d: %d samples for %dx%d", i, len(c.Data), c.Width, c.Height)
				}
				for _, v := range c.Data {
					samples = binary.LittleEndian.AppendUint32(samples, uint32(v))
				}
			}

			if tc.sha256 != "" {
				sum := sha256.Sum256(samples)
				if got := hex.EncodeToString(sum[:]); got != tc.sha256 {
					t.Errorf("decoded samples have SHA-256 %s, want %s", got, tc.sha256)
				}
				return
			}

			ref := readGzip(t, filepath.Join("testdata", tc.name+".ref.gz"))
			if len(ref) != len(samples) {
				t.Fatalf("reference has %d bytes, decoded %d", len(ref), len(samples))
			}
			for i := 0; i < len(ref); i += 4 {
				got := int32(binary.LittleEndian.Uint32(samples[i:]))
				want := int32(binary.LittleEndian.Uint32(ref[i:]))
				if got-want > 1 || want-got > 1 {
					t.Fatalf("sample %d: got %d, want %d", i/4, got, want)
				}
			}
		})
	}
}

func readGzip(t *testing.T, path string) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDecodeJP2(t *testing.T) {
	cs, err := os.ReadFile(filepath.Join("testdata", "default.j2k"))
	if err != nil {
		t.Fatal(err)
	}
	box := func(typ string, payload []byte) []byte {
		b := binary.BigEndian.AppendUint32(nil, uint32(8+len(payload)))
		return append(append(b, typ...), payload...)
	}
	var jp2 []byte
	jp2 = append(jp2, box("jP  ", []byte{0x0D, 0x0A, 0x87, 0x0A})...)
	jp2 = append(jp2, box("ftyp", []byte("jp2 \x00\x00\x00\x00jp2 "))...)
	jp2 = append(jp2, box("jp2c", cs)...)

	want, err := Decode(cs)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(jp2)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range want.Components[0].Data {
		if got.Components[0].Data[i] != v {
			t.Fatalf("sample %d: got %d, want %d", i, got.Components[0].Data[i], v)
		}
	}
}

// Corrupt or truncated input must produce an error (or garbage), never a
// panic or a hang.
func TestDecodeMalformed(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "nam_hawaii_msg3.j2k"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(nil); err == nil {
		t.Error("expected error for empty input")
	}
	for _, n := range []int{2, 10, 50, 116, 130, 500, len(data) / 2, len(data) - 3} {
		if _, err := Decode(data[:n]); err == nil {
			t.Errorf("expected error for codestream truncated to %d bytes", n)
		}
	}
	for i := range 200 {
		b := append([]byte(nil), data...)
		b[(i*7919)%len(b)] ^= byte(1 << (i % 8))
		_, _ = Decode(b)
	}
}

func FuzzDecode(f *testing.F) {
	for _, tc := range conformanceTests {
		data, err := os.ReadFile(filepath.Join("testdata", tc.name+".j2k"))
		if err != nil {
			f.Fatal(err)
		}
		if len(data) < 8192 {
			f.Add(data)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Decode(data)
	})
}

func TestMirror(t *testing.T) {
	// Whole-sample symmetric extension of a length-4 signal: ...2 1 [0 1 2 3] 2 1 0...
	want := map[int]int{-3: 3, -2: 2, -1: 1, 0: 0, 3: 3, 4: 2, 5: 1, 6: 0, 7: 1, 9: 3}
	for i, w := range want {
		if got := mirror(i, 4); got != w {
			t.Errorf("mirror(%d, 4) = %d, want %d", i, got, w)
		}
	}
}

func TestCeilDiv(t *testing.T) {
	for _, tc := range []struct{ a, b, want int }{
		{0, 2, 0}, {1, 2, 1}, {2, 2, 1}, {3, 2, 2}, {-1, 2, 0}, {-2, 2, -1}, {-3, 2, -1},
	} {
		if got := ceilDiv(tc.a, tc.b); got != tc.want {
			t.Errorf("ceilDiv(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
		if tc.b == 2 {
			if got := ceilDivPow2(tc.a, 1); got != tc.want {
				t.Errorf("ceilDivPow2(%d, 1) = %d, want %d", tc.a, got, tc.want)
			}
		}
	}
}

// The reversible 5/3 transform is lossless, so a forward transform followed
// by inverse53 must reproduce the input for any alignment of the signal.
func TestInverse53RoundTrip(t *testing.T) {
	forward := func(x []int32, i0 int) []int32 {
		n := len(x)
		if n == 1 {
			if i0&1 != 0 {
				return []int32{2 * x[0]}
			}
			return []int32{x[0]}
		}
		ext := func(i int) int64 { return int64(x[mirror(i-i0, n)]) }
		y := make([]int64, n+4)
		get := func(i int) int64 { return y[i-i0+2] }
		for i := i0 - 2; i < i0+n+2; i++ {
			y[i-i0+2] = ext(i)
		}
		// F.4.8.1: high-pass first, then low-pass.
		for i := i0 - 1; i < i0+n+1; i++ {
			if i&1 != 0 {
				y[i-i0+2] = ext(i) - (ext(i-1)+ext(i+1))>>1
			}
		}
		for i := i0; i < i0+n; i++ {
			if i&1 == 0 {
				y[i-i0+2] = ext(i) + (get(i-1)+get(i+1)+2)>>2
			}
		}
		out := make([]int32, n)
		for i := range out {
			out[i] = int32(y[i+2])
		}
		return out
	}
	buf := make([]int64, 64)
	for n := 1; n < 20; n++ {
		for i0 := range 3 {
			x := make([]int32, n)
			for i := range x {
				x[i] = int32((i*37+i0*11)%101 - 50)
			}
			y := forward(x, i0)
			inverse53(y, i0, buf)
			for i := range x {
				if y[i] != x[i] {
					t.Fatalf("n=%d i0=%d: sample %d: got %d, want %d", n, i0, i, y[i], x[i])
				}
			}
		}
	}
}

func TestTagTree(t *testing.T) {
	values := []int32{1, 2, 0, 2, 3, 1, 4, 0, 5, 2, 2, 3} // a 4x3 grid

	// Encode as in B.10.2: for each leaf, raise the threshold until the
	// leaf's value is known, the same sequence of queries the packet header
	// decoder makes.
	enc := newTagTree(4, 3)
	for i, v := range values {
		for n := int32(i); n >= 0; n = enc.parent[n] {
			enc.value[n] = min(enc.value[n], v)
		}
	}
	known := make([]bool, len(enc.value))
	var bits []int
	encode := func(leaf int, threshold int32) {
		var stack []int32
		n := int32(leaf)
		for ; enc.parent[n] >= 0; n = enc.parent[n] {
			stack = append(stack, n)
		}
		low := int32(0)
		for {
			if low > enc.low[n] {
				enc.low[n] = low
			} else {
				low = enc.low[n]
			}
			for low < threshold {
				if low >= enc.value[n] {
					if !known[n] {
						bits = append(bits, 1)
						known[n] = true
					}
					break
				}
				bits = append(bits, 0)
				low++
			}
			enc.low[n] = low
			if len(stack) == 0 {
				break
			}
			n = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		}
	}
	for leaf, v := range values {
		for th := int32(1); th <= v+1; th++ {
			encode(leaf, th)
		}
	}

	var packed []byte
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := range 8 {
			b <<= 1
			if i+j < len(bits) {
				b |= byte(bits[i+j])
			}
		}
		packed = append(packed, b)
	}
	if len(packed) > 0 && packed[len(packed)-1] == 0xFF {
		t.Fatal("test data needs bit stuffing")
	}

	dec := newTagTree(4, 3)
	r := headerReader{data: packed}
	for leaf, v := range values {
		th := int32(0)
		for !dec.decode(&r, leaf, th+1) {
			th++
		}
		if th != v {
			t.Errorf("leaf %d: decoded %d, want %d", leaf, th, v)
		}
	}
}

func TestHeaderReaderBitStuffing(t *testing.T) {
	// After a 0xFF byte, the next byte contributes only 7 bits.
	r := headerReader{data: []byte{0xFF, 0x7F, 0x80}}
	if got := r.bits(8); got != 0xFF {
		t.Errorf("first byte: got %#x", got)
	}
	if got := r.bits(7); got != 0x7F {
		t.Errorf("stuffed byte: got %#x", got)
	}
	if got := r.bit(); got != 1 {
		t.Errorf("third byte: got %d", got)
	}
	// Aligning after a 0xFF consumes the following byte.
	r = headerReader{data: []byte{0xFF, 0x00, 0x42}}
	r.bits(8)
	r.align()
	if r.pos != 2 {
		t.Errorf("align after 0xFF: pos %d, want 2", r.pos)
	}
}

func BenchmarkDecode(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "nam_hawaii_msg3.j2k"))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(321 * 225 * 4)
	for range b.N {
		if _, err := Decode(data); err != nil {
			b.Fatal(err)
		}
	}
}
