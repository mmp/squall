package squall

import (
	"github.com/mmp/squall/tables"
)

// ParameterID uniquely identifies a GRIB2 parameter using WMO standard codes.
//
// GRIB2 parameters are defined by a three-number tuple:
//   - Discipline: Product discipline (0=Meteorological, 1=Hydrological, etc.)
//   - Category: Parameter category within the discipline
//   - Number: Specific parameter within the category
//
// This matches the GRIB2 specification (WMO Manual 306, Tables 4.1 and 4.2).
type ParameterID struct {
	Discipline uint8 // WMO Code Table 0.0
	Category   uint8 // WMO Code Table 4.1 (discipline-specific)
	Number     uint8 // WMO Code Table 4.2 (category-specific within discipline)
}

// String returns the full parameter name from WMO tables.
//
// Example: "Temperature", "Geopotential Height", "Relative Humidity"
func (p ParameterID) String() string {
	return tables.GetParameterName(int(p.Discipline), int(p.Category), int(p.Number))
}

// ShortName returns the parameter's abbreviation as used by wgrib2 (e.g.,
// "TMP" or "UGRD"), or the empty string if it is unknown.
//
// Parameters in the local-use range (a discipline, category, or number from
// 192 to 254) are named according to NCEP's local table, so ShortName is
// only meaningful for them with data from NCEP.
func (p ParameterID) ShortName() string {
	param, _ := tables.LookupParameter(int(p.Discipline), int(p.Category), int(p.Number))
	return param.ShortName
}

// CategoryName returns the parameter category name.
func (p ParameterID) CategoryName() string {
	return tables.GetParameterCategoryName(int(p.Discipline), int(p.Category))
}
