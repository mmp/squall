package squall

import (
	"fmt"

	"github.com/mmp/squall/section"
)

// MessageBoundary represents the location and size of a GRIB2 message within a file.
type MessageBoundary struct {
	Start  int    // Byte offset where the message starts
	Length uint64 // Length of the message in bytes
	Index  int    // Sequential index of this message in the file (0-based)
}

// FindMessages scans the data for GRIB2 message boundaries.
//
// This function performs a quick scan of the input data to locate all GRIB2
// messages by finding "GRIB" magic numbers and reading their lengths from
// Section 0. It does not parse the full message content.
//
// Returns a slice of MessageBoundary structs indicating where each message
// starts and how long it is. The boundaries preserve the original order of
// messages in the file.
//
// This function is designed to be fast so that message boundaries can be
// found quickly before parallel decoding begins.
func FindMessages(data []byte) ([]MessageBoundary, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var boundaries []MessageBoundary
	offset := 0
	index := 0

	for offset < len(data) {
		// Look for "GRIB" magic number
		if offset+16 > len(data) {
			// Not enough data for a complete Section 0
			if offset < len(data) {
				// There's some data left but not enough for a message
				return boundaries, &ParseError{
					Section: -1,
					Offset:  offset,
					Message: fmt.Sprintf("incomplete data at end of file: %d bytes remaining, need at least 16", len(data)-offset),
				}
			}
			break
		}

		// Check for GRIB magic number
		if data[offset] != 'G' || data[offset+1] != 'R' || data[offset+2] != 'I' || data[offset+3] != 'B' {
			return nil, &InvalidFormatError{
				Offset:  offset,
				Message: fmt.Sprintf("expected GRIB magic number, found %q", string(data[offset:offset+4])),
			}
		}

		// Parse Section 0 to get message length
		sec0Data := data[offset : offset+16]
		sec0, err := section.ParseSection0(sec0Data)
		if err != nil {
			return nil, &ParseError{
				Section:    0,
				Offset:     offset,
				Message:    "failed to parse Section 0",
				Underlying: err,
			}
		}

		// Validate that we have enough data for the complete message
		messageEnd := offset + int(sec0.MessageLength)
		if messageEnd > len(data) {
			return nil, &ParseError{
				Section: 0,
				Offset:  offset,
				Message: fmt.Sprintf("message length %d exceeds available data (have %d bytes from offset %d)",
					sec0.MessageLength, len(data)-offset, offset),
			}
		}

		// Validate that the message ends with "7777"
		endMarker := data[messageEnd-4 : messageEnd]
		if string(endMarker) != "7777" {
			return nil, &ParseError{
				Section: -1,
				Offset:  messageEnd - 4,
				Message: fmt.Sprintf("expected end marker \"7777\", found %q", string(endMarker)),
			}
		}

		// Record this message boundary
		boundaries = append(boundaries, MessageBoundary{
			Start:  offset,
			Length: sec0.MessageLength,
			Index:  index,
		})

		// Move to next message
		offset = messageEnd
		index++
	}

	return boundaries, nil
}

// SplitMessages splits the data into individual GRIB2 messages.
//
// This is a convenience function that calls FindMessages and then extracts
// the actual message data for each boundary.
//
// Returns a slice of byte slices, where each inner slice is a complete
// GRIB2 message.
func SplitMessages(data []byte) ([][]byte, error) {
	boundaries, err := FindMessages(data)
	if err != nil {
		return nil, err
	}

	messages := make([][]byte, len(boundaries))
	for i, boundary := range boundaries {
		messages[i] = data[boundary.Start : boundary.Start+int(boundary.Length)]
	}

	return messages, nil
}

// ValidateMessageStructure performs a basic validation of a GRIB2 message structure.
//
// This function checks that:
//   - The message starts with "GRIB"
//   - Section 0 is valid
//   - The message ends with "7777"
//   - The message length matches the data length
//
// It does NOT parse the full message content or validate all sections.
func ValidateMessageStructure(data []byte) error {
	if len(data) < 16 {
		return &ParseError{
			Section: -1,
			Offset:  0,
			Message: fmt.Sprintf("message too short: %d bytes, minimum is 16", len(data)),
		}
	}

	// Parse Section 0
	sec0, err := section.ParseSection0(data[0:16])
	if err != nil {
		return &ParseError{
			Section:    0,
			Offset:     0,
			Message:    "invalid Section 0",
			Underlying: err,
		}
	}

	// Check message length
	if uint64(len(data)) != sec0.MessageLength {
		return &ParseError{
			Section: 0,
			Offset:  0,
			Message: fmt.Sprintf("message length mismatch: Section 0 says %d bytes, but have %d bytes",
				sec0.MessageLength, len(data)),
		}
	}

	// Check for end marker "7777"
	if len(data) < 4 {
		return &ParseError{
			Section: -1,
			Offset:  len(data),
			Message: "message too short to contain end marker",
		}
	}

	endMarker := data[len(data)-4:]
	if string(endMarker) != "7777" {
		return &ParseError{
			Section: -1,
			Offset:  len(data) - 4,
			Message: fmt.Sprintf("expected end marker \"7777\", found %q", string(endMarker)),
		}
	}

	return nil
}
