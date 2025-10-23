package grib

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
	var messages []*Message
	var err error

	if config.sequential {
		if config.skipErrors {
			messages, err = ParseMessagesFromStreamSequentialSkipErrors(r)
		} else {
			messages, err = ParseMessagesFromStreamSequential(r)
		}
	} else if config.ctx != nil {
		messages, err = ParseMessagesFromStreamWithContext(config.ctx, r, config.workers)
	} else {
		messages, err = ParseMessagesFromStreamWithWorkers(r, config.workers)
	}

	if err != nil && !config.skipErrors {
		return nil, err
	}

	// Phase 1: Identify unique grids
	gridToMessages := make(map[gridKey][]*Message)
	uniqueGrids := make(map[gridKey]*Message) // Keep one example of each grid

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

	// Phase 2: Compute coordinates for unique grids in parallel
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

	// Phase 3: Convert messages to GRIB2 structs using cached coordinates (in parallel)
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
		g2.GridNi = int(msg.Section3.NumDataPoints) // Simplified
		g2.GridNj = 1                               // Simplified
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
			levelType := int(template.FirstSurfaceType)
			g2.Level = tables.GetLevelName(levelType)
			g2.LevelValue = float32(template.FirstSurfaceValue)

			// Format level description with value
			if template.FirstSurfaceValue != 0 {
				g2.Level = fmt.Sprintf("%s %g", g2.Level, g2.LevelValue)
			}
		}
	}

	return g2
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
		if val > 9e20 {
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
		if val > 9e20 {
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
