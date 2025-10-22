package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	grib "github.com/mmp/squall"
	"github.com/mmp/squall/grid"
)

var (
	listFlag    = flag.Bool("list", false, "List all records with basic info")
	detailFlag  = flag.Bool("detail", false, "Show detailed information for all records")
	recordFlag  = flag.Int("record", -1, "Show detailed information for specific record (0-based)")
	valuesFlag  = flag.Bool("values", false, "Print data values for the record(s)")
	statsFlag   = flag.Bool("stats", false, "Show statistics (min/max/count) for each record")
	bboxFlag    = flag.Bool("bbox", false, "Show bounding box and grid information")
	summaryFlag = flag.Bool("summary", true, "Show file summary (default)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <grib2-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examine GRIB2 files and display information about their contents.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s file.grib2              # Show summary\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -list file.grib2        # List all records\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s file.grib2 -list        # Flags can appear anywhere\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -detail file.grib2      # Show details for all records\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -record 0 file.grib2    # Show details for first record\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -stats file.grib2       # Show statistics for all records\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -bbox file.grib2        # Show bounding box information\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -record 0 -values file.grib2  # Show data values for record 0\n", os.Args[0])
	}

	// Manually parse to allow flags anywhere and find non-flag argument as filename
	filename := ""
	args := []string{}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			args = append(args, arg)
			// Check if this flag takes a value (only -record does)
			if arg == "-record" && i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				i++
				args = append(args, os.Args[i])
			}
		} else {
			if filename != "" {
				fmt.Fprintf(os.Stderr, "Error: multiple filenames specified: %s and %s\n", filename, arg)
				os.Exit(1)
			}
			filename = arg
		}
	}

	// Parse the collected flags
	flag.CommandLine.Parse(args)

	if filename == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Open and read file
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fields, err := grib.Read(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading GRIB2 file: %v\n", err)
		os.Exit(1)
	}

	if len(fields) == 0 {
		fmt.Println("No GRIB2 messages found in file")
		return
	}

	// Determine what to display
	if *recordFlag >= 0 {
		if *recordFlag >= len(fields) {
			fmt.Fprintf(os.Stderr, "Record %d does not exist (file has %d records, numbered 0-%d)\n",
				*recordFlag, len(fields), len(fields)-1)
			os.Exit(1)
		}
		showRecordDetail(fields[*recordFlag], *recordFlag, *valuesFlag)
	} else if *listFlag {
		showList(fields)
	} else if *detailFlag {
		showAllDetails(fields, *valuesFlag)
	} else if *statsFlag {
		showStats(fields)
	} else if *bboxFlag {
		showBoundingBoxes(fields)
	} else if *summaryFlag {
		showSummary(filename, fields)
	}
}

func showSummary(filename string, fields []*grib.GRIB2) {
	fmt.Printf("File: %s\n", filename)
	fmt.Printf("Total records: %d\n\n", len(fields))

	// Get file info
	if info, err := os.Stat(filename); err == nil {
		fmt.Printf("File size: %s\n\n", formatBytes(uint64(info.Size())))
	}

	// Collect unique attributes
	disciplines := make(map[string]bool)
	centers := make(map[string]bool)
	paramTypes := make(map[string]bool)
	levels := make(map[string]bool)
	gridTypes := make(map[string]bool)
	refTimes := make(map[string]bool)

	for _, f := range fields {
		disciplines[f.Discipline] = true
		centers[f.Center] = true
		paramTypes[fmt.Sprintf("%s / %s", f.ParameterCategory, f.ParameterName)] = true
		levels[f.Level] = true
		gridTypes[f.GridType] = true
		refTimes[f.ReferenceTime.Format("2006-01-02 15:04 MST")] = true
	}

	fmt.Printf("Disciplines: %s\n", strings.Join(keys(disciplines), ", "))
	fmt.Printf("Centers: %s\n", strings.Join(keys(centers), ", "))
	fmt.Printf("Reference times: %s\n", strings.Join(keys(refTimes), ", "))
	fmt.Printf("Grid types: %s\n", strings.Join(keys(gridTypes), ", "))
	fmt.Printf("\nParameter types present:\n")
	for _, p := range keys(paramTypes) {
		count := 0
		for _, f := range fields {
			if fmt.Sprintf("%s / %s", f.ParameterCategory, f.ParameterName) == p {
				count++
			}
		}
		fmt.Printf("  %s (%d records)\n", p, count)
	}

	fmt.Printf("\nLevels present:\n")
	for _, l := range keys(levels) {
		count := 0
		for _, f := range fields {
			if f.Level == l {
				count++
			}
		}
		fmt.Printf("  %s (%d records)\n", l, count)
	}

	// Show bounding box for first record
	if len(fields) > 0 {
		fmt.Printf("\nGrid information (from first record):\n")
		showGridInfo(fields[0])
	}

	fmt.Printf("\nUse -list to see all records, -detail for full information\n")
}

