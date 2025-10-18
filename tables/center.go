package tables

// WMO Common Code Table C-1: Identification of Originating/Generating Centers
//
// This table identifies the centers that originate or generate GRIB2 messages.
// Only the most common centers are listed here. The full list contains 200+ entries.

var centerEntries = []*Entry{
	// Major operational centers
	{7, "NCEP", "US National Centers for Environmental Prediction", ""},
	{8, "NWS-NWSTG", "US NWS Telecommunications Gateway", ""},
	{9, "NWS-OTHER", "US NWS - Other", ""},
	{34, "JMA", "Japanese Meteorological Agency - Tokyo", ""},
	{38, "CNMC", "Beijing (National Meteorological Centre)", ""},
	{40, "RKSL", "Seoul (Korea Meteorological Administration)", ""},
	{43, "NIFS", "Tokyo (Japanese Meteorological Agency)", ""},
	{46, "RUMS", "Russian Meteorological Service - Moscow", ""},
	{52, "NCWF", "US National Hurricane Center", ""},
	{54, "CMC", "Canadian Meteorological Centre - Montreal", ""},
	{57, "USAF", "US Air Force - Global Weather Central", ""},
	{58, "FNMOC", "US Navy - Fleet Numerical Oceanography Center", ""},
	{59, "NOAA-FSL", "US NOAA Forecast Systems Laboratory", ""},
	{60, "NCAR", "US National Center for Atmospheric Research", ""},
	{74, "UK-MET", "U.K. Met Office - Exeter", ""},
	{78, "DWD", "Deutscher Wetterdienst (German Weather Service)", ""},
	{80, "CNMCA", "Rome (Italian Meteorological Service)", ""},
	{82, "EDZW", "ECMWF Operations Centre", ""},
	{84, "LFPW", "Toulouse (Meteorological Service)", ""},
	{85, "LFPW", "Toulouse (French Weather Service)", ""},
	{86, "EKMI", "Helsinki (Finnish Meteorological Institute)", ""},
	{87, "ENMI", "Oslo (Norwegian Meteorological Institute)", ""},
	{88, "ESWE", "Sweden (Swedish Meteorological and Hydrological Institute)", ""},
	{94, "NMC", "U.K. Met Office", ""},
	{97, "ESA", "European Space Agency (ESA)", ""},
	{98, "ECMWF", "European Centre for Medium-Range Weather Forecasts", ""},
	{99, "De Bilt", "De Bilt, Netherlands", ""},
	{161, "NCMRWF", "US National Centre for Medium Range Weather Forecasting", ""},
	{173, "ARL", "US Air Resources Laboratory", ""},

	// Regional centers
	{210, "FNMO", "Frascati (ESA/ESRIN)", ""},
	{211, "LANL", "Los Alamos National Laboratory", ""},
	{212, "NSSL", "National Severe Storms Laboratory", ""},
	{213, "NOAA/OAR", "NOAA Office of Oceanic and Atmospheric Research", ""},
	{214, "NESDIS", "NOAA National Environmental Satellite, Data, and Information Service", ""},
	{215, "CIRA", "NOAA Cooperative Institute for Research in the Atmosphere", ""},
	{216, "CAPS", "Center for Analysis and Prediction of Storms", ""},
	{217, "UCLA", "University Corporation for Atmospheric Research", ""},
	{218, "FSU-COAPS", "Florida State University/Center for Ocean-Atmospheric Prediction Studies", ""},
	{219, "NASA-GSFC", "NASA Goddard Space Flight Center", ""},
	{220, "NSSL", "NOAA National Severe Storms Laboratory", ""},
}

var centerRanges = []RangeEntry{
	{1, 0, "WMC", "WMC (codes allocated by WMO)"},
	{241, 254, "Local", "Reserved for local use"},
	{255, 255, "Missing", "Missing"},
}

// CenterTable is the WMO Common Code Table C-1 for originating centers.
var CenterTable = NewRangeTable(centerEntries, centerRanges, "Unknown center")

// GetCenterName returns the name for an originating center code.
func GetCenterName(code int) string {
	return CenterTable.Name(code)
}

// GetCenterDescription returns the full description for an originating center code.
func GetCenterDescription(code int) string {
	return CenterTable.Description(code)
}
