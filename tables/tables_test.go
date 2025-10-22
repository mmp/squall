package tables

import (
	"testing"
)

// Test SimpleTable
func TestSimpleTable(t *testing.T) {
	entries := []*Entry{
		{0, "Zero", "Entry zero", "unit0"},
		{1, "One", "Entry one", "unit1"},
		{10, "Ten", "Entry ten", "unit10"},
	}

	table := NewSimpleTable(entries, "Unknown")

	// Test Lookup
	if e := table.Lookup(0); e == nil || e.Name != "Zero" {
		t.Error("Lookup(0) failed")
	}

	if e := table.Lookup(999); e != nil {
		t.Error("Lookup(999) should return nil")
	}

	// Test Name
	if name := table.Name(1); name != "One" {
		t.Errorf("Name(1) = %q, want \"One\"", name)
	}

	if name := table.Name(999); name != "Unknown (999)" {
		t.Errorf("Name(999) = %q, want \"Unknown (999)\"", name)
	}

	// Test Exists
	if !table.Exists(0) {
		t.Error("Exists(0) should be true")
	}

	if table.Exists(999) {
		t.Error("Exists(999) should be false")
	}

	// Test AllCodes
	codes := table.AllCodes()
	if len(codes) != 3 {
		t.Errorf("AllCodes length = %d, want 3", len(codes))
	}
}

// Test RangeTable
func TestRangeTable(t *testing.T) {
	entries := []*Entry{
		{0, "Zero", "Entry zero", ""},
		{1, "One", "Entry one", ""},
	}

	ranges := []RangeEntry{
		{10, 20, "Range10-20", "Range from 10 to 20"},
		{100, 200, "Range100-200", "Range from 100 to 200"},
	}

	table := NewRangeTable(entries, ranges, "Unknown")

	// Test explicit entries
	if e := table.Lookup(0); e == nil || e.Name != "Zero" {
		t.Error("Lookup(0) failed")
	}

	// Test range entries
	if e := table.Lookup(15); e == nil || e.Name != "Range10-20" {
		t.Errorf("Lookup(15) failed: got %v", e)
	}

	if e := table.Lookup(150); e == nil || e.Name != "Range100-200" {
		t.Errorf("Lookup(150) failed: got %v", e)
	}

	// Test out of range
	if e := table.Lookup(999); e != nil {
		t.Error("Lookup(999) should return nil")
	}

	// Test Name for range
	if name := table.Name(15); name != "Range10-20" {
		t.Errorf("Name(15) = %q, want \"Range10-20\"", name)
	}

	// Test Exists for range
	if !table.Exists(15) {
		t.Error("Exists(15) should be true")
	}

	if table.Exists(999) {
		t.Error("Exists(999) should be false")
	}
}

// Test DisciplineSpecificTable
func TestDisciplineSpecificTable(t *testing.T) {
	dst := NewDisciplineSpecificTable("Unknown")

	// Add tables for different disciplines
	disc0Entries := []*Entry{
		{0, "D0P0", "Discipline 0 Parameter 0", ""},
		{1, "D0P1", "Discipline 0 Parameter 1", ""},
	}
	dst.AddTable(0, NewSimpleTable(disc0Entries, "Unknown D0"))

	disc1Entries := []*Entry{
		{0, "D1P0", "Discipline 1 Parameter 0", ""},
		{1, "D1P1", "Discipline 1 Parameter 1", ""},
	}
	dst.AddTable(1, NewSimpleTable(disc1Entries, "Unknown D1"))

	// Test lookups
	if e := dst.Lookup(0, 0); e == nil || e.Name != "D0P0" {
		t.Error("Lookup(0,0) failed")
	}

	if e := dst.Lookup(1, 1); e == nil || e.Name != "D1P1" {
		t.Error("Lookup(1,1) failed")
	}

	// Test non-existent discipline
	if e := dst.Lookup(999, 0); e != nil {
		t.Error("Lookup(999,0) should return nil")
	}

	// Test Name
	if name := dst.Name(0, 1); name != "D0P1" {
		t.Errorf("Name(0,1) = %q, want \"D0P1\"", name)
	}

	// Test Exists
	if !dst.Exists(0, 0) {
		t.Error("Exists(0,0) should be true")
	}

	if dst.Exists(999, 0) {
		t.Error("Exists(999,0) should be false")
	}
}

