package squall

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/mmp/squall/grid"
	"github.com/mmp/squall/product"
	"github.com/mmp/squall/tables"
)

// GRIB2 missing value constants (matching wgrib2 implementation).
const (
	missingValue     = 9.999e20  // The actual value written for undefined data
	missingValueLow  = 9.9989e20 // Lower bound for range check (handles float precision)
	missingValueHigh = 9.9991e20 // Upper bound for range check (handles float precision)
)

// IsMissing returns true if the value represents missing/undefined GRIB2 data.
//
// GRIB2 uses 9.999e20 as a sentinel value for missing data points (when bitmap
// indicates undefined values). This function uses a range check (9.9989e20 to
// 9.9991e20) to account for floating-point precision and handle GRIB2 files
// from various sources, matching wgrib2's defensive approach.
func IsMissing(v float32) bool {
	return v >= missingValueLow && v <= missingValueHigh
}

// GRIB2 represents a single meteorological field from a GRIB2 message.
//
// This is the main public type returned by the Read function. It contains
// all the information needed to work with GRIB2 data: values, coordinates,
// and metadata.
type GRIB2 struct {
	// Data values in grid scan order
	Data []float32

	// Latitudes for each grid point (same length as Data)
	Latitudes []float32

	// Longitudes for each grid point (same length as Data)
	Longitudes []float32

	// Metadata from the message
	Discipline       string    // Meteorological, Hydrological, etc.
	Center           string    // Originating center (NCEP, ECMWF, etc.)
	ReferenceTime    time.Time // Reference time of the data
	ProductionStatus string    // Operational, Test, etc.
	DataType         string    // Forecast, Analysis, etc.

	// Parameter information
	Parameter ParameterID // WMO standard parameter identifier (D.C.P)

	// Level/surface information
	Level      string  // Type of level (isobaric, surface, etc.)
	LevelValue float32 // Value of the level (e.g., 500 for 500 hPa)

	// Grid information
	GridType  string // Lat/Lon, Gaussian, Lambert, etc.
	GridNi    int    // Number of points in i direction
	GridNj    int    // Number of points in j direction
	NumPoints int    // Total number of grid points

	// Raw message for advanced users
	message *Message
}

// Read parses GRIB2 messages from an io.ReadSeeker.
//
// This is the main entry point for the library. It parses all GRIB2 messages
// in the input stream and returns them as GRIB2 structs with decoded data and
// coordinates.
//
// Messages are parsed in parallel for performance. Individual messages are
// read into memory as needed, but the entire file is not loaded at once.
// Use ReadWithOptions to control parallelism or apply filters.
//
// Example:
//
//	file, _ := os.Open("forecast.grib2")
//	defer file.Close()
//	fields, err := grib.Read(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, field := range fields {
//	    fmt.Printf("%s at %s: %d points\n",
//	        field.Parameter, field.Level, field.NumPoints)
//	}
func Read(r io.ReadSeeker) ([]*GRIB2, error) {
	return ReadWithOptions(r)
}

// gridKey uniquely identifies a grid configuration for coordinate caching
type gridKey struct {
	templateNumber uint16
	numDataPoints  uint32
	nx, ny         uint32
}

// createGridKey creates a unique key for a grid
func createGridKey(msg *Message) (gridKey, bool) {
	if msg.Section3 == nil || msg.Section3.Grid == nil {
		return gridKey{}, false
	}

	var nx, ny uint32
	switch g := msg.Section3.Grid.(type) {
	case *grid.LambertConformalGrid:
		nx, ny = g.Nx, g.Ny
	case *grid.LatLonGrid:
		nx, ny = g.Ni, g.Nj
	case *grid.MercatorGrid:
		nx, ny = g.Ni, g.Nj
	case *grid.PolarStereographicGrid:
		nx, ny = g.Nx, g.Ny
	default:
		return gridKey{}, false
	}

	return gridKey{
		templateNumber: msg.Section3.TemplateNumber,
		numDataPoints:  msg.Section3.NumDataPoints,
		nx:             nx,
		ny:             ny,
	}, true
}

// coordinateCache stores pre-computed coordinates for unique grids
type coordinateCache struct {
	latitudes  []float32
	longitudes []float32
}

