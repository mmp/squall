package testutil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// NormalizeFieldName converts various field naming conventions to a canonical form.
// This allows comparison between different implementations (squall vs wgrib2).
//
// Squall uses descriptive GRIB2 table names ("Geopotential Height"),
// wgrib2 uses WMO abbreviated names ("HGT").
func NormalizeFieldName(name string) string {
	name = strings.TrimSpace(name)

	// wgrib2 abbreviated names -> canonical names
	wgrib2ToCanonical := map[string]string{
		"HGT":    "Geopotential Height",
		"TMP":    "Temperature",
		"RH":     "Relative Humidity",
		"DPT":    "Dew Point Temperature",
		"SPFH":   "Specific Humidity",
		"VVEL":   "Vertical Velocity (Pressure)",
		"UGRD":   "U-Component of Wind",
		"VGRD":   "V-Component of Wind",
		"ABSV":   "Absolute Vorticity",
		"CLMR":   "Cloud Mixing Ratio",
		"CIMIXR": "Cloud Ice Mixing Ratio",
		"RWMR":   "Rain Mixing Ratio",
		"SNMR":   "Snow Mixing Ratio",
		"GRLE":   "Graupel",
		"PRES":   "Pressure",
		"CAPE":   "Convective Available Potential Energy",
		"CIN":    "Convective Inhibition",
		"REFC":   "Composite Reflectivity",
	}

	// Check if it's a wgrib2 name
	if canonical, ok := wgrib2ToCanonical[name]; ok {
		return canonical
	}

	// Check if it's already canonical (squall format)
	// If so, return as-is
	for _, canonical := range wgrib2ToCanonical {
		if name == canonical {
			return name
		}
	}

	// Handle "Unknown parameter (D.C.P)" format from squall
	// Keep as-is for now - these are parameters not in standard tables
	if strings.HasPrefix(name, "Unknown parameter") {
		return name
	}

	return name
}

// NormalizeLevel converts various level description conventions to a canonical form.
//
// Squall uses "Isobaric 5000" (level type + value in Pascals),
// wgrib2 uses "50 mb" (value in meteorological units).
func NormalizeLevel(level string) string {
	level = strings.TrimSpace(level)

	// Parse wgrib2 format: "50 mb", "2 m above ground", etc.
	if strings.Contains(level, " mb") {
		// Convert millibars to Pascals
		re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s+mb$`)
		if matches := re.FindStringSubmatch(level); matches != nil {
			if mb, err := strconv.ParseFloat(matches[1], 64); err == nil {
				pascals := int(mb * 100)
				return fmt.Sprintf("Isobaric %d", pascals)
			}
		}
	}

	// Parse wgrib2 format: "2 m above ground"
	if strings.Contains(level, " m above ground") {
		re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s+m above ground$`)
		if matches := re.FindStringSubmatch(level); matches != nil {
			return fmt.Sprintf("Height Above Ground %s", matches[1])
		}
	}

	// Parse wgrib2 format: "surface"
	if level == "surface" {
		return "Ground or Water Surface 0"
	}

	// Parse wgrib2 format: "mean sea level"
	if level == "mean sea level" {
		return "Mean Sea Level 0"
	}

	// Parse wgrib2 format: "entire atmosphere"
	if strings.Contains(level, "entire atmosphere") {
		return "Entire Atmosphere 0"
	}

	// Parse wgrib2 format: "cloud base"
	if level == "cloud base" {
		return "Cloud Base Level 0"
	}

	// Parse wgrib2 format: "cloud top"
	if level == "cloud top" {
		return "Cloud Top Level 0"
	}

	// Parse wgrib2 format: "tropopause"
	if level == "tropopause" {
		return "Tropopause 0"
	}

	// Parse wgrib2 format: "max wind"
	if level == "max wind" {
		return "Maximum Wind Level 0"
	}

	// Parse squall format: "Isobaric 5000" -> already canonical
	if strings.HasPrefix(level, "Isobaric ") {
		return level
	}

	// Parse squall format: already canonical
	if strings.Contains(level, "Height Above Ground") ||
		strings.Contains(level, "Ground or Water Surface") ||
		strings.Contains(level, "Mean Sea Level") {
		return level
	}

	// Unknown format, return as-is
	return level
}