func showList(fields []*grib.GRIB2) {
	fmt.Printf("%-5s %-40s %-25s %-15s %s\n", "Rec#", "Parameter", "Level", "Grid", "Ref Time")
	fmt.Println(strings.Repeat("-", 120))

	for i, f := range fields {
		paramName := f.ParameterName
		if len(paramName) > 40 {
			paramName = paramName[:37] + "..."
		}

		levelStr := f.Level
		if f.LevelValue != 0 {
			levelStr = fmt.Sprintf("%s (%.1f)", f.Level, f.LevelValue)
		}
		if len(levelStr) > 25 {
			levelStr = levelStr[:22] + "..."
		}

		gridStr := fmt.Sprintf("%s %dx%d", f.GridType, f.GridNi, f.GridNj)
		if len(gridStr) > 15 {
			gridStr = gridStr[:12] + "..."
		}

		fmt.Printf("%-5d %-40s %-25s %-15s %s\n",
			i,
			paramName,
			levelStr,
			gridStr,
			f.ReferenceTime.Format("2006-01-02 15:04"))
	}
}

func showAllDetails(fields []*grib.GRIB2, showValues bool) {
	for i, f := range fields {
		showRecordDetail(f, i, showValues)
		if i < len(fields)-1 {
			fmt.Println(strings.Repeat("=", 80))
		}
	}
}

func showRecordDetail(f *grib.GRIB2, recordNum int, showValues bool) {
	fmt.Printf("Record #%d\n", recordNum)
	fmt.Println(strings.Repeat("-", 80))

	// Basic identification
	fmt.Printf("Discipline:         %s\n", f.Discipline)
	fmt.Printf("Center:             %s\n", f.Center)
	fmt.Printf("Production Status:  %s\n", f.ProductionStatus)
	fmt.Printf("Data Type:          %s\n", f.DataType)
	fmt.Printf("Reference Time:     %s\n", f.ReferenceTime.Format("2006-01-02 15:04:05 MST"))

	// Parameter information
	fmt.Printf("\nParameter:\n")
	fmt.Printf("  Category:         %s\n", f.ParameterCategory)
	fmt.Printf("  Number:           %s\n", f.ParameterNumber)
	fmt.Printf("  Name:             %s\n", f.ParameterName)

	// Level information
	fmt.Printf("\nLevel:\n")
	fmt.Printf("  Type:             %s\n", f.Level)
	if f.LevelValue != 0 {
		fmt.Printf("  Value:            %.2f\n", f.LevelValue)
	}

	// Grid information
	fmt.Printf("\nGrid:\n")
	showGridInfo(f)

	// Data statistics
	fmt.Printf("\nData:\n")
	fmt.Printf("  Total points:     %d\n", f.NumPoints)

	min, max := getMinMax(f.Data)
	validCount := countValid(f.Data)

	fmt.Printf("  Valid points:     %d\n", validCount)
	fmt.Printf("  Missing points:   %d\n", f.NumPoints-validCount)

	if validCount > 0 {
		fmt.Printf("  Min value:        %.6f\n", min)
		fmt.Printf("  Max value:        %.6f\n", max)
		fmt.Printf("  Range:            %.6f\n", max-min)
	}

	// Show values if requested
	if showValues {
		fmt.Printf("\nData Values:\n")
		printDataValues(f.Data, f.GridNi)
	}
}

func showStats(fields []*grib.GRIB2) {
	fmt.Printf("%-5s %-40s %-15s %12s %12s %12s\n",
		"Rec#", "Parameter", "Level", "Min", "Max", "Valid/Total")
	fmt.Println(strings.Repeat("-", 100))

	for i, f := range fields {
		paramName := f.ParameterName
		if len(paramName) > 40 {
			paramName = paramName[:37] + "..."
		}

		levelStr := f.Level
		if f.LevelValue != 0 {
			levelStr = fmt.Sprintf("%s %.0f", f.Level, f.LevelValue)
		}
		if len(levelStr) > 15 {
			levelStr = levelStr[:12] + "..."
		}

		min, max := getMinMax(f.Data)
		validCount := countValid(f.Data)

		fmt.Printf("%-5d %-40s %-15s %12.4f %12.4f %6d/%-6d\n",
			i,
			paramName,
			levelStr,
			min,
			max,
			validCount,
			f.NumPoints)
	}
}