// Test Discipline Table (Table 0.0)
func TestDisciplineTable(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "Meteorological"},
		{1, "Hydrological"},
		{2, "Land Surface"},
		{10, "Oceanographic"},
		{192, "Local"},   // Range entry
		{255, "Missing"}, // Range entry
	}

	for _, tt := range tests {
		got := GetDisciplineName(tt.code)
		if got != tt.want {
			t.Errorf("GetDisciplineName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}

	// Test unknown code
	unknown := GetDisciplineName(99)
	if unknown != "Unknown discipline (99)" {
		t.Errorf("GetDisciplineName(99) = %q, want \"Unknown discipline (99)\"", unknown)
	}
}

// Test Center Table (Common C-1)
func TestCenterTable(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{7, "NCEP"},
		{98, "ECMWF"},
		{34, "JMA"},
		{54, "CMC"},
	}

	for _, tt := range tests {
		got := GetCenterName(tt.code)
		if got != tt.want {
			t.Errorf("GetCenterName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// Test Time Significance Table (Table 1.2)
func TestTimeSignificanceTable(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "Analysis"},
		{1, "Start of Forecast"},
		{2, "Verifying Time"},
		{3, "Observation Time"},
	}

	for _, tt := range tests {
		got := GetTimeSignificanceName(tt.code)
		if got != tt.want {
			t.Errorf("GetTimeSignificanceName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// Test Production Status Table (Table 1.3)
func TestProductionStatusTable(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "Operational"},
		{1, "Experimental"},
		{2, "Research"},
		{3, "Re-analysis"},
	}

	for _, tt := range tests {
		got := GetProductionStatusName(tt.code)
		if got != tt.want {
			t.Errorf("GetProductionStatusName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// Test Data Type Table (Table 1.4)
func TestDataTypeTable(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "Analysis"},
		{1, "Forecast"},
		{2, "Analysis & Forecast"},
	}

	for _, tt := range tests {
		got := GetDataTypeName(tt.code)
		if got != tt.want {
			t.Errorf("GetDataTypeName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// Test Level Table (Table 4.5)
func TestLevelTable(t *testing.T) {
	tests := []struct {
		code int
		name string
		unit string
	}{
		{1, "Surface", ""},
		{100, "Isobaric", "Pa"},
		{103, "Height AGL", "m"},
		{106, "Depth BG", "m"},
	}

	for _, tt := range tests {
		if name := GetLevelName(tt.code); name != tt.name {
			t.Errorf("GetLevelName(%d) = %q, want %q", tt.code, name, tt.name)
		}

		if unit := GetLevelUnit(tt.code); unit != tt.unit {
			t.Errorf("GetLevelUnit(%d) = %q, want %q", tt.code, unit, tt.unit)
		}
	}
}

// Test Parameter Category Table (Table 4.1)
func TestParameterCategoryTable(t *testing.T) {
	tests := []struct {
		discipline int
		category   int
		want       string
	}{
		{0, 0, "Temperature"},
		{0, 1, "Moisture"},
		{0, 2, "Momentum"},
		{1, 0, "Hydrology Basic"},
		{10, 0, "Waves"},
	}

	for _, tt := range tests {
		got := GetParameterCategoryName(tt.discipline, tt.category)
		if got != tt.want {
			t.Errorf("GetParameterCategoryName(%d,%d) = %q, want %q",
				tt.discipline, tt.category, got, tt.want)
		}
	}
}

// Test Parameter Name Table (Table 4.2 subset)
func TestParameterNameTable(t *testing.T) {
	tests := []struct {
		discipline int
		category   int
		parameter  int
		name       string
		unit       string
	}{
		{0, 0, 0, "Temperature", "K"},
		{0, 0, 5, "Minimum Temperature", "K"},
		{0, 1, 0, "Specific Humidity", "kg/kg"},
		{0, 1, 1, "Relative Humidity", "%"},
		{0, 1, 8, "Total Precipitation", "kg/m²"},
		{0, 2, 0, "Wind Direction", "°"},
		{0, 2, 1, "Wind Speed", "m/s"},
		{0, 2, 2, "U-Component of Wind", "m/s"},
		{0, 3, 0, "Pressure", "Pa"},
		{0, 3, 5, "Geopotential Height", "gpm"},
	}

	for _, tt := range tests {
		name := GetParameterName(tt.discipline, tt.category, tt.parameter)
		if name != tt.name {
			t.Errorf("GetParameterName(%d,%d,%d) = %q, want %q",
				tt.discipline, tt.category, tt.parameter, name, tt.name)
		}

		unit := GetParameterUnit(tt.discipline, tt.category, tt.parameter)
		if unit != tt.unit {
			t.Errorf("GetParameterUnit(%d,%d,%d) = %q, want %q",
				tt.discipline, tt.category, tt.parameter, unit, tt.unit)
		}
	}

	// Test unknown parameter
	unknown := GetParameterName(0, 0, 999)
	if unknown != "Unknown parameter (0.0.999)" {
		t.Errorf("GetParameterName(0,0,999) = %q", unknown)
	}
}
