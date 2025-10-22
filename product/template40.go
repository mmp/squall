package product

import (
	"fmt"

	"github.com/mmp/mgrib2/internal"
)

// Template40 represents Product Definition Template 4.0:
// Analysis or forecast at a horizontal level or in a horizontal layer
// at a point in time.
//
// This is the most common product template, used for standard forecast
// and analysis data.
type Template40 struct {
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
}

// ParseTemplate40 parses Product Definition Template 4.0.
//
// The template data should be 25 bytes for Template 4.0.
func ParseTemplate40(data []byte) (*Template40, error) {
	if len(data) < 25 {
		return nil, fmt.Errorf("template 4.0 requires at least 25 bytes, got %d", len(data))
	}

	r := internal.NewReader(data)

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

	return &Template40{
		ParameterCategory:        paramCategory,
		ParameterNumber:          paramNumber,
		GeneratingProcess:        generatingProcess,
		BackgroundProcess:        backgroundProcess,
		ForecastProcess:          forecastProcess,
		HoursAfterCutoff:         hoursAfterCutoff,
		MinutesAfterCutoff:       minutesAfterCutoff,
		TimeRangeUnit:            timeRangeUnit,
		ForecastTime:             forecastTime,
		FirstSurfaceType:         firstSurfaceType,
		FirstSurfaceScaleFactor:  firstSurfaceScaleFactor,
		FirstSurfaceValue:        firstSurfaceValue,
		SecondSurfaceType:        secondSurfaceType,
		SecondSurfaceScaleFactor: secondSurfaceScaleFactor,
		SecondSurfaceValue:       secondSurfaceValue,
	}, nil
}

// TemplateNumber returns 0 for Template 4.0.
func (t *Template40) TemplateNumber() int {
	return 0
}

// GetParameterCategory returns the parameter category code.
func (t *Template40) GetParameterCategory() uint8 {
	return t.ParameterCategory
}

// GetParameterNumber returns the parameter number code.
func (t *Template40) GetParameterNumber() uint8 {
	return t.ParameterNumber
}

// String returns a human-readable description.
func (t *Template40) String() string {
	return fmt.Sprintf("Template 4.0: Category=%d, Parameter=%d, Surface Type=%d",
		t.ParameterCategory, t.ParameterNumber, t.FirstSurfaceType)
}

// FirstSurfaceValueScaled returns the scaled value of the first fixed surface.
// This applies the scale factor to get the actual value.
func (t *Template40) FirstSurfaceValueScaled() float64 {
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
func (t *Template40) SecondSurfaceValueScaled() float64 {
	if t.SecondSurfaceScaleFactor == 0 {
		return float64(t.SecondSurfaceValue)
	}
	divisor := 1.0
	for i := uint8(0); i < t.SecondSurfaceScaleFactor; i++ {
		divisor *= 10.0
	}
	return float64(t.SecondSurfaceValue) / divisor
}
