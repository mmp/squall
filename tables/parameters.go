package tables

//go:generate go run gen_parameters.go

// Parameter describes a GRIB2 parameter (Code Table 4.2).
type Parameter struct {
	ShortName   string // Abbreviation, as used by wgrib2 (e.g., "TMP")
	Description string // Full name (e.g., "Temperature")
	Unit        string // Units (e.g., "K")
}

// LookupParameter returns the parameter with the given discipline,
// category, and number, using wgrib2's parameter table.
//
// Parameters with a discipline, category, or number in the range reserved
// for local use (192-254) are looked up in NCEP's local table, as these are
// only meaningful for data from the center that defined them.
func LookupParameter(discipline, category, number int) (Parameter, bool) {
	if discipline < 0 || discipline > 255 || category < 0 || category > 255 || number < 0 || number > 255 {
		return Parameter{}, false
	}
	key := [3]uint8{uint8(discipline), uint8(category), uint8(number)}
	local := func(v int) bool { return v >= 192 && v <= 254 }
	if local(discipline) || local(category) || local(number) {
		p, ok := ncepLocalParameters[key]
		return p, ok
	}
	p, ok := wmoParameters[key]
	return p, ok
}
