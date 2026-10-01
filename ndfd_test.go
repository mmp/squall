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

// TestNDFDMissingValues decodes NDFD fields that use complex packing with
// spatial differencing and missing value management (template 5.3 with
// primary missing values) rather than a bitmap to mark points outside
// CONUS. The expected values are the SHA-256 of wgrib2's output in the
// grid's scan order (-order raw -no_header -bin): little-endian float32s,
// with 9.999e20 for missing values.
func TestNDFDMissingValues(t *testing.T) {
	f, err := os.Open("testgribs/ndfd_conus_temp.grib2")
	if err != nil {
		t.Skip("testgribs/ndfd_conus_temp.grib2 not found")
	}
	defer f.Close()

	fields, err := squall.Read(f)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"bb463359af6ceb56df0400389b1ba92bee00f8befb9705f00c3078f81175637f",
		"fba69a9f8570aae424bdba66221dec34a02f3608810c0356dde6c610cc8850f5",
		"fbd35a73735476da9cd1fb1377e87e7fea181f18c4c575c538a7d50969a6f60a",
	}
	if len(fields) != len(want) {
		t.Fatalf("got %d fields, want %d", len(fields), len(want))
	}
	for i, field := range fields {
		if n := len(field.Data) - field.CountValid(); n != 1479351 {
			t.Errorf("field %d: %d missing values, want 1479351", i, n)
		}
		var b []byte
		for _, v := range field.Data {
			b = binary.LittleEndian.AppendUint32(b, math.Float32bits(v))
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != want[i] {
			t.Errorf("field %d: values differ from wgrib2's (SHA-256 %s)", i, got)
		}
	}
}
