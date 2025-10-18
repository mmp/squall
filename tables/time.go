package tables

// WMO Code Table 1.2: Significance of Reference Time
//
// This table defines the meaning of the reference time in Section 1.

var timeSignificanceEntries = []*Entry{
	{0, "Analysis", "Analysis", ""},
	{1, "Start of Forecast", "Start of forecast", ""},
	{2, "Verifying Time", "Verifying time of forecast", ""},
	{3, "Observation Time", "Observation time", ""},
	{4, "Analysis Valid Time", "Time of analysis valid at reference time", ""},
}

var timeSignificanceRanges = []RangeEntry{
	{192, 254, "Local", "Reserved for local use"},
	{255, 255, "Missing", "Missing"},
}

// TimeSignificanceTable is the WMO Code Table 1.2.
var TimeSignificanceTable = NewRangeTable(timeSignificanceEntries, timeSignificanceRanges, "Unknown time significance")

// GetTimeSignificanceName returns the name for a time significance code.
func GetTimeSignificanceName(code int) string {
	return TimeSignificanceTable.Name(code)
}

// WMO Code Table 1.3: Production Status of Processed Data
//
// This table defines the production status of the data.

var productionStatusEntries = []*Entry{
	{0, "Operational", "Operational products", ""},
	{1, "Experimental", "Operational test products", ""},
	{2, "Research", "Research products", ""},
	{3, "Re-analysis", "Re-analysis products", ""},
	{4, "TIGGE", "THORPEX Interactive Grand Global Ensemble (TIGGE)", ""},
	{5, "TIGGE-Test", "TIGGE test", ""},
	{6, "S2S", "Sub-seasonal to seasonal prediction project (S2S)", ""},
	{7, "S2S-Test", "S2S test", ""},
	{8, "UERRA", "Uncertainties in Ensembles of Regional ReAnalyses project (UERRA)", ""},
	{9, "UERRA-Test", "UERRA test", ""},
}

var productionStatusRanges = []RangeEntry{
	{192, 254, "Local", "Reserved for local use"},
	{255, 255, "Missing", "Missing"},
}

// ProductionStatusTable is the WMO Code Table 1.3.
var ProductionStatusTable = NewRangeTable(productionStatusEntries, productionStatusRanges, "Unknown production status")

// GetProductionStatusName returns the name for a production status code.
func GetProductionStatusName(code int) string {
	return ProductionStatusTable.Name(code)
}

// WMO Code Table 1.4: Type of Data
//
// This table defines the type of processed data.

var dataTypeEntries = []*Entry{
	{0, "Analysis", "Analysis products", ""},
	{1, "Forecast", "Forecast products", ""},
	{2, "Analysis & Forecast", "Analysis and forecast products", ""},
	{3, "Control Forecast", "Control forecast products", ""},
	{4, "Perturbed Forecast", "Perturbed forecast products", ""},
	{5, "Control & Perturbed", "Control and perturbed forecast products", ""},
	{6, "Processed Satellite", "Processed satellite observations", ""},
	{7, "Processed Radar", "Processed radar observations", ""},
	{8, "Event Probability", "Event probability", ""},
}

var dataTypeRanges = []RangeEntry{
	{192, 254, "Local", "Reserved for local use"},
	{255, 255, "Missing", "Missing"},
}

// DataTypeTable is the WMO Code Table 1.4.
var DataTypeTable = NewRangeTable(dataTypeEntries, dataTypeRanges, "Unknown data type")

// GetDataTypeName returns the name for a data type code.
func GetDataTypeName(code int) string {
	return DataTypeTable.Name(code)
}
