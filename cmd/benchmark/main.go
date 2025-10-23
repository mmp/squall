// Package main provides a benchmarking tool for GRIB2 file parsing performance.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	grib "github.com/mmp/squall"
)

type MemStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	NumGC      uint32
}

func getMemStats() MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemStats{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
	}
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

func benchmarkMgrib2(filename string, cpuprofile string, memprofile string) error {
	fmt.Printf("\n=== squall: %s ===\n", filename)

	// Force GC before starting
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	startMem := getMemStats()
	startWall := time.Now()

	var peakAlloc uint64
	var peakSys uint64

	// Start CPU profiling if requested
	var profileFile *os.File
	if cpuprofile != "" {
		var err error
		profileFile, err = os.Create(cpuprofile)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %w", err)
		}
		defer func() {
			if err := profileFile.Close(); err != nil {
				fmt.Printf("Warning: failed to close profile file: %v\n", err)
			}
		}()
		if err := pprof.StartCPUProfile(profileFile); err != nil {
			return fmt.Errorf("could not start CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	// Open file
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	// Read all messages at once
	fields, err := grib.Read(f)
	if err != nil {
		return fmt.Errorf("error reading messages: %w", err)
	}
	messageCount := len(fields)

	endWall := time.Now()

	// Final memory check
	finalMem := getMemStats()
	if finalMem.Alloc > peakAlloc {
		peakAlloc = finalMem.Alloc
	}
	if finalMem.Sys > peakSys {
		peakSys = finalMem.Sys
	}

	wallTime := endWall.Sub(startWall)

	fmt.Printf("Messages read: %d\n", messageCount)
	fmt.Printf("Wall clock time: %v\n", wallTime)
	fmt.Printf("Memory allocated at start: %s\n", formatBytes(startMem.Alloc))
	fmt.Printf("Memory allocated at end: %s\n", formatBytes(finalMem.Alloc))
	fmt.Printf("Peak memory allocated: %s\n", formatBytes(peakAlloc))
	fmt.Printf("Peak system memory: %s\n", formatBytes(peakSys))
	fmt.Printf("Total allocated during run: %s\n", formatBytes(finalMem.TotalAlloc-startMem.TotalAlloc))
	fmt.Printf("GC runs: %d\n", finalMem.NumGC-startMem.NumGC)
	if cpuprofile != "" {
		fmt.Printf("CPU profile written to: %s\n", cpuprofile)
	}
	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			return fmt.Errorf("could not create memory profile: %w", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Printf("Warning: failed to close memory profile file: %v\n", err)
			}
		}()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write memory profile: %w", err)
		}
		fmt.Printf("Memory profile written to: %s\n", memprofile)
	}

	return nil
}

func main() {
	// Define flags
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: benchmark [-cpuprofile file] [-memprofile file] <grib2-file>")
		fmt.Println("  -cpuprofile string")
		fmt.Println("        write cpu profile to file")
		fmt.Println("  -memprofile string")
		fmt.Println("        write memory profile to file")
		os.Exit(1)
	}

	filename := flag.Arg(0)

	fmt.Printf("Benchmarking file: %s\n", filename)

	// Get file size
	info, err := os.Stat(filename)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("File size: %s\n", formatBytes(uint64(info.Size())))

	// Run squall benchmark
	if err := benchmarkMgrib2(filename, *cpuprofile, *memprofile); err != nil {
		fmt.Printf("squall error: %v\n", err)
	}
}
