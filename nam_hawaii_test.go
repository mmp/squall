package squall_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"os"
	"testing"

	"github.com/mmp/squall"
)

// TestNAMHawaiiJPEG2000 decodes fields from the NAM Hawaii nest, which uses
// JPEG 2000 packing (template 5.40) on a Mercator grid. The test file holds
// a selection of its messages: vice's fields at 500 mb, a constant field
// (0 bits per value), and fields with bitmaps. Two of the messages hold both
// the U and V wind components.
//
// The expected values are the SHA-256 of wgrib2's output for each field
// (-order we:sn -no_header -bin, with g2c's JasPer-based JPEG 2000 decoder):
// little-endian float32s, with 9.999e20 for missing values.
func TestNAMHawaiiJPEG2000(t *testing.T) {
	f, err := os.Open("testdata/nam-hawaii-subset.grib2")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	fields, err := squall.Read(f)
	if err != nil {
		t.Fatal(err)
	}

	var (
		cice   = squall.ParameterID{Discipline: 0, Category: 6, Number: 0}
		hgt    = squall.ParameterID{Discipline: 0, Category: 3, Number: 5}
		tmp    = squall.ParameterID{Discipline: 0, Category: 0, Number: 0}
		dpt    = squall.ParameterID{Discipline: 0, Category: 0, Number: 6}
		ugrd   = squall.ParameterID{Discipline: 0, Category: 2, Number: 2}
		vgrd   = squall.ParameterID{Discipline: 0, Category: 2, Number: 3}
		hindex = squall.ParameterID{Discipline: 2, Category: 4, Number: 2}
	)
	expected := []struct {
		param   squall.ParameterID
		level   string
		missing int
		sha256  string
	}{
		{cice, "Hybrid 1", 0, "897095e7f80f881b862cbf454bd32246f3fb0f168e021c785334d29e2e84d74a"},
		{hgt, "500 mb", 0, "646b7c6e6a662dc71c4e3a732ce4f0878c7ebf13fd7ff89647c903901e86f2b6"},
		{tmp, "500 mb", 0, "23253954921a04ab1fc794b6f2eb6aaec8e322c18873c4f54d3ccfb406c8b8d9"},
		{dpt, "500 mb", 0, "650f87b290098c7110d7b2fe886b71cfc9d827c031ad73f904dadc4a731cd822"},
		{ugrd, "500 mb", 0, "ee6e7768ccab077797bdc429e52eedfe1298832e462c62fce8e1a2a4455c5516"},
		{vgrd, "500 mb", 0, "79a725fae06c8be58f81083406ced261039e1954f060d9318db3cf87ecbf3470"},
		{hindex, "surface", 69548, "c8f64d2ffffa48d8d4e0c4fc1978199bf0e4d9b8c6a65f3d2c9c86bdd37a1043"},
		{tmp, "Altitude MSL 305", 1795, "1a4c5899944be146684d885dbaaae2024c4a690d3d1748af75cecbcac996307c"},
		{ugrd, "Altitude MSL 305", 1795, "f408273445b8812b63ae25aa3fe4950df85a366efb0ddf48cc6814edfd143798"},
		{vgrd, "Altitude MSL 305", 1795, "66b5d879e8e202bf3a3d0d194ef7a4c29a4cdb0aeb95c5656aec1205434b35ec"},
	}
	if len(fields) != len(expected) {
		t.Errorf("got %d fields, want %d", len(fields), len(expected))
	}

	for _, want := range expected {
		var field *squall.GRIB2
		for _, f := range fields {
			if f.Parameter == want.param && f.Level == want.level {
				field = f
				break
			}
		}
		if field == nil {
			t.Errorf("%s %s: not found", want.param, want.level)
			continue
		}

		if field.GridNi != 321 || field.GridNj != 225 || len(field.Data) != 321*225 {
			t.Errorf("%s %s: got %dx%d grid with %d values, want 321x225", want.param, want.level,
				field.GridNi, field.GridNj, len(field.Data))
			continue
		}
		if n := len(field.Data) - field.CountValid(); n != want.missing {
			t.Errorf("%s %s: %d missing values, want %d", want.param, want.level, n, want.missing)
		}

		var b []byte
		for _, v := range field.Data {
			b = binary.LittleEndian.AppendUint32(b, math.Float32bits(v))
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != want.sha256 {
			t.Errorf("%s %s: values differ from wgrib2's (SHA-256 %s)", want.param, want.level, got)
		}
	}
}
