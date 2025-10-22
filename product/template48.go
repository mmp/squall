package product

import (
	"fmt"

	"github.com/mmp/mgrib2/internal"
)

// Template48 represents Product Definition Template 4.8:
// Average, accumulation, extreme values or other statistically processed values
// at a horizontal level or in a horizontal layer in a continuous or non-continuous
// time interval.
//
// This template extends Template 4.0 with statistical processing information
// for time-averaged, accumulated, or extreme value products.
type Template48 struct {
	// Fields from Template 4.0 (octets 10-34)
	ParameterCategory        uint8  // Parameter category (Table 4.1)
	ParameterNumber          uint8  // Parameter number (Table 4.2)
	GeneratingProcess        uint8  // Type of generating process (Table 4.3)
	BackgroundProcess        uint8  // Background generating process
	ForecastProcess          uint8  // Analysis or forecast generating process
	HoursAfterCutoff         uint16 // Hours after data cutoff
	MinutesAfterCutoff       uint8  // Minutes after data cutoff
	TimeRangeUnit            uint8  // Indicator of unit of time range (Table 4.4)
	ForecastTime             uint32 // Forecast time in units defined by TimeRangeUnit
	FirstSurfaceType         uint8  // Type of first fixed surface (Table 4.5)
	FirstSurfaceScaleFactor  uint8  // Scale factor of first fixed surface
	FirstSurfaceValue        uint32 // Scaled value of first fixed surface
	SecondSurfaceType        uint8  // Type of second fixed surface (Table 4.5)
	SecondSurfaceScaleFactor uint8  // Scale factor of second fixed surface
	SecondSurfaceValue       uint32 // Scaled value of second fixed surface

	// Template 4.8 specific fields (octets 35-58)
	EndYear                    uint16 // Year of end of overall time interval
	EndMonth                   uint8  // Month of end of overall time interval
	EndDay                     uint8  // Day of end of overall time interval
	EndHour                    uint8  // Hour of end of overall time interval
	EndMinute                  uint8  // Minute of end of overall time interval
	EndSecond                  uint8  // Second of end of overall time interval
	NumberOfTimeRanges         uint8  // Number of time range specifications
	NumberMissingInStatProcess uint32 // Number of missing values in statistical process

	// Statistical processing specifications (12 bytes per time range)
	TimeRanges []StatisticalTimeRange
}

// StatisticalTimeRange describes one statistical processing specification.
// Each specification is 12 bytes (octets 47-58 for first range, then repeated).
type StatisticalTimeRange struct {
	StatisticalProcess uint8  // Type of statistical processing (Table 4.10)
	TimeIncrementType  uint8  // Type of time increment (Table 4.11)
	TimeRangeUnit      uint8  // Unit of time range (Table 4.4)
	TimeRangeLength    uint32 // Length of time range
	TimeIncrementUnit  uint8  // Unit of time increment (Table 4.4)
	TimeIncrement      uint32 // Time increment between successive fields
}

