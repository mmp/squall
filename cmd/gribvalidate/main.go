// Package main provides a command-line tool for validating GRIB2 files.
//
// This tool parses GRIB2 files message-by-message and reports which messages
// succeed or fail, making it useful for debugging GRIB2 parsing issues.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mmp/squall"
)

var (
	verboseFlag = flag.Bool("v", false, "Verbose output (show details for successful messages)")
	quietFlag   = flag.Bool("q", false, "Quiet output (only show summary)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <grib2-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Validate GRIB2 files by parsing each message individually.\n\n")
		fmt.Fprintf(os.Stderr, "This tool is useful for debugging GRIB2 parsing issues. It parses each\n")
		fmt.Fprintf(os.Stderr, "message in the file separately and reports successes and failures with\n")
		fmt.Fprintf(os.Stderr, "detailed error information.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s file.grib2           # Validate file, show failures\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -v file.grib2        # Show details for all messages\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -q file.grib2        # Only show summary\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	gribPath := flag.Arg(0)

	if err := validateGRIBFile(gribPath); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
}

// validateGRIBFile analyzes a GRIB2 file message-by-message to identify parsing failures
func validateGRIBFile(gribPath string) error {
	f, err := os.Open(gribPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
		}
	}()

	// Find all message boundaries
	boundaries, err := squall.FindMessagesInStream(f)
	if err != nil {
		return fmt.Errorf("failed to find messages: %w", err)
	}

	if !*quietFlag {
		fmt.Println("=== GRIB2 File Validation ===")
		fmt.Printf("File: %s\n", gribPath)
		fmt.Printf("Total messages found: %d\n", len(boundaries))
		fmt.Println()
	}

	successCount := 0
	failCount := 0

	for _, boundary := range boundaries {
		// Read single message
		if _, err := f.Seek(int64(boundary.Start), io.SeekStart); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Message %d: seek failed: %v\n", boundary.Index, err)
			failCount++
			continue
		}

		msgData := make([]byte, boundary.Length)
		if _, err := io.ReadFull(f, msgData); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Message %d: read failed: %v\n", boundary.Index, err)
			failCount++
			continue
		}

		// Try to parse
		msgReader := bytes.NewReader(msgData)
		msgs, err := squall.Read(msgReader)

		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Message %d FAILED:\n", boundary.Index)
			fmt.Fprintf(os.Stderr, "  Offset: %d\n", boundary.Start)
			fmt.Fprintf(os.Stderr, "  Length: %d bytes\n", boundary.Length)
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			fmt.Fprintln(os.Stderr)
			failCount++
			continue
		}

		if len(msgs) > 0 {
			msg := msgs[0]
			if *verboseFlag {
				fmt.Printf("Message %d SUCCESS:\n", boundary.Index)
				fmt.Printf("  Parameter: %s\n", msg.Parameter.ShortName())
				fmt.Printf("  Level: %s", msg.Level)
				if msg.LevelValue != 0 {
					fmt.Printf(" (%.1f)", msg.LevelValue)
				}
				fmt.Println()
				fmt.Printf("  Points: %d\n", msg.NumPoints)
				fmt.Printf("  Grid: %dx%d\n", msg.GridNi, msg.GridNj)
				fmt.Println()
			}
			successCount++
		}
	}

	if !*quietFlag {
		fmt.Println("=== Summary ===")
	}
	fmt.Printf("Success: %d messages\n", successCount)
	fmt.Printf("Failed: %d messages\n", failCount)

	if failCount > 0 {
		return fmt.Errorf("%d messages failed to parse", failCount)
	}

	if !*quietFlag {
		fmt.Println("\n✓ All messages validated successfully")
	}

	return nil
}
