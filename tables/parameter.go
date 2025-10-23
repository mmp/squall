package tables

import "fmt"

// WMO Code Table 4.1: Parameter Category by Product Discipline
//
// This table defines parameter categories within each discipline.
// The actual parameters are defined in Table 4.2, which is further subdivided
// by discipline and category.

// Discipline 0: Meteorological Products
var parameterCategoryMeteorologicalEntries = []*Entry{
	{0, "Temperature", "Temperature", ""},
	{1, "Moisture", "Moisture", ""},
	{2, "Momentum", "Momentum", ""},
	{3, "Mass", "Mass", ""},
	{4, "Short-wave Radiation", "Short-wave radiation", ""},
	{5, "Long-wave Radiation", "Long-wave radiation", ""},
	{6, "Cloud", "Cloud", ""},
	{7, "Thermodynamic Stability", "Thermodynamic stability indices", ""},
	{8, "Aerosols", "Aerosols", ""},
	{9, "Trace Gases", "Trace gases (e.g. ozone, CO2)", ""},
	{10, "Radar", "Radar", ""},
	{11, "Radar Imagery", "Radar imagery", ""},
	{12, "Electrodynamics", "Electrodynamics", ""},
	{13, "Nuclear/Radiology", "Nuclear/radiology", ""},
	{14, "Physical Atmospheric", "Physical atmospheric properties", ""},
	{15, "Atmospheric Chemical", "Atmospheric chemical constituents", ""},
	{16, "Forecast Radar Imagery", "Forecast radar imagery", ""},
	{17, "Electrodynamics", "Electrodynamics", ""},
	{18, "Signal Processing", "Signal processing", ""},
	{19, "Vegetation/Biomass", "Vegetation/biomass", ""},
	{20, "Atmospheric", "Atmospheric", ""},
	{190, "CCITT IA5 String", "CCITT IA5 string", ""},
	{191, "Miscellaneous", "Miscellaneous", ""},
}

// Discipline 1: Hydrological Products
var parameterCategoryHydrologicalEntries = []*Entry{
	{0, "Hydrology Basic", "Hydrology basic products", ""},
	{1, "Hydrology Probabilities", "Hydrology probabilities", ""},
	{2, "Inland Water", "Inland water and sediment properties", ""},
}

// Discipline 2: Land Surface Products
var parameterCategoryLandSurfaceEntries = []*Entry{
	{0, "Vegetation/Biomass", "Vegetation/biomass", ""},
	{1, "Agricultural", "Agricultural/aquacultural special products", ""},
	{2, "Transportation", "Transportation-related products", ""},
	{3, "Soil Products", "Soil products", ""},
	{4, "Fire Weather", "Fire weather products", ""},
	{5, "Glaciers and Ice Sheets", "Glaciers and inland ice", ""},
}

// Discipline 3: Space Products
var parameterCategorySpaceEntries = []*Entry{
	{0, "Image Format", "Image format products", ""},
	{1, "Quantitative", "Quantitative products", ""},
	{2, "Cloud Properties", "Cloud properties", ""},
	{3, "Flight Rules", "Flight rule conditions", ""},
	{4, "Volcanic Ash", "Volcanic ash", ""},
	{5, "Sea-surface Temperature", "Sea-surface temperature", ""},
	{6, "Solar Radiation", "Solar radiation", ""},
}

// Discipline 10: Oceanographic Products
var parameterCategoryOceanographicEntries = []*Entry{
	{0, "Waves", "Waves", ""},
	{1, "Currents", "Currents", ""},
	{2, "Ice", "Ice", ""},
	{3, "Surface Properties", "Surface properties", ""},
	{4, "Sub-surface Properties", "Sub-surface properties", ""},
	{191, "Miscellaneous", "Miscellaneous", ""},
}

// ParameterCategoryTable is a discipline-specific table for parameter categories.
var ParameterCategoryTable *DisciplineSpecificTable

