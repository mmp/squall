// Package tables provides WMO code table lookups for GRIB2 metadata.
//
// GRIB2 uses numerous code tables defined by WMO (World Meteorological Organization)
// to encode metadata. This package provides a data-driven approach to table lookups,
// replacing the switch/case statements found in wgrib2.
//
// All tables are implemented as Go data structures (maps, slices, structs) for:
//   - Fast lookups (O(1) for maps)
//   - Easy maintenance and updates
//   - Type safety
//   - No code generation required at runtime
package tables

import "fmt"

// Entry represents a single entry in a WMO code table.
type Entry struct {
	Code        int    // Numeric code
	Name        string // Short name (e.g., "Temperature")
	Description string // Full description (e.g., "Temperature (K)")
	Unit        string // Unit of measurement, if applicable
}

// Table represents a WMO code table with lookup capabilities.
type Table interface {
	// Lookup returns the entry for the given code.
	// Returns nil if the code is not found.
	Lookup(code int) *Entry

	// Name returns the short name for the given code.
	// Returns a fallback string if the code is not found.
	Name(code int) string

	// Description returns the full description for the given code.
	// Returns a fallback string if the code is not found.
	Description(code int) string

	// Exists checks if a code exists in the table.
	Exists(code int) bool

	// AllCodes returns all valid codes in the table.
	AllCodes() []int
}

// SimpleTable is a basic implementation of Table using a map.
type SimpleTable struct {
	entries      map[int]*Entry
	fallbackName string // Used when code is not found
}

// NewSimpleTable creates a new SimpleTable from a slice of entries.
func NewSimpleTable(entries []*Entry, fallbackName string) *SimpleTable {
	m := make(map[int]*Entry, len(entries))
	for _, e := range entries {
		m[e.Code] = e
	}
	return &SimpleTable{
		entries:      m,
		fallbackName: fallbackName,
	}
}

// Lookup returns the entry for the given code.
func (t *SimpleTable) Lookup(code int) *Entry {
	return t.entries[code]
}

// Name returns the short name for the given code.
func (t *SimpleTable) Name(code int) string {
	if e := t.entries[code]; e != nil {
		return e.Name
	}
	return fmt.Sprintf("%s (%d)", t.fallbackName, code)
}

// Description returns the full description for the given code.
func (t *SimpleTable) Description(code int) string {
	if e := t.entries[code]; e != nil {
		if e.Description != "" {
			return e.Description
		}
		return e.Name
	}
	return fmt.Sprintf("%s (%d)", t.fallbackName, code)
}

// Exists checks if a code exists in the table.
func (t *SimpleTable) Exists(code int) bool {
	_, ok := t.entries[code]
	return ok
}

// AllCodes returns all valid codes in the table.
func (t *SimpleTable) AllCodes() []int {
	codes := make([]int, 0, len(t.entries))
	for code := range t.entries {
		codes = append(codes, code)
	}
	return codes
}

// RangeTable handles tables where codes are organized by ranges.
// For example, codes 192-254 might be "reserved for local use".
type RangeTable struct {
	entries      map[int]*Entry
	ranges       []RangeEntry
	fallbackName string
}

// RangeEntry represents a range of codes with a common description.
type RangeEntry struct {
	Start       int    // Start of range (inclusive)
	End         int    // End of range (inclusive)
	Name        string // Name for codes in this range
	Description string // Description for codes in this range
}

// NewRangeTable creates a new RangeTable.
func NewRangeTable(entries []*Entry, ranges []RangeEntry, fallbackName string) *RangeTable {
	m := make(map[int]*Entry, len(entries))
	for _, e := range entries {
		m[e.Code] = e
	}
	return &RangeTable{
		entries:      m,
		ranges:       ranges,
		fallbackName: fallbackName,
	}
}

// Lookup returns the entry for the given code.
func (t *RangeTable) Lookup(code int) *Entry {
	if e := t.entries[code]; e != nil {
		return e
	}

	// Check ranges
	for _, r := range t.ranges {
		if code >= r.Start && code <= r.End {
			return &Entry{
				Code:        code,
				Name:        r.Name,
				Description: r.Description,
			}
		}
	}

	return nil
}

// Name returns the short name for the given code.
func (t *RangeTable) Name(code int) string {
	if e := t.Lookup(code); e != nil {
		return e.Name
	}
	return fmt.Sprintf("%s (%d)", t.fallbackName, code)
}

// Description returns the full description for the given code.
func (t *RangeTable) Description(code int) string {
	if e := t.Lookup(code); e != nil {
		if e.Description != "" {
			return e.Description
		}
		return e.Name
	}
	return fmt.Sprintf("%s (%d)", t.fallbackName, code)
}

// Exists checks if a code exists in the table.
func (t *RangeTable) Exists(code int) bool {
	return t.Lookup(code) != nil
}

// AllCodes returns all explicitly defined codes (not range codes).
func (t *RangeTable) AllCodes() []int {
	codes := make([]int, 0, len(t.entries))
	for code := range t.entries {
		codes = append(codes, code)
	}
	return codes
}

// DisciplineSpecificTable handles tables that vary by discipline.
// For example, Table 4.2 (parameter number) depends on both discipline and category.
type DisciplineSpecificTable struct {
	tables       map[int]Table // tables[discipline] = table for that discipline
	fallbackName string
}

// NewDisciplineSpecificTable creates a new DisciplineSpecificTable.
func NewDisciplineSpecificTable(fallbackName string) *DisciplineSpecificTable {
	return &DisciplineSpecificTable{
		tables:       make(map[int]Table),
		fallbackName: fallbackName,
	}
}

// AddTable adds a table for a specific discipline.
func (t *DisciplineSpecificTable) AddTable(discipline int, table Table) {
	t.tables[discipline] = table
}

// Lookup returns the entry for the given discipline and code.
func (t *DisciplineSpecificTable) Lookup(discipline, code int) *Entry {
	if table, ok := t.tables[discipline]; ok {
		return table.Lookup(code)
	}
	return nil
}

// Name returns the short name for the given discipline and code.
func (t *DisciplineSpecificTable) Name(discipline, code int) string {
	if table, ok := t.tables[discipline]; ok {
		return table.Name(code)
	}
	return fmt.Sprintf("%s (%d)", t.fallbackName, code)
}

// Description returns the full description for the given discipline and code.
func (t *DisciplineSpecificTable) Description(discipline, code int) string {
	if table, ok := t.tables[discipline]; ok {
		return table.Description(code)
	}
	return fmt.Sprintf("%s (%d)", t.fallbackName, code)
}

// Exists checks if a code exists for the given discipline.
func (t *DisciplineSpecificTable) Exists(discipline, code int) bool {
	if table, ok := t.tables[discipline]; ok {
		return table.Exists(code)
	}
	return false
}