// ParseTemplate48 parses Product Definition Template 4.8.
//
// The template data should be at least 37 bytes for Template 4.8 base fields.
// With n time ranges: 37 + 12*n bytes.
// (25 bytes from Template 4.0 + 12 bytes Template 4.8 specific + 12*n bytes for time ranges)
func ParseTemplate48(data []byte) (*Template48, error) {
	if len(data) < 37 {
		return nil, fmt.Errorf("template 4.8 requires at least 37 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

	// Read Template 4.0 fields (octets 10-34, 25 bytes)
	paramCategory, _ := r.Uint8()
	paramNumber, _ := r.Uint8()
	generatingProcess, _ := r.Uint8()
	backgroundProcess, _ := r.Uint8()
	forecastProcess, _ := r.Uint8()
	hoursAfterCutoff, _ := r.Uint16()
	minutesAfterCutoff, _ := r.Uint8()
	timeRangeUnit, _ := r.Uint8()
	forecastTime, _ := r.Uint32()
	firstSurfaceType, _ := r.Uint8()
	firstSurfaceScaleFactor, _ := r.Uint8()
	firstSurfaceValue, _ := r.Uint32()
	secondSurfaceType, _ := r.Uint8()
	secondSurfaceScaleFactor, _ := r.Uint8()
	secondSurfaceValue, _ := r.Uint32()

	// Read Template 4.8 specific fields (octets 35-46, 12 bytes)
	endYear, _ := r.Uint16()
	endMonth, _ := r.Uint8()
	endDay, _ := r.Uint8()
	endHour, _ := r.Uint8()
	endMinute, _ := r.Uint8()
	endSecond, _ := r.Uint8()
	numTimeRanges, _ := r.Uint8()
	numMissing, _ := r.Uint32()

	// Verify we have enough data for the time ranges
	expectedLen := 37 + int(numTimeRanges)*12
	if len(data) < expectedLen {
		return nil, fmt.Errorf("template 4.8 with %d time ranges requires %d bytes, got %d",
			numTimeRanges, expectedLen, len(data))
	}

	// Read statistical processing specifications (12 bytes each)
	timeRanges := make([]StatisticalTimeRange, numTimeRanges)
	for i := uint8(0); i < numTimeRanges; i++ {
		statProcess, _ := r.Uint8()
		timeIncrType, _ := r.Uint8()
		timeRangeUnit, _ := r.Uint8()
		timeRangeLen, _ := r.Uint32()
		timeIncrUnit, _ := r.Uint8()
		timeIncr, _ := r.Uint32()

		timeRanges[i] = StatisticalTimeRange{
			StatisticalProcess: statProcess,
			TimeIncrementType:  timeIncrType,
			TimeRangeUnit:      timeRangeUnit,
			TimeRangeLength:    timeRangeLen,
			TimeIncrementUnit:  timeIncrUnit,
			TimeIncrement:      timeIncr,
		}
	}

	return &Template48{
		ParameterCategory:          paramCategory,
		ParameterNumber:            paramNumber,
		GeneratingProcess:          generatingProcess,
		BackgroundProcess:          backgroundProcess,
		ForecastProcess:            forecastProcess,
		HoursAfterCutoff:           hoursAfterCutoff,
		MinutesAfterCutoff:         minutesAfterCutoff,
		TimeRangeUnit:              timeRangeUnit,
		ForecastTime:               forecastTime,
		FirstSurfaceType:           firstSurfaceType,
		FirstSurfaceScaleFactor:    firstSurfaceScaleFactor,
		FirstSurfaceValue:          firstSurfaceValue,
		SecondSurfaceType:          secondSurfaceType,
		SecondSurfaceScaleFactor:   secondSurfaceScaleFactor,
		SecondSurfaceValue:         secondSurfaceValue,
		EndYear:                    endYear,
		EndMonth:                   endMonth,
		EndDay:                     endDay,
		EndHour:                    endHour,
		EndMinute:                  endMinute,
		EndSecond:                  endSecond,
		NumberOfTimeRanges:         numTimeRanges,
		NumberMissingInStatProcess: numMissing,
		TimeRanges:                 timeRanges,
	}, nil
}

// TemplateNumber returns 8 for Template 4.8.
func (t *Template48) TemplateNumber() int {
	return 8
}

// GetParameterCategory returns the parameter category code.
func (t *Template48) GetParameterCategory() uint8 {
	return t.ParameterCategory
}

// GetParameterNumber returns the parameter number code.
func (t *Template48) GetParameterNumber() uint8 {
	return t.ParameterNumber
}

// String returns a human-readable description.
func (t *Template48) String() string {
	return fmt.Sprintf("Template 4.8: Category=%d, Parameter=%d, Surface Type=%d, Time Ranges=%d",
		t.ParameterCategory, t.ParameterNumber, t.FirstSurfaceType, t.NumberOfTimeRanges)
}

// FirstSurfaceValueScaled returns the scaled value of the first fixed surface.
func (t *Template48) FirstSurfaceValueScaled() float64 {
	if t.FirstSurfaceScaleFactor == 0 {
		return float64(t.FirstSurfaceValue)
	}
	divisor := 1.0
	for i := uint8(0); i < t.FirstSurfaceScaleFactor; i++ {
		divisor *= 10.0
	}
	return float64(t.FirstSurfaceValue) / divisor
}

// SecondSurfaceValueScaled returns the scaled value of the second fixed surface.
func (t *Template48) SecondSurfaceValueScaled() float64 {
	if t.SecondSurfaceScaleFactor == 0 {
		return float64(t.SecondSurfaceValue)
	}
	divisor := 1.0
	for i := uint8(0); i < t.SecondSurfaceScaleFactor; i++ {
		divisor *= 10.0
	}
	return float64(t.SecondSurfaceValue) / divisor
}
