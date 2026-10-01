// Package testutil provides utilities for testing GRIB2 parsing against reference implementations.
package testutil

import (
	"fmt"
	"os"

	grib "github.com/mmp/squall"
	"github.com/mmp/squall/grid"
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
		// TODO: Calculate verification time from forecast time
		// For now, use reference time for both
		verTime := field.ReferenceTime

		// Use short name for comparison with wgrib2 (if available)
		fieldName := field.Parameter.ShortName()
		if fieldName == "" {
			// Fall back to full name if no short name exists
			fieldName = field.Parameter.String()
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
	if !iNegative && jPositive && consecutive {
		return v
	}
	out := make([]float32, len(v))
	for idx, x := range v {
		i, j := idx%ni, idx/ni
		if !consecutive {
			i, j = idx/nj, idx%nj
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