// ReadWithOptions parses GRIB2 messages with configuration options.
//
// Options can control parallelism, apply filters, or configure other
// behavior. See ReadOption for available options.
//
// Example:
//
//	file, _ := os.Open("forecast.grib2")
//	defer file.Close()
//	fields, err := grib.ReadWithOptions(file,
//	    WithWorkers(4),
//	    WithParameter("Temperature"),
//	)
func ReadWithOptions(r io.ReadSeeker, opts ...ReadOption) ([]*GRIB2, error) {
	// Apply options
	config := defaultReadConfig()
	for _, opt := range opts {
		opt(&config)
	}

	// Parse messages
	messages, err := parseMessages(r, config)
	if err != nil && !config.skipErrors {
		return nil, err
	}

	// Phase 1: Identify unique grids
	gridToMessages, uniqueGrids := identifyUniqueGrids(messages, config)

	// Phase 2: Compute coordinates for unique grids in parallel
	coordCache := computeCoordinatesForGrids(uniqueGrids)

	// Phase 3: Convert messages to GRIB2 structs using cached coordinates
	fields, err := convertMessagesToGRIB2(gridToMessages, coordCache, config)
	if err != nil && !config.skipErrors {
		return nil, err
	}

	return fields, nil
}

// parseMessages parses GRIB2 messages using the configured strategy
func parseMessages(r io.ReadSeeker, config readConfig) ([]*Message, error) {
	if config.sequential {
		if config.skipErrors {
			return ParseMessagesFromStreamSequentialSkipErrors(r)
		}
		return ParseMessagesFromStreamSequential(r)
	}

	if config.ctx != nil {
		return ParseMessagesFromStreamWithContext(config.ctx, r, config.workers)
	}

	return ParseMessagesFromStreamWithWorkers(r, config.workers)
}

// identifyUniqueGrids groups messages by grid and identifies unique grids
func identifyUniqueGrids(messages []*Message, config readConfig) (map[gridKey][]*Message, map[gridKey]*Message) {
	gridToMessages := make(map[gridKey][]*Message)
	uniqueGrids := make(map[gridKey]*Message)

	for _, msg := range messages {
		// Apply filters first
		if !config.filter(msg) {
			continue
		}

		key, ok := createGridKey(msg)
		if !ok {
			continue
		}

		gridToMessages[key] = append(gridToMessages[key], msg)
		if _, exists := uniqueGrids[key]; !exists {
			uniqueGrids[key] = msg
		}
	}

	return gridToMessages, uniqueGrids
}

// computeCoordinatesForGrids computes coordinates for each unique grid in parallel
func computeCoordinatesForGrids(uniqueGrids map[gridKey]*Message) map[gridKey]*coordinateCache {
	coordCache := make(map[gridKey]*coordinateCache)
	var cacheMutex sync.Mutex
	var wg sync.WaitGroup

	for key, exampleMsg := range uniqueGrids {
		wg.Add(1)
		go func(k gridKey, msg *Message) {
			defer wg.Done()

			lats, lons, err := msg.Coordinates()
			if err != nil {
				// Skip this grid if coordinates fail
				return
			}

			cacheMutex.Lock()
			coordCache[k] = &coordinateCache{
				latitudes:  lats,
				longitudes: lons,
			}
			cacheMutex.Unlock()
		}(key, exampleMsg)
	}
	wg.Wait()

	return coordCache
}

