package tables

// WMO Code Table 0.0: Discipline of Processed Data
//
// This table defines the disciplines (domains) of meteorological and related data.
// Each discipline has its own set of parameter tables.

var disciplineEntries = []*Entry{
	{0, "Meteorological", "Meteorological products", ""},
	{1, "Hydrological", "Hydrological products", ""},
	{2, "Land Surface", "Land surface products", ""},
	{3, "Space", "Space products", ""},
	{4, "Space Weather", "Space weather products", ""},
	{10, "Oceanographic", "Oceanographic products", ""},
	{20, "Health", "Health and socioeconomic impacts", ""},
}

var disciplineRanges = []RangeEntry{
	{192, 254, "Local", "Reserved for local use"},
	{255, 255, "Missing", "Missing"},
}

// DisciplineTable is the WMO Code Table 0.0 for discipline codes.
var DisciplineTable = NewRangeTable(disciplineEntries, disciplineRanges, "Unknown discipline")

// GetDisciplineName returns the name for a discipline code.
// This is a convenience function for the most common use case.
func GetDisciplineName(code int) string {
	return DisciplineTable.Name(code)
}

// GetDisciplineDescription returns the full description for a discipline code.
func GetDisciplineDescription(code int) string {
	return DisciplineTable.Description(code)
}
