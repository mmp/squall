package tables

import (
	"fmt"
	"strconv"
	"strings"
)

// Originating centers with local level types (Code Table C-1).
const (
	centerNCEP = 7
	centerKMA  = 40
)

// missingSurfaceValue is the value wgrib2 uses for a missing surface value.
const missingSurfaceValue = 9.999e20

// FormatLevel describes the level or layer given by a pair of fixed surfaces
// (Code Table 4.5) the way wgrib2 does, e.g., "500 mb", "2 m above ground",
// or "0-6000 m above ground". The values are those of the surfaces with
// their scale factors applied; missing indicates a missing value. center
// is the originating center, which determines the meaning of the local
// level types 192-254.
func FormatLevel(type1 uint8, value1 float64, missing1 bool,
	type2 uint8, value2 float64, missing2 bool, center int) string {
	// wgrib2 works with the values as floats.
	v1, v2 := float32(value1), float32(value2)
	if missing1 {
		v1 = missingSurfaceValue
	}
	if missing2 {
		v2 = missingSurfaceValue
	}

	// Layers between two surfaces of the same type.
	layers := map[uint8]string{
		100: "%g-%g mb",
		102: "%g-%g m above mean sea level",
		103: "%g-%g m above ground",
		104: "%g-%g sigma layer",
		105: "%g-%g hybrid layer",
		106: "%g-%g m below ground",
		107: "%g-%g K isentropic layer",
		108: "%g-%g mb above ground",
		111: "%g-%g Eta layer",
		115: "%g-%g sigma height layer",
		118: "%g-%g hybrid height layer",
		119: "%g-%g hybrid pressure layer",
		150: "%g-%g generalized vertical height coordinate",
		160: "%g-%g m below sea level",
		161: "%g-%g m ocean layer",
	}
	if format, ok := layers[type1]; ok && type1 == type2 {
		if type1 == 100 || type1 == 108 { // Pa to mb
			v1, v2 = v1/100, v2/100
		}
		return printfG(format, v1, v2)
	}

	switch {
	case type1 == 1 && type2 == 8:
		return "atmos col"
	case type1 == 9 && type2 == 1:
		return "ocean column"
	case center == centerNCEP && type1 == 235 && type2 == 235:
		return printfG("%g-%gC ocean isotherm layer", v1/10, v2/10)
	case center == centerNCEP && type1 == 236 && type2 == 236:
		return printfG("%g-%g m ocean layer", v1*10, v2*10)
	case type1 == 255 && type2 == 255:
		return "no_level"
	}

	s := formatSingleLevel(type1, v1, missing1, center)
	if type2 != 255 {
		s += " - " + formatSingleLevel(type2, v2, missing2, center)
	}
	return s
}

// formatSingleLevel describes a single fixed surface, as wgrib2's level1().
func formatSingleLevel(typ uint8, v float32, missing bool, center int) string {
	switch {
	case typ < 192:
		if typ == 100 || typ == 108 { // Pa to mb
			v = float32(float64(v) * 0.01)
		}
		return printfG(wgrib2Levels[typ], v)
	case typ == 255:
		return ""
	case center == centerNCEP || center == centerKMA:
		if typ == 235 { // 0.1C to C
			v = float32(float64(v) * 0.1)
		}
		if center == centerNCEP {
			return printfG(wgrib2NCEPLevels[typ-192], v)
		}
		return printfG(wgrib2KMALevels[typ-192], v)
	case missing:
		return fmt.Sprintf("local level type %d", typ)
	default:
		return printfG(fmt.Sprintf("local level type %d %%g", typ), v)
	}
}

// printfG formats values into a format string whose only verbs are %g and
// %%, matching C's printf. (Go's %g differs: it uses the shortest
// representation rather than 6 significant digits.)
func printfG(format string, values ...float32) string {
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 == len(format) {
			b.WriteByte(format[i])
			continue
		}
		i++
		switch format[i] {
		case 'g':
			if len(values) > 0 {
				b.WriteString(strconv.FormatFloat(float64(values[0]), 'g', 6, 64))
				values = values[1:]
			}
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(format[i])
		}
	}
	return b.String()
}
