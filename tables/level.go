package tables

// WMO Code Table 4.5: Fixed Surface Types and Units
//
// This table defines the types of fixed surfaces (levels) where data is defined.
// Each surface type has associated units for its value.

var levelEntries = []*Entry{
	// Surface and ground-based levels
	{1, "Surface", "Ground or water surface", ""},
	{2, "Cloud Base", "Cloud base level", ""},
	{3, "Cloud Top", "Cloud top level", ""},
	{4, "0°C Isotherm", "Level of 0°C isotherm", ""},
	{5, "Condensation", "Level of adiabatic condensation lifted from the surface", ""},
	{6, "Max Wind", "Maximum wind level", ""},
	{7, "Tropopause", "Tropopause", ""},
	{8, "Nominal Top", "Nominal top of the atmosphere", ""},
	{9, "Sea Bottom", "Sea bottom", ""},
	{10, "Atmosphere", "Entire atmosphere", ""},
	{11, "Cumulonimbus Base", "Cumulonimbus (CB) base", "m"},
	{12, "Cumulonimbus Top", "Cumulonimbus (CB) top", "m"},
	{13, "Lowest Level", "Lowest level where vertically integrated cloud cover exceeds the specified percentage", "%"},
	{14, "Freezing Rain", "Level of free convection (LFC)", ""},
	{15, "Radar Base", "Radar base reflectivity", ""},
	{20, "Isothermal", "Isothermal level", "K"},

	// Isobaric surfaces
	{100, "Isobaric", "Isobaric surface", "Pa"},
	{101, "MSL", "Mean sea level", ""},
	{102, "Altitude MSL", "Specific altitude above mean sea level", "m"},
	{103, "Height AGL", "Specified height level above ground", "m"},
	{104, "Sigma", "Sigma level", ""},
	{105, "Hybrid", "Hybrid level", ""},
	{106, "Depth BG", "Depth below land surface", "m"},
	{107, "Isentropic", "Isentropic (theta) level", "K"},
	{108, "Pressure Diff", "Level at specified pressure difference from ground to level", "Pa"},
	{109, "Potential Vorticity", "Potential vorticity surface", "K m²/(kg s)"},
	{110, "Reserved", "Reserved", ""},
	{111, "Eta", "Eta level", ""},
	{112, "Reserved", "Reserved", ""},
	{113, "Log-Hybrid", "Logarithmic hybrid level", ""},
	{114, "Snow Level", "Snow level", "Numeric"},
	{115, "Sigma Height", "Sigma height level", ""},
	{116, "Reserved", "Reserved", ""},
	{117, "Planetary Boundary Layer", "Planetary boundary layer", ""},
	{118, "Eta Coordinate", "Eta coordinate", ""},

	// Ocean levels
	{160, "Depth BelowSea", "Depth below sea level", "m"},
	{161, "Ocean Model", "Ocean model level (generic)", ""},
	{162, "Reserved", "Reserved", ""},
	{163, "Reserved", "Reserved", ""},
	{168, "Reserved", "Reserved", ""},
	{169, "Hybrid Height", "Hybrid height level", ""},

	// Depth ranges
	{170, "Ocean Isotherm", "Ocean isotherm level (1/10 °C)", ""},
	{171, "Ocean Layer", "Ocean layer between two depths", "m"},
	{172, "Ocean Bottom", "Ocean bottom", ""},
	{173, "Ocean Mixed Layer", "Ocean mixed layer bottom", ""},
	{174, "Ocean Layer Avg", "Ocean layer between two isobaths", "m"},
	{175, "Reserved", "Reserved", ""},
	{176, "Lake or River Bottom", "Lake or river bottom", ""},
	{177, "Entire Lake/River", "Entire lake or river as a single layer", ""},
	{178, "Water Surface", "Top surface of ice on sea, lake, or river", ""},
	{179, "Grid Tile", "Top surface of ice, as in mosaic", ""},
	{180, "Generalized Vertical Height", "Generalized vertical height coordinate", ""},

	// Satellite/radar
	{200, "Entire Atmosphere", "Entire atmosphere (considered as a single layer)", ""},
	{201, "Entire Ocean", "Entire ocean (considered as a single layer)", ""},
	{204, "Highest Tropospheric Freezing", "Highest tropospheric freezing level", ""},
	{206, "Grid Scale Cloud Bottom", "Grid scale cloud bottom level", ""},
	{207, "Grid Scale Cloud Top", "Grid scale cloud top level", ""},
	{209, "Boundary Layer Cloud Bottom", "Boundary layer cloud bottom level", ""},
	{210, "Boundary Layer Cloud Top", "Boundary layer cloud top level", ""},
	{211, "Boundary Layer Cloud Layer", "Boundary layer cloud layer", ""},
	{212, "Low Cloud Bottom", "Low cloud bottom level", ""},
	{213, "Low Cloud Top", "Low cloud top level", ""},
	{214, "Low Cloud Layer", "Low cloud layer", ""},
	{215, "Cloud Ceiling", "Cloud ceiling", ""},
	{220, "Planetary Boundary Layer", "Planetary boundary layer (single layer)", ""},
	{221, "Layer Between Hybrid", "Layer between two hybrid levels", "Numeric"},
	{222, "Middle Cloud Bottom", "Middle cloud bottom level", ""},
	{223, "Middle Cloud Top", "Middle cloud top level", ""},
	{224, "Middle Cloud Layer", "Middle cloud layer", ""},
	{232, "High Cloud Bottom", "High cloud bottom level", ""},
	{233, "High Cloud Top", "High cloud top level", ""},
	{234, "High Cloud Layer", "High cloud layer", ""},
	{235, "Ocean Isotherm", "Ocean isotherm level (1/10 °C)", ""},
	{236, "Layer Depth Below", "Layer between two depths below ocean surface", "m"},
	{237, "Bottom of Ocean Mixed Layer", "Bottom of ocean mixed layer", "m"},
	{238, "Bottom of Ocean Isothermal Layer", "Bottom of ocean isothermal layer", "m"},
	{239, "Layer Ocean Depth", "Layer ocean surface and 26°C ocean isothermal level", ""},
	{240, "Ocean Mixed Layer", "Ocean mixed layer", "m"},
	{241, "Ordered Sequence", "Ordered sequence of data", ""},
	{242, "Convective Cloud Bottom", "Convective cloud bottom level", ""},
	{243, "Convective Cloud Top", "Convective cloud top level", ""},
	{244, "Convective Cloud Layer", "Convective cloud layer", ""},
	{245, "Lowest Level", "Lowest level of the wet bulb zero", ""},
	{246, "Maximum e-folding", "Maximum equivalent potential temperature level", ""},
	{247, "Equilibrium", "Equilibrium level", ""},
	{248, "Shallow Convective Cloud Bottom", "Shallow convective cloud bottom level", ""},
	{249, "Shallow Convective Cloud Top", "Shallow convective cloud top level", ""},
	{251, "Deep Convective Cloud Bottom", "Deep convective cloud bottom level", ""},
	{252, "Deep Convective Cloud Top", "Deep convective cloud top level", ""},
	{253, "Lowest Bottom Level", "Lowest bottom level of supercooled liquid water layer", ""},
	{254, "Highest Top Level", "Highest top level of supercooled liquid water layer", ""},
}

var levelRanges = []RangeEntry{
	{192, 254, "Local", "Reserved for local use"},
	{255, 255, "Missing", "Missing"},
}

// LevelTable is the WMO Code Table 4.5 for fixed surface types.
var LevelTable = NewRangeTable(levelEntries, levelRanges, "Unknown level type")

// GetLevelName returns the name for a level type code.
func GetLevelName(code int) string {
	return LevelTable.Name(code)
}

// GetLevelDescription returns the full description for a level type code.
func GetLevelDescription(code int) string {
	return LevelTable.Description(code)
}

// GetLevelUnit returns the unit for a level type code.
// Returns empty string if the level type doesn't have a unit or is not found.
func GetLevelUnit(code int) string {
	if e := LevelTable.Lookup(code); e != nil {
		return e.Unit
	}
	return ""
}