func init() {
	ParameterCategoryTable = NewDisciplineSpecificTable("Unknown parameter category")

	// Add tables for each discipline
	ParameterCategoryTable.AddTable(0, NewSimpleTable(parameterCategoryMeteorologicalEntries, "Unknown category"))
	ParameterCategoryTable.AddTable(1, NewSimpleTable(parameterCategoryHydrologicalEntries, "Unknown category"))
	ParameterCategoryTable.AddTable(2, NewSimpleTable(parameterCategoryLandSurfaceEntries, "Unknown category"))
	ParameterCategoryTable.AddTable(3, NewSimpleTable(parameterCategorySpaceEntries, "Unknown category"))
	ParameterCategoryTable.AddTable(10, NewSimpleTable(parameterCategoryOceanographicEntries, "Unknown category"))
}

// GetParameterCategoryName returns the name for a parameter category code
// within a specific discipline.
func GetParameterCategoryName(discipline, category int) string {
	return ParameterCategoryTable.Name(discipline, category)
}

// GetParameterCategoryDescription returns the full description for a parameter
// category code within a specific discipline.
func GetParameterCategoryDescription(discipline, category int) string {
	return ParameterCategoryTable.Description(discipline, category)
}

// WMO Code Table 4.2: Parameter Number by Product Discipline and Parameter Category
//
// This table is extremely large and varies by both discipline and category.
// We implement only the most common parameters from Discipline 0 (Meteorological).
//
// Full implementation would require hundreds of entries across multiple tables.

// Discipline 0, Category 0: Temperature parameters
var parameterTemperatureEntries = []*Entry{
	{0, "Temperature", "Temperature", "K"},
	{1, "Virtual Temperature", "Virtual temperature", "K"},
	{2, "Potential Temperature", "Potential temperature", "K"},
	{3, "Pseudo-Adiabatic Potential Temperature", "Pseudo-adiabatic potential temperature", "K"},
	{4, "Maximum Temperature", "Maximum temperature", "K"},
	{5, "Minimum Temperature", "Minimum temperature", "K"},
	{6, "Dew Point Temperature", "Dew point temperature", "K"},
	{7, "Dew Point Depression", "Dew point depression (or deficit)", "K"},
	{8, "Lapse Rate", "Lapse rate", "K/m"},
	{9, "Temperature Anomaly", "Temperature anomaly", "K"},
	{10, "Latent Heat", "Latent heat net flux", "W/m²"},
	{11, "Sensible Heat", "Sensible heat net flux", "W/m²"},
	{12, "Heat Index", "Heat index", "K"},
	{13, "Wind Chill", "Wind chill factor", "K"},
	{14, "Minimum Dew Point", "Minimum dew point depression", "K"},
	{15, "Virtual Potential Temperature", "Virtual potential temperature", "K"},
	{16, "Snow Phase Change Heat Flux", "Snow phase change heat flux", "W/m²"},
	{17, "Skin Temperature", "Skin temperature", "K"},
	{18, "Snow Temperature", "Snow temperature", "K"},
	{19, "Turbulent Transfer Coefficient", "Turbulent transfer coefficient for heat", "Numeric"},
	{20, "Turbulent Diffusion Coefficient", "Turbulent diffusion coefficient for heat", "m²/s"},
	{21, "Apparent Temperature", "Apparent temperature", "K"},
	{192, "Snow Phase Change Heat Flux", "Snow phase change heat flux", "W/m²"},
}

