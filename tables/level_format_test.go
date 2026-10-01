package tables

import "testing"

func TestFormatLevel(t *testing.T) {
	type surface struct {
		typ     uint8
		value   float64
		missing bool
	}
	none := surface{255, 0, true}
	for _, tc := range []struct {
		s1, s2 surface
		center int
		want   string
	}{
		// Isobaric levels, as vice's atmos ingest expects them.
		{surface{100, 50000, false}, none, 7, "500 mb"},
		{surface{100, 101320, false}, none, 7, "1013.2 mb"},
		{surface{1, 0, false}, none, 7, "surface"},
		{surface{103, 2, false}, none, 7, "2 m above ground"},
		{surface{102, 305, false}, none, 7, "305 m above mean sea level"},
		{surface{105, 1, false}, none, 7, "1 hybrid level"},
		{surface{4, 0, true}, none, 7, "0C isotherm"},
		{surface{241, 3, false}, none, 7, "3 in sequence"},

		// Layers.
		{surface{100, 50000, false}, surface{100, 100000, false}, 7, "500-1000 mb"},
		{surface{108, 18000, false}, surface{108, 0, false}, 7, "180-0 mb above ground"},
		{surface{103, 3000, false}, surface{103, 0, false}, 7, "3000-0 m above ground"},
		{surface{106, 0, false}, surface{106, 0.1, false}, 7, "0-0.1 m below ground"},
		{surface{1, 0, false}, surface{8, 0, false}, 7, "atmos col"},
		{none, none, 7, "no_level"},
		{surface{1, 0, false}, surface{105, 2, false}, 7, "surface - 2 hybrid level"},

		// Local level types depend on the center.
		{surface{204, 0, false}, none, 7, "highest tropospheric freezing level"},
		{surface{214, 0, false}, none, 7, "low cloud layer"},
		{surface{204, 0, false}, none, 54, "local level type 204 0"},
		{surface{204, 0, true}, none, 54, "local level type 204"},
	} {
		got := FormatLevel(tc.s1.typ, tc.s1.value, tc.s1.missing, tc.s2.typ, tc.s2.value, tc.s2.missing, tc.center)
		if got != tc.want {
			t.Errorf("%+v, %+v, center %d: got %q, want %q", tc.s1, tc.s2, tc.center, got, tc.want)
		}
	}
}

func TestPrintfG(t *testing.T) {
	for _, tc := range []struct {
		format string
		v      float32
		want   string
	}{
		{"%g", 263.379162, "263.379"}, // 6 significant digits, as in C
		{"%g", 1e6, "1e+06"},
		{"%g", 0.0001, "0.0001"},
		{"lowest level %g%% integrated cloud cover", 50, "lowest level 50% integrated cloud cover"},
	} {
		if got := printfG(tc.format, tc.v); got != tc.want {
			t.Errorf("printfG(%q, %v) = %q, want %q", tc.format, tc.v, got, tc.want)
		}
	}
}
