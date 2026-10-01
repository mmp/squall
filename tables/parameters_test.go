package tables

import "testing"

func TestLookupParameter(t *testing.T) {
	for _, tc := range []struct {
		d, c, n   int
		shortName string
		unit      string
	}{
		{0, 0, 0, "TMP", "K"},
		{0, 0, 6, "DPT", "K"},
		{0, 2, 2, "UGRD", "m/s"},
		{0, 2, 3, "VGRD", "m/s"},
		{0, 3, 5, "HGT", "gpm"},
		{0, 3, 9, "GPA", "gpm"},
		{0, 6, 0, "CICE", "kg/m^2"},
		{2, 4, 2, "HINDEX", "Numeric"},
		{10, 0, 7, "SWDIR", "deg"},
		// NCEP local parameters.
		{0, 7, 199, "MXUPHL", "m^2/s^2"},
		{0, 16, 195, "REFD", "dB"},
		{0, 3, 196, "HPBL", "m"},
	} {
		p, ok := LookupParameter(tc.d, tc.c, tc.n)
		if !ok || p.ShortName != tc.shortName || p.Unit != tc.unit {
			t.Errorf("%d.%d.%d: got %+v (found %v), want %s [%s]", tc.d, tc.c, tc.n, p, ok, tc.shortName, tc.unit)
		}
	}

	if p, ok := LookupParameter(0, 10, 0); ok {
		t.Errorf("0.10.0: got %+v, want not found", p)
	}
	if _, ok := LookupParameter(0, 0, 300); ok {
		t.Error("out of range number should not be found")
	}
}