func showBoundingBoxes(fields []*grib.GRIB2) {
	// Group by unique grids
	type gridKey struct {
		gridType string
		ni, nj   int
	}

	grids := make(map[gridKey]*grib.GRIB2)
	for _, f := range fields {
		key := gridKey{f.GridType, f.GridNi, f.GridNj}
		if _, exists := grids[key]; !exists {
			grids[key] = f
		}
	}

	fmt.Printf("Found %d unique grid(s) in file:\n\n", len(grids))

	i := 1
	for _, f := range grids {
		fmt.Printf("Grid #%d: %s (%d x %d = %d points)\n", i, f.GridType, f.GridNi, f.GridNj, f.NumPoints)
		showGridInfo(f)
		fmt.Println()
		i++
	}
}

func showGridInfo(f *grib.GRIB2) {
	fmt.Printf("  Type:             %s\n", f.GridType)
	fmt.Printf("  Dimensions:       %d x %d\n", f.GridNi, f.GridNj)
	fmt.Printf("  Total points:     %d\n", f.NumPoints)

	// Get bounding box from lat/lon arrays
	if len(f.Latitudes) > 0 && len(f.Longitudes) > 0 {
		minLat, maxLat := getMinMax(f.Latitudes)
		minLon, maxLon := getMinMax(f.Longitudes)

		fmt.Printf("  Latitude range:   %.4f to %.4f\n", minLat, maxLat)
		fmt.Printf("  Longitude range:  %.4f to %.4f\n", minLon, maxLon)

		// Try to get more specific grid info
		if msg := f.GetMessage(); msg != nil && msg.Section3 != nil {
			switch g := msg.Section3.Grid.(type) {
			case *grid.LatLonGrid:
				lat1, lon1 := g.FirstGridPoint()
				lat2, lon2 := g.LastGridPoint()
				di, dj := g.Increment()
				fmt.Printf("  First point:      %.4f N, %.4f E\n", lat1, lon1)
				fmt.Printf("  Last point:       %.4f N, %.4f E\n", lat2, lon2)
				fmt.Printf("  Grid spacing:     %.4f x %.4f degrees\n", di, dj)

			case *grid.LambertConformalGrid:
				fmt.Printf("  First point:      %.4f N, %.4f E\n",
					float64(g.La1)/1e6, float64(g.Lo1)/1e6)
				fmt.Printf("  Grid spacing:     %d x %d meters\n", g.Dx, g.Dy)
				fmt.Printf("  Ref latitude:     %.4f N\n", float64(g.LaD)/1e6)
				fmt.Printf("  Ref longitude:    %.4f E\n", float64(g.LoV)/1e6)
				fmt.Printf("  Std parallels:    %.4f N, %.4f N\n",
					float64(g.Latin1)/1e6, float64(g.Latin2)/1e6)
			}
		}
	}
}

func printDataValues(data []float32, ni int) {
	const maxRowsToPrint = 20
	const maxColsToPrint = 10

	nj := len(data) / ni
	if ni == 0 {
		ni = len(data)
		nj = 1
	}

	rowsToPrint := nj
	if rowsToPrint > maxRowsToPrint {
		rowsToPrint = maxRowsToPrint
	}

	colsToPrint := ni
	if colsToPrint > maxColsToPrint {
		colsToPrint = maxColsToPrint
	}

	for j := 0; j < rowsToPrint; j++ {
		fmt.Printf("  Row %3d: ", j)
		for i := 0; i < colsToPrint; i++ {
			idx := j*ni + i
			if idx < len(data) {
				val := data[idx]
				if isMissing(val) {
					fmt.Printf("    MISS")
				} else {
					fmt.Printf(" %8.2f", val)
				}
			}
		}
		if ni > colsToPrint {
			fmt.Printf(" ... (%d more columns)", ni-colsToPrint)
		}
		fmt.Println()
	}

	if nj > rowsToPrint {
		fmt.Printf("  ... (%d more rows)\n", nj-rowsToPrint)
	}
	fmt.Printf("\n  Total: %d rows x %d columns = %d values\n", nj, ni, len(data))
}

func getMinMax(data []float32) (min, max float32) {
	min = float32(math.MaxFloat32)
	max = float32(-math.MaxFloat32)

	for _, v := range data {
		if !isMissing(v) {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
	}

	if min == float32(math.MaxFloat32) {
		min = 0
		max = 0
	}

	return
}

func countValid(data []float32) int {
	count := 0
	for _, v := range data {
		if !isMissing(v) {
			count++
		}
	}
	return count
}

func isMissing(v float32) bool {
	return v > 9e20
}

func keys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
