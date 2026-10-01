// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"os"
	"time"

	grib "github.com/mmp/squall"
	"github.com/mmp/squall/grid"
	"github.com/mmp/squall/product"
)

// ParseMgrib2 parses a GRIB2 file using squall (this implementation).
//
// Returns an array of FieldData structures in message order.
func ParseMgrib2(gribFile string) ([]*FieldData, error) {
	// Open GRIB2 file
	file, err := os.Open(gribFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Parse with squall (use sequential + skip errors for robustness)
	fields, err := grib.ReadWithOptions(file,
		grib.WithSequential(),
		grib.WithSkipErrors())
	if err != nil {
		return nil, fmt.Errorf("squall parse failed: %v", err)
	}

	// Convert to FieldData array (preserving message order)
	fieldArray := make([]*FieldData, 0, len(fields))

	for _, field := range fields {
		// The wgrib2 inventory gives verification times for forecasts at a
		// point in time ("1 hour fcst"); compute the same here.
		verTime := field.ReferenceTime
		if msg := field.GetMessage(); msg != nil && msg.Section4 != nil {
			if t, ok := msg.Section4.Product.(*product.Template40); ok {
				if d, ok := forecastDuration(t.TimeRangeUnit, t.ForecastTime); ok {
					verTime = verTime.Add(d)
				}
			}
		}

		// Use short name for comparison with wgrib2 (if available)
		fieldName := field.ShortName()
		if fieldName == "" {
			p := field.Parameter
			fieldName = fmt.Sprintf("unknown parameter %d.%d.%d", p.Discipline, p.Category, p.Number)
		}

		// wgrib2 gives coordinates only in we:sn order, so put squall's
		// values, which are in the grid's scan order, in that order too.
		toWESN := func(v []float32) []float32 { return v }
		if mode, ok := scanningMode(field); ok && len(field.Data) == field.GridNi*field.GridNj {
			toWESN = func(v []float32) []float32 {
				return reorderWESN(v, field.GridNi, field.GridNj, mode)
			}
		}

		fd := &FieldData{
			RefTime:    field.ReferenceTime,
			VerTime:    verTime,
			Field:      fieldName,
			Level:      field.Level,
			Latitudes:  toWESN(field.Latitudes),
			Longitudes: toWESN(field.Longitudes),
			Values:     toWESN(field.Data),
			Source:     "squall",
		}

		fieldArray = append(fieldArray, fd)
	}

	return fieldArray, nil
}

// scanningMode returns the scanning mode flags (Table 3.4) of the field's grid.
func scanningMode(field *grib.GRIB2) (uint8, bool) {
	msg := field.GetMessage()
	if msg == nil || msg.Section3 == nil {
		return 0, false
	}
	switch g := msg.Section3.Grid.(type) {
	case *grid.LatLonGrid:
		return g.ScanningMode, true
	case *grid.LambertConformalGrid:
		return g.ScanningMode, true
	case *grid.MercatorGrid:
		return g.ScanningMode, true
	case *grid.PolarStereographicGrid:
		return g.ScanningMode, true
	}
	return 0, false
}

// reorderWESN reorders values from a grid's scan order to west-to-east,
// south-to-north order.
func reorderWESN(v []float32, ni, nj int, mode uint8) []float32 {
	iNegative := mode&0x80 != 0
	jPositive := mode&0x40 != 0
	consecutive := mode&0x20 == 0
	alternating := mode&0x10 != 0 // boustrophedonic: odd rows are reversed
	if !iNegative && jPositive && consecutive && !alternating {
		return v
	}
	out := make([]float32, len(v))
	for idx, x := range v {
		i, j := idx%ni, idx/ni
		if !consecutive {
			i, j = idx/nj, idx%nj
		}
		if alternating && j%2 == 1 {
			i = ni - 1 - i
		}
		if iNegative {
			i = ni - 1 - i
		}
		if !jPositive {
			j = nj - 1 - j
		}
		out[j*ni+i] = x
	}
	return out
}

// forecastDuration converts a forecast time to a duration given its unit
// (Code Table 4.4).
func forecastDuration(unit uint8, value uint32) (time.Duration, bool) {
	units := map[uint8]time.Duration{
		0:  time.Minute,
		1:  time.Hour,
		2:  24 * time.Hour,
		10: 3 * time.Hour,
		11: 6 * time.Hour,
		12: 12 * time.Hour,
		13: time.Second,
	}
	u, ok := units[unit]
	return time.Duration(value) * u, ok
}