// convertMessagesToGRIB2 converts messages to GRIB2 structs using cached coordinates
func convertMessagesToGRIB2(gridToMessages map[gridKey][]*Message, coordCache map[gridKey]*coordinateCache, config readConfig) ([]*GRIB2, error) {
	type result struct {
		field *GRIB2
		err   error
		index int
	}

	// Count total messages to process
	totalMessages := 0
	for _, msgs := range gridToMessages {
		totalMessages += len(msgs)
	}

	resultChan := make(chan result, totalMessages)
	var decodeWg sync.WaitGroup

	// Limit parallelism to 2 * NumCPU to reduce memory pressure
	maxWorkers := runtime.NumCPU() * 2
	semaphore := make(chan struct{}, maxWorkers)

	// Process all messages with bounded parallelism
	messageIndex := 0
	for key, msgs := range gridToMessages {
		cache, ok := coordCache[key]
		if !ok {
			// Coordinates failed for this grid, skip these messages
			messageIndex += len(msgs)
			continue
		}

		for _, msg := range msgs {
			decodeWg.Add(1)
			idx := messageIndex
			messageIndex++

			// Acquire semaphore slot (blocks if maxWorkers goroutines are active)
			semaphore <- struct{}{}

			go func(m *Message, lats, lons []float32, i int) {
				defer decodeWg.Done()
				defer func() { <-semaphore }() // Release semaphore slot
				field, err := messageToGRIB2WithCoords(m, lats, lons)
				resultChan <- result{field: field, err: err, index: i}
			}(msg, cache.latitudes, cache.longitudes, idx)
		}
	}

	// Wait for all decoding to complete
	go func() {
		decodeWg.Wait()
		close(resultChan)
	}()

	// Collect results maintaining order
	results := make([]*result, totalMessages)
	for res := range resultChan {
		if res.err != nil {
			if !config.skipErrors {
				return nil, fmt.Errorf("failed to convert message: %w", res.err)
			}
			continue
		}
		results[res.index] = &res
	}

	// Build final slice in order, skipping nils (errors that were skipped)
	fields := make([]*GRIB2, 0, totalMessages)
	for _, res := range results {
		if res != nil && res.field != nil {
			fields = append(fields, res.field)
		}
	}

	return fields, nil
}

// messageToGRIB2WithCoords converts a Message to GRIB2 using pre-computed coordinates
func messageToGRIB2WithCoords(msg *Message, lats, lons []float32) (*GRIB2, error) {
	// Decode data
	data, err := msg.DecodeData()
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	// Build GRIB2 struct using provided coordinates
	g2 := &GRIB2{
		Data:       data,
		Latitudes:  lats,
		Longitudes: lons,
		NumPoints:  len(data),
		message:    msg,
	}

	return populateMetadata(g2, msg), nil
}

// populateMetadata extracts metadata from a Message into a GRIB2 struct
func populateMetadata(g2 *GRIB2, msg *Message) *GRIB2 {
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

		// Set proper grid dimensions based on grid type
		switch g := msg.Section3.Grid.(type) {
		case *grid.LambertConformalGrid:
			g2.GridNi = int(g.Nx)
			g2.GridNj = int(g.Ny)
		case *grid.LatLonGrid:
			g2.GridNi = int(g.Ni)
			g2.GridNj = int(g.Nj)
		case *grid.MercatorGrid:
			g2.GridNi = int(g.Ni)
			g2.GridNj = int(g.Nj)
		case *grid.PolarStereographicGrid:
			g2.GridNi = int(g.Nx)
			g2.GridNj = int(g.Ny)
		default:
			g2.GridNi = int(msg.Section3.NumDataPoints) // Fallback
			g2.GridNj = 1
		}
	}

	if msg.Section4 != nil && msg.Section4.Product != nil {
		discipline := msg.Section0.Discipline
		category := msg.Section4.Product.GetParameterCategory()
		number := msg.Section4.Product.GetParameterNumber()

		// Store WMO parameter identifier
		g2.Parameter = ParameterID{
			Discipline: discipline,
			Category:   category,
			Number:     number,
		}

		// Extract level information from product template
		if template, ok := msg.Section4.Product.(*product.Template40); ok {
			g2.Level = formatLevel(template)
			g2.LevelValue = float32(template.FirstSurfaceValue) / float32(scaleFactorToMultiplier(template.FirstSurfaceScaleFactor))
		}
	}

	return g2
}

// scaleFactorToMultiplier converts a GRIB2 scale factor to a multiplier.
// The scale factor is defined as: actual_value = scaled_value / 10^scale_factor
func scaleFactorToMultiplier(scaleFactor uint8) float64 {
	// Handle special case of 0
	if scaleFactor == 0 {
		return 1.0
	}

	// Calculate 10^scaleFactor
	multiplier := 1.0
	for i := uint8(0); i < scaleFactor; i++ {
		multiplier *= 10.0
	}
	return multiplier
}

