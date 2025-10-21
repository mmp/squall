package mgrib2

import (
	"fmt"
	"time"

	"github.com/mmp/mgrib2/product"
	"github.com/mmp/mgrib2/tables"
)

// GRIB2 represents a single meteorological field from a GRIB2 message.
//
// This is the main public type returned by the Read function. It contains
// all the information needed to work with GRIB2 data: values, coordinates,
// and metadata.
type GRIB2 struct {
	// Data values in grid scan order
	Data []float64

	// Latitudes for each grid point (same length as Data)
	Latitudes []float64

	// Longitudes for each grid point (same length as Data)
	Longitudes []float64

	// Metadata from the message
	Discipline       string    // Meteorological, Hydrological, etc.
	Center           string    // Originating center (NCEP, ECMWF, etc.)
	ReferenceTime    time.Time // Reference time of the data
	ProductionStatus string    // Operational, Test, etc.
	DataType         string    // Forecast, Analysis, etc.

	// Parameter information
	ParameterCategory string // Temperature, Moisture, etc.
	ParameterNumber   string // Specific parameter within category
	ParameterName     string // Human-readable parameter name

	// Level/surface information
	Level      string  // Type of level (isobaric, surface, etc.)
	LevelValue float64 // Value of the level (e.g., 500 for 500 hPa)

	// Grid information
	GridType   string // Lat/Lon, Gaussian, Lambert, etc.
	GridNi     int    // Number of points in i direction
	GridNj     int    // Number of points in j direction
	NumPoints  int    // Total number of grid points

	// Raw message for advanced users
	message *Message
}

// Read parses GRIB2 messages from a byte slice.
//
// This is the main entry point for the library. It parses all GRIB2 messages
// in the input data and returns them as GRIB2 structs with decoded data and
// coordinates.
//
// Messages are parsed in parallel for performance. Use ReadWithOptions to
// control parallelism or apply filters.
//
// Example:
//
//	data, _ := os.ReadFile("forecast.grib2")
//	fields, err := mgrib2.Read(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, field := range fields {
//	    fmt.Printf("%s at %s: %d points\n",
//	        field.ParameterName, field.Level, field.NumPoints)
//	}
func Read(data []byte) ([]*GRIB2, error) {
	return ReadWithOptions(data)
}

// ReadWithOptions parses GRIB2 messages with configuration options.
//
// Options can control parallelism, apply filters, or configure other
// behavior. See ReadOption for available options.
//
// Example:
//
//	fields, err := mgrib2.ReadWithOptions(data,
//	    WithWorkers(4),
//	    WithParameter("Temperature"),
//	)
func ReadWithOptions(data []byte, opts ...ReadOption) ([]*GRIB2, error) {
	// Apply options
	config := defaultReadConfig()
	for _, opt := range opts {
		opt(&config)
	}

	// Parse messages
	var messages []*Message
	var err error

	if config.sequential {
		if config.skipErrors {
			messages, err = ParseMessagesSequentialSkipErrors(data)
		} else {
			messages, err = ParseMessagesSequential(data)
		}
	} else if config.ctx != nil {
		messages, err = ParseMessagesWithContext(config.ctx, data, config.workers)
	} else {
		messages, err = ParseMessagesWithWorkers(data, config.workers)
	}

	if err != nil && !config.skipErrors {
		return nil, err
	}

	// Convert to GRIB2 structs
	var fields []*GRIB2
	for _, msg := range messages {
		// Apply filters
		if !config.filter(msg) {
			continue
		}

		field, err := messageToGRIB2(msg)
		if err != nil {
			if config.skipErrors {
				continue
			}
			return nil, fmt.Errorf("failed to convert message: %w", err)
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// messageToGRIB2 converts an internal Message to a public GRIB2 struct.
func messageToGRIB2(msg *Message) (*GRIB2, error) {
	// Decode data
	data, err := msg.DecodeData()
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	// Get coordinates
	lats, lons, err := msg.Coordinates()
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates: %w", err)
	}

	// Build GRIB2 struct
	g2 := &GRIB2{
		Data:       data,
		Latitudes:  lats,
		Longitudes: lons,
		NumPoints:  len(data),
		message:    msg,
	}

	// Extract metadata
	if msg.Section0 != nil {
		g2.Discipline = msg.Section0.DisciplineName()
	}

	if msg.Section1 != nil {
		g2.Center = msg.Section1.CenterName()
		g2.ReferenceTime = msg.Section1.ReferenceTime
		g2.ProductionStatus = msg.Section1.ProductionStatusName()
		g2.DataType = msg.Section1.DataTypeName()
	}

	if msg.Section3 != nil && msg.Section3.Grid != nil {
		g2.GridType = fmt.Sprintf("Template %d", msg.Section3.Grid.TemplateNumber())
		g2.GridNi = int(msg.Section3.NumDataPoints) // Simplified
		g2.GridNj = 1                                // Simplified
	}

	if msg.Section4 != nil && msg.Section4.Product != nil {
		discipline := int(msg.Section0.Discipline)
		category := msg.Section4.Product.GetParameterCategory()
		number := msg.Section4.Product.GetParameterNumber()

		// Use table lookups for human-readable names
		g2.ParameterCategory = tables.GetParameterCategoryName(discipline, int(category))
		g2.ParameterNumber = fmt.Sprintf("%d", number)
		g2.ParameterName = tables.GetParameterName(discipline, int(category), int(number))

		// Extract level information from product template
		if template, ok := msg.Section4.Product.(*product.Template40); ok {
			levelType := int(template.FirstSurfaceType)
			g2.Level = tables.GetLevelName(levelType)
			g2.LevelValue = float64(template.FirstSurfaceValue)

			// Format level description with value
			if template.FirstSurfaceValue != 0 {
				g2.Level = fmt.Sprintf("%s %g", g2.Level, g2.LevelValue)
			}
		}
	}

	return g2, nil
}

// String returns a human-readable summary of the field.
func (g *GRIB2) String() string {
	return fmt.Sprintf("GRIB2: %s from %s, %d points, ref time %s",
		g.ParameterName, g.Center, g.NumPoints, g.ReferenceTime.Format(time.RFC3339))
}

// MinValue returns the minimum data value in the field.
func (g *GRIB2) MinValue() float64 {
	if len(g.Data) == 0 {
		return 0
	}

	min := g.Data[0]
	for _, val := range g.Data {
		// Skip missing values
		if val > 9e20 {
			continue
		}
		if val < min {
			min = val
		}
	}
	return min
}

// MaxValue returns the maximum data value in the field.
func (g *GRIB2) MaxValue() float64 {
	if len(g.Data) == 0 {
		return 0
	}

	max := g.Data[0]
	for _, val := range g.Data {
		// Skip missing values
		if val > 9e20 {
			continue
		}
		if val > max {
			max = val
		}
	}
	return max
}

// CountValid returns the number of valid (non-missing) data values.
func (g *GRIB2) CountValid() int {
	count := 0
	for _, val := range g.Data {
		if val < 9e20 { // Not a missing value
			count++
		}
	}
	return count
}

// GetMessage returns the underlying parsed message for advanced users.
//
// This provides access to the raw section data and allows for custom
// processing beyond what the GRIB2 struct provides.
func (g *GRIB2) GetMessage() *Message {
	return g.message
}