// Discipline 0, Category 1: Moisture parameters
var parameterMoistureEntries = []*Entry{
	{0, "Specific Humidity", "Specific humidity", "kg/kg"},
	{1, "Relative Humidity", "Relative humidity", "%"},
	{2, "Humidity Mixing Ratio", "Humidity mixing ratio", "kg/kg"},
	{3, "Precipitable Water", "Precipitable water", "kg/m²"},
	{4, "Vapor Pressure", "Vapor pressure", "Pa"},
	{5, "Saturation Deficit", "Saturation deficit", "Pa"},
	{6, "Evaporation", "Evaporation", "kg/m²"},
	{7, "Precipitation Rate", "Precipitation rate", "kg/(m² s)"},
	{8, "Total Precipitation", "Total precipitation", "kg/m²"},
	{9, "Large Scale Precipitation", "Large scale precipitation", "kg/m²"},
	{10, "Convective Precipitation", "Convective precipitation", "kg/m²"},
	{11, "Snow Depth", "Snow depth", "m"},
	{12, "Snowfall Rate Water Equivalent", "Snowfall rate water equivalent", "kg/(m² s)"},
	{13, "Water Equiv of Accumulated Snow", "Water equivalent of accumulated snow depth", "kg/m²"},
	{14, "Convective Snow", "Convective snow", "kg/m²"},
	{15, "Large Scale Snow", "Large scale snow", "kg/m²"},
	{16, "Snow Melt", "Snow melt", "kg/m²"},
	{17, "Snow Age", "Snow age", "day"},
	{18, "Absolute Humidity", "Absolute humidity", "kg/m³"},
	{19, "Precipitation Type", "Precipitation type", "Code table 4.201"},
	{20, "Integrated Liquid Water", "Integrated liquid water", "kg/m²"},
	{21, "Condensate", "Condensate", "kg/kg"},
	{22, "Cloud Mixing Ratio", "Cloud mixing ratio", "kg/kg"},
	{23, "Ice Water Mixing Ratio", "Ice water mixing ratio", "kg/kg"},
	{24, "Rain Mixing Ratio", "Rain mixing ratio", "kg/kg"},
	{25, "Snow Mixing Ratio", "Snow mixing ratio", "kg/kg"},
	{26, "Horizontal Moisture Convergence", "Horizontal moisture convergence", "kg/(kg s)"},
	{27, "Maximum Relative Humidity", "Maximum relative humidity", "%"},
	{28, "Maximum Absolute Humidity", "Maximum absolute humidity", "kg/m³"},
	{29, "Total Snowfall", "Total snowfall", "m"},
	{32, "Graupel", "Graupel (precipitation-sized ice particles)", "kg/kg"},
	{82, "Cloud Ice Mixing Ratio", "Cloud ice mixing ratio", "kg/kg"},
}

// Discipline 0, Category 2: Momentum parameters
var parameterMomentumEntries = []*Entry{
	{0, "Wind Direction", "Wind direction (from which blowing)", "°"},
	{1, "Wind Speed", "Wind speed", "m/s"},
	{2, "U-Component of Wind", "U-component of wind", "m/s"},
	{3, "V-Component of Wind", "V-component of wind", "m/s"},
	{4, "Stream Function", "Stream function", "m²/s"},
	{5, "Velocity Potential", "Velocity potential", "m²/s"},
	{6, "Montgomery Stream Function", "Montgomery stream function", "m²/s²"},
	{7, "Sigma Vertical Velocity", "Sigma coordinate vertical velocity", "1/s"},
	{8, "Vertical Velocity (Pressure)", "Vertical velocity (pressure)", "Pa/s"},
	{9, "Vertical Velocity (Geometric)", "Vertical velocity (geometric)", "m/s"},
	{10, "Absolute Vorticity", "Absolute vorticity", "1/s"},
	{11, "Absolute Divergence", "Absolute divergence", "1/s"},
	{12, "Relative Vorticity", "Relative vorticity", "1/s"},
	{13, "Relative Divergence", "Relative divergence", "1/s"},
	{14, "Potential Vorticity", "Potential vorticity", "K m²/(kg s)"},
	{15, "Vertical U Shear", "Vertical u-component shear", "1/s"},
	{16, "Vertical V Shear", "Vertical v-component shear", "1/s"},
	{17, "Momentum Flux U", "Momentum flux, u-component", "N/m²"},
	{18, "Momentum Flux V", "Momentum flux, v-component", "N/m²"},
	{19, "Wind Mixing Energy", "Wind mixing energy", "J"},
	{20, "Boundary Layer Dissipation", "Boundary layer dissipation", "W/m²"},
	{21, "Maximum Wind Speed", "Maximum wind speed", "m/s"},
	{22, "Wind Gust", "Wind speed (gust)", "m/s"},
	{23, "U-Component Gust", "U-component of wind (gust)", "m/s"},
	{24, "V-Component Gust", "V-component of wind (gust)", "m/s"},
}