// formatLevel formats a level description in wgrib2-compatible format.
func formatLevel(template *product.Template40) string {
	levelType := int(template.FirstSurfaceType)

	// Apply scale factors to get actual values
	value1 := float64(template.FirstSurfaceValue) / scaleFactorToMultiplier(template.FirstSurfaceScaleFactor)
	value2 := float64(template.SecondSurfaceValue) / scaleFactorToMultiplier(template.SecondSurfaceScaleFactor)

	// Special formatting for specific level types to match wgrib2
	switch levelType {
	case 1: // Surface
		return "surface"
	case 2: // Cloud base
		return "cloud base"
	case 3: // Cloud top
		return "cloud top"
	case 8: // Nominal top of atmosphere
		return "top of atmosphere"
	case 10: // Entire atmosphere (single layer)
		return "entire atmosphere"
	case 20: // Isothermal level
		if template.SecondSurfaceType == 20 && value2 > 0 {
			// Range between two isothermal levels
			return fmt.Sprintf("%.0f K level - %.0f K level", value1, value2)
		}
		return fmt.Sprintf("%.0f K level", value1)
	case 100: // Isobaric surface
		// Convert Pa to mb
		valueMb := value1 / 100.0
		valueMb2 := value2 / 100.0
		if template.SecondSurfaceType == 100 && valueMb2 > 0 {
			// Range (layer between two isobaric surfaces)
			return fmt.Sprintf("%.0f-%.0f mb above ground", valueMb, valueMb2)
		}
		// Format with appropriate precision (show decimal if needed)
		if valueMb == float64(int(valueMb)) {
			return fmt.Sprintf("%.0f mb", valueMb)
		}
		return fmt.Sprintf("%.1f mb", valueMb)
	case 101: // Mean sea level
		return "mean sea level"
	case 103: // Height above ground
		if template.SecondSurfaceType == 103 && value2 > 0 {
			// Range (layer)
			return fmt.Sprintf("%.0f-%.0f m above ground", value1, value2)
		}
		if value1 == 0 {
			return "surface"
		}
		return fmt.Sprintf("%.0f m above ground", value1)
	case 104: // Sigma level
		if template.SecondSurfaceType == 104 && value2 > 0 {
			// Range (sigma layer)
			return fmt.Sprintf("%.1f-%.1f sigma layer", value1, value2)
		}
		return fmt.Sprintf("%.1f sigma level", value1)
	case 106: // Depth below land surface
		if template.SecondSurfaceType == 106 && value2 > 0 {
			// Range (layer)
			if value1 == 0 {
				return fmt.Sprintf("%.2g m underground", value2)
			}
			return fmt.Sprintf("%.2g-%.2g m below ground", value1, value2)
		}
		if value1 == 0 {
			return "0 m underground"
		}
		return fmt.Sprintf("%.2g m below ground", value1)
	case 200: // Entire atmosphere (single layer)
		return "entire atmosphere (considered as a single layer)"
	}

	// Default: use table name
	levelName := tables.GetLevelName(levelType)

	// Add value if non-zero
	if value1 != 0 {
		return fmt.Sprintf("%s %g", levelName, value1)
	}

	return levelName
}

// String returns a human-readable summary of the field.
func (g *GRIB2) String() string {
	return fmt.Sprintf("GRIB2: %s from %s, %d points, ref time %s",
		g.Parameter, g.Center, g.NumPoints, g.ReferenceTime.Format(time.RFC3339))
}

// MinValue returns the minimum data value in the field.
func (g *GRIB2) MinValue() float32 {
	if len(g.Data) == 0 {
		return 0
	}

	minVal := g.Data[0]
	for _, val := range g.Data {
		// Skip missing values
		if IsMissing(val) {
			continue
		}
		if val < minVal {
			minVal = val
		}
	}
	return minVal
}

// MaxValue returns the maximum data value in the field.
func (g *GRIB2) MaxValue() float32 {
	if len(g.Data) == 0 {
		return 0
	}

	maxVal := g.Data[0]
	for _, val := range g.Data {
		// Skip missing values
		if IsMissing(val) {
			continue
		}
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal
}

// CountValid returns the number of valid (non-missing) data values.
func (g *GRIB2) CountValid() int {
	count := 0
	for _, val := range g.Data {
		if !IsMissing(val) {
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
