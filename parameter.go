package grib

import (
	"fmt"

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

// ShortName returns a standardized short name for the parameter.
//
// This matches common meteorological abbreviations used in tools like wgrib2.
// Returns empty string if no standard abbreviation exists.
func (p ParameterID) ShortName() string {
	// Map common parameters to their standard WMO abbreviations
	// These match wgrib2's conventions for compatibility
	key := fmt.Sprintf("%d.%d.%d", p.Discipline, p.Category, p.Number)

	// Meteorological parameters (Discipline 0)
	switch key {
	// Temperature (Category 0)
	case "0.0.0":
		return "TMP"
	case "0.0.6":
		return "DPT"
	case "0.0.15":
		return "VPTMP"
	case "0.0.17":
		return "SKINT"

	// Moisture (Category 1)
	case "0.1.0":
		return "SPFH"
	case "0.1.1":
		return "RH"
	case "0.1.3":
		return "PWAT"
	case "0.1.8":
		return "APCP"
	case "0.1.11":
		return "SNOD"
	case "0.1.13":
		return "WEASD"
	case "0.1.22":
		return "CLWMR"
	case "0.1.23":
		return "ICMR"
	case "0.1.24":
		return "RWMR"
	case "0.1.25":
		return "SNMR"

	// Momentum (Category 2)
	case "0.2.0":
		return "WDIR"
	case "0.2.1":
		return "WIND"
	case "0.2.2":
		return "UGRD"
	case "0.2.3":
		return "VGRD"
	case "0.2.8":
		return "VVEL"
	case "0.2.9":
		return "DZDT"
	case "0.2.10":
		return "ABSV"
	case "0.2.11":
		return "ABSD"
	case "0.2.12":
		return "RELV"
	case "0.2.13":
		return "RELD"
	case "0.2.14":
		return "PVORT"

	// Mass (Category 3)
	case "0.3.0":
		return "PRES"
	case "0.3.1":
		return "PRMSL"
	case "0.3.3":
		return "ICAHT"
	case "0.3.4":
		return "GP"
	case "0.3.5":
		return "HGT"
	case "0.3.6":
		return "DIST"
	case "0.3.9":
		return "HPBL"

	// Cloud (Category 6)
	case "0.6.1":
		return "TCDC"
	case "0.6.3":
		return "LCDC"
	case "0.6.4":
		return "MCDC"
	case "0.6.5":
		return "HCDC"
	case "0.6.6":
		return "CWAT"
	case "0.6.22":
		return "CLMR"
	case "0.6.23":
		return "CIMR"
	case "0.6.24":
		return "RWMR"
	case "0.6.25":
		return "SNMR"
	case "0.6.32":
		return "GRLE"

	// Thermodynamic Stability (Category 7)
	case "0.7.0":
		return "PLI"
	case "0.7.6":
		return "CAPE"
	case "0.7.7":
		return "CIN"
	case "0.7.8":
		return "HLCY"

	// Radar (Category 10)
	case "0.10.0":
		return "REFZR"
	case "0.10.3":
		return "REFD"
	case "0.10.6":
		return "REFC"
	}

	// No standard abbreviation
	return ""
}

// CategoryName returns the parameter category name.
func (p ParameterID) CategoryName() string {
	return tables.GetParameterCategoryName(int(p.Discipline), int(p.Category))
}
