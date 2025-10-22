package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	mgrib2 "github.com/mmp/mgrib2"
	gogrib2 "github.com/mmp/go-grib2"
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
	fmt.Printf("\n=== mgrib2: %s ===\n", filename)

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
		defer profileFile.Close()
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
	defer f.Close()

	// Read all messages at once
	fields, err := mgrib2.Read(f)
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
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write memory profile: %w", err)
		}
		fmt.Printf("Memory profile written to: %s\n", memprofile)
	}

	return nil
}

func benchmarkGoGrib2(filename string, cpuprofile string, memprofile string) error{
	fmt.Printf("\n=== go-grib2: %s ===\n", filename)

	// Force GC before starting
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	startMem := getMemStats()
	startWall := time.Now()

	var peakAlloc uint64
	var peakSys uint64
	messageCount := 0

	// Start CPU profiling if requested
	var profileFile *os.File
	if cpuprofile != "" {
		var err error
		profileFile, err = os.Create(cpuprofile)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %w", err)
		}
		defer profileFile.Close()
		if err := pprof.StartCPUProfile(profileFile); err != nil {
			return fmt.Errorf("could not start CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	// Open file and read all data
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// Read all messages using go-grib2
	messages, err := gogrib2.Read(data)
	if err != nil {
		return err
	}
	messageCount = len(messages)

	// Check peak memory
	m := getMemStats()
	if m.Alloc > peakAlloc {
		peakAlloc = m.Alloc
	}
	if m.Sys > peakSys {
		peakSys = m.Sys
	}

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
		defer f.Close()
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
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file` (separate files for mgrib2 and go-grib2)")
	memprofile := flag.String("memprofile", "", "write memory profile to `file` (separate files for mgrib2 and go-grib2)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: benchmark [-cpuprofile file] [-memprofile file] <grib2-file>")
		fmt.Println("  -cpuprofile string")
		fmt.Println("        write cpu profiles to files (will create <file>.mgrib2 and <file>.gogrib2)")
		fmt.Println("  -memprofile string")
		fmt.Println("        write memory profiles to files (will create <file>.mgrib2 and <file>.gogrib2)")
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

	// Determine profile file names
	mgrib2CPUProfile := ""
	gogrib2CPUProfile := ""
	if *cpuprofile != "" {
		mgrib2CPUProfile = *cpuprofile + ".mgrib2"
		gogrib2CPUProfile = *cpuprofile + ".gogrib2"
	}

	mgrib2MemProfile := ""
	gogrib2MemProfile := ""
	if *memprofile != "" {
		mgrib2MemProfile = *memprofile + ".mgrib2"
		gogrib2MemProfile = *memprofile + ".gogrib2"
	}

	// Run mgrib2 benchmark
	if err := benchmarkMgrib2(filename, mgrib2CPUProfile, mgrib2MemProfile); err != nil {
		fmt.Printf("mgrib2 error: %v\n", err)
	}

	// Wait a bit between benchmarks
	time.Sleep(1 * time.Second)
	runtime.GC()
	time.Sleep(1 * time.Second)

	// Run go-grib2 benchmark
	if err := benchmarkGoGrib2(filename, gogrib2CPUProfile, gogrib2MemProfile); err != nil {
		fmt.Printf("go-grib2 error: %v\n", err)
	}
}
