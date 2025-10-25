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

// ShortName returns a standardized short name for the parameter.
//
// This matches common meteorological abbreviations used in tools like wgrib2.
// Returns empty string if no standard abbreviation exists.
func (p ParameterID) ShortName() string {
	// Map common parameters to their standard WMO abbreviations
	// These match wgrib2's conventions for compatibility
	switch p.Discipline {
	case 0: // Meteorological parameters
		switch p.Category {
		case 0: // Temperature
			switch p.Number {
			case 0:
				return "TMP"
			case 6:
				return "DPT"
			case 15:
				return "VPTMP"
			case 17:
				return "SKINT"
			}
		case 1: // Moisture
			switch p.Number {
			case 0:
				return "SPFH"
			case 1:
				return "RH"
			case 3:
				return "PWAT"
			case 8:
				return "APCP"
			case 11:
				return "SNOD"
			case 13:
				return "WEASD"
			case 22:
				return "CLMR"
			case 23:
				return "ICMR"
			case 24:
				return "RWMR"
			case 25:
				return "SNMR"
			case 32:
				return "GRLE"
			case 82:
				return "CIMIXR"
			case 242:
				return "RHPW" // Relative humidity with respect to precipitable water
			}
		case 2: // Momentum
			switch p.Number {
			case 0:
				return "WDIR"
			case 1:
				return "WIND"
			case 2:
				return "UGRD"
			case 3:
				return "VGRD"
			case 8:
				return "VVEL"
			case 9:
				return "DZDT"
			case 10:
				return "ABSV"
			case 11:
				return "ABSD"
			case 12:
				return "RELV"
			case 13:
				return "RELD"
			case 14:
				return "PVORT"
			}
		case 3: // Mass
			switch p.Number {
			case 0:
				return "PRES"
			case 1:
				return "PRMSL"
			case 3:
				return "ICAHT"
			case 4:
				return "GP"
			case 5:
				return "HGT"
			case 6:
				return "DIST"
			case 9:
				return "HPBL"
			case 205:
				return "LAYTH" // Layer thickness
			}
		case 6: // Cloud
			switch p.Number {
			case 1:
				return "TCDC"
			case 3:
				return "LCDC"
			case 4:
				return "MCDC"
			case 5:
				return "HCDC"
			case 6:
				return "CWAT"
			case 22:
				return "CLMR"
			case 23:
				return "CIMR"
			case 24:
				return "RWMR"
			case 25:
				return "SNMR"
			case 32:
				return "GRLE"
			}
		case 7: // Thermodynamic Stability
			switch p.Number {
			case 0:
				return "PLI"
			case 6:
				return "CAPE"
			case 7:
				return "CIN"
			case 8:
				return "HLCY"
			case 205:
				return "ESP" // Echoes specifying type of precipitation
			case 206:
				return "CANGLE" // Convection angle
			}
		case 10: // Radar
			switch p.Number {
			case 0:
				return "REFZR"
			case 3:
				return "REFD"
			case 6:
				return "REFC"
			}
		}
	case 2: // Land surface parameters
		switch p.Category {
		case 0: // Vegetation/Biomass
			switch p.Number {
			case 0:
				return "LAND" // Land cover
			case 2:
				return "TSOIL" // Soil temperature
			case 192:
				return "SOILW" // Volumetric soil moisture
			}
		}
	case 3: // Space products
		switch p.Category {
		case 192: // Forecast Satellite Imagery
			switch p.Number {
			case 1:
				return "SBT123" // Simulated brightness temperature for GOES 12, channel 3
			case 2:
				return "SBT124" // Simulated brightness temperature for GOES 12, channel 4
			case 7:
				return "SBT113" // Simulated brightness temperature for GOES 11, channel 3
			case 8:
				return "SBT114" // Simulated brightness temperature for GOES 11, channel 4
			}
		}
	case 10: // Oceanographic parameters
		switch p.Category {
		case 2: // Ice
			switch p.Number {
			case 0:
				return "ICEC" // Ice cover
			}
		}
	}

	// No standard abbreviation
	return ""
}

// CategoryName returns the parameter category name.
func (p ParameterID) CategoryName() string {
	return tables.GetParameterCategoryName(int(p.Discipline), int(p.Category))
}