// Discipline 0, Category 3: Mass parameters
var parameterMassEntries = []*Entry{
	{0, "Pressure", "Pressure", "Pa"},
	{1, "Pressure Reduced to MSL", "Pressure reduced to MSL", "Pa"},
	{2, "Pressure Tendency", "Pressure tendency", "Pa/s"},
	{3, "ICAO Standard Atmosphere", "ICAO standard atmosphere reference height", "m"},
	{4, "Geopotential", "Geopotential", "m²/s²"},
	{5, "Geopotential Height", "Geopotential height", "gpm"},
	{6, "Geometric Height", "Geometric height", "m"},
	{7, "Standard Deviation Height", "Standard deviation of height", "m"},
	{8, "Pressure Anomaly", "Pressure anomaly", "Pa"},
	{9, "Geopotential Height Anomaly", "Geopotential height anomaly", "gpm"},
	{10, "Density", "Density", "kg/m³"},
	{11, "Altimeter Setting", "Altimeter setting", "Pa"},
	{12, "Thickness", "Thickness", "m"},
	{13, "Pressure Altitude", "Pressure altitude", "m"},
	{14, "Density Altitude", "Density altitude", "m"},
	{15, "5-Wave Geopotential Height", "5-wave geopotential height", "gpm"},
	{16, "Zonal Flux Gravity Wave Stress", "Zonal flux of gravity wave stress", "N/m²"},
	{17, "Meridional Flux Gravity Wave Stress", "Meridional flux of gravity wave stress", "N/m²"},
	{18, "Planetary Boundary Layer Height", "Planetary boundary layer height", "m"},
	{19, "5-Wave Geopotential Height Anomaly", "5-wave geopotential height anomaly", "gpm"},
	{20, "Standard Deviation Pressure", "Standard deviation of pressure", "Pa"},
}

// ParameterNumberTable is the global lookup table for GRIB2 parameter names.
// It maps discipline, category, and parameter numbers to human-readable names
// according to WMO GRIB2 Code Table 4.2.
var ParameterNumberTable *DisciplineSpecificTable

func init() {
	// Create the main parameter number table
	ParameterNumberTable = NewDisciplineSpecificTable("Unknown parameter")

	// For now, only implement Discipline 0 (Meteorological)
	// We'll need a two-level lookup: discipline -> category -> parameter

	// This is simplified - in reality, we'd need a more complex structure
	// to handle discipline->category->parameter hierarchy
}

// GetParameterName returns the name for a specific parameter.
// This is a simplified version - full implementation would require
// discipline and category context.
func GetParameterName(discipline, category, parameter int) string {
	// For now, implement a simple lookup for common cases
	// Full implementation in future phases

	// Meteorological products (discipline 0)
	if discipline == 0 {
		switch category {
		case 0: // Temperature
			if entry := lookupInEntries(parameterTemperatureEntries, parameter); entry != nil {
				return entry.Name
			}
		case 1: // Moisture
			if entry := lookupInEntries(parameterMoistureEntries, parameter); entry != nil {
				return entry.Name
			}
		case 2: // Momentum
			if entry := lookupInEntries(parameterMomentumEntries, parameter); entry != nil {
				return entry.Name
			}
		case 3: // Mass
			if entry := lookupInEntries(parameterMassEntries, parameter); entry != nil {
				return entry.Name
			}
		}
	}

	return fmt.Sprintf("Unknown parameter (%d.%d.%d)", discipline, category, parameter)
}

// GetParameterUnit returns the unit for a specific parameter.
func GetParameterUnit(discipline, category, parameter int) string {
	if discipline == 0 {
		switch category {
		case 0: // Temperature
			if entry := lookupInEntries(parameterTemperatureEntries, parameter); entry != nil {
				return entry.Unit
			}
		case 1: // Moisture
			if entry := lookupInEntries(parameterMoistureEntries, parameter); entry != nil {
				return entry.Unit
			}
		case 2: // Momentum
			if entry := lookupInEntries(parameterMomentumEntries, parameter); entry != nil {
				return entry.Unit
			}
		case 3: // Mass
			if entry := lookupInEntries(parameterMassEntries, parameter); entry != nil {
				return entry.Unit
			}
		}
	}

	return ""
}

// Helper function to lookup in entry slices
func lookupInEntries(entries []*Entry, code int) *Entry {
	for _, e := range entries {
		if e.Code == code {
			return e
		}
	}
	return nil
}
