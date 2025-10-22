package grib

import (
	"fmt"
	"io"

	"github.com/mmp/mgrib2/section"
)

// FindMessagesInStream scans an io.ReadSeeker for GRIB2 message boundaries.
//
// This function performs a quick scan of the input stream to locate all GRIB2
// messages by finding "GRIB" magic numbers and reading their lengths from
// Section 0. It does not parse the full message content.
//
// The stream position is restored to its original position after scanning.
//
// Returns a slice of MessageBoundary structs indicating where each message
// starts and how long it is. The boundaries preserve the original order of
// messages in the stream.
func FindMessagesInStream(r io.ReadSeeker) ([]MessageBoundary, error) {
	// Save current position
	startPos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("failed to get current position: %w", err)
	}

	// Seek to beginning
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to start: %w", err)
	}

	var boundaries []MessageBoundary
	index := 0
	offset := int64(0)

	// Buffer for reading section 0
	sec0Buf := make([]byte, 16)

	for {
		// Try to read Section 0 (16 bytes)
		n, err := io.ReadFull(r, sec0Buf)
		if err == io.EOF {
			// Normal end of file
			break
		}
		if err == io.ErrUnexpectedEOF {
			// Incomplete data at end
			if n > 0 {
				return boundaries, &ParseError{
					Section: -1,
					Offset:  int(offset),
					Message: fmt.Sprintf("incomplete data at end of stream: %d bytes remaining, need at least 16", n),
				}
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read at offset %d: %w", offset, err)
		}

		// Check for GRIB magic number
		if sec0Buf[0] != 'G' || sec0Buf[1] != 'R' || sec0Buf[2] != 'I' || sec0Buf[3] != 'B' {
			return nil, &InvalidFormatError{
				Offset:  int(offset),
				Message: fmt.Sprintf("expected GRIB magic number, found %q", string(sec0Buf[0:4])),
			}
		}

		// Parse Section 0 to get message length
		sec0, err := section.ParseSection0(sec0Buf)
		if err != nil {
			return nil, &ParseError{
				Section:    0,
				Offset:     int(offset),
				Message:    "failed to parse Section 0",
				Underlying: err,
			}
		}

		// Seek to end of message to validate it exists and check end marker
		messageEnd := offset + int64(sec0.MessageLength)

		// Seek to 4 bytes before end to read "7777" marker
		if _, err := r.Seek(messageEnd-4, io.SeekStart); err != nil {
			return nil, &ParseError{
				Section: 0,
				Offset:  int(offset),
				Message: fmt.Sprintf("message length %d exceeds stream size", sec0.MessageLength),
			}
		}

		// Read end marker
		endMarker := make([]byte, 4)
		if _, err := io.ReadFull(r, endMarker); err != nil {
			return nil, &ParseError{
				Section: 0,
				Offset:  int(offset),
				Message: fmt.Sprintf("cannot read end marker for message at offset %d", offset),
			}
		}

		if string(endMarker) != "7777" {
			return nil, &ParseError{
				Section: -1,
				Offset:  int(messageEnd - 4),
				Message: fmt.Sprintf("expected end marker \"7777\", found %q", string(endMarker)),
			}
		}

		// Record this message boundary
		boundaries = append(boundaries, MessageBoundary{
			Start:  int(offset),
			Length: sec0.MessageLength,
			Index:  index,
		})

		// Move to next message
		offset = messageEnd
		if _, err := r.Seek(offset, io.SeekStart); err != nil {
			// If we can't seek to the next message, we're at EOF
			break
		}
		index++
	}

	// Restore original position
	if _, err := r.Seek(startPos, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to restore stream position: %w", err)
	}

	return boundaries, nil
}

// readMessageAt reads a complete GRIB2 message from the stream at the given offset.
//
// This function seeks to the specified offset, reads the message data into memory,
// and returns it as a byte slice. The stream position after this call is undefined.
func readMessageAt(r io.ReadSeeker, offset int64, length uint64) ([]byte, error) {
	// Seek to message start
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to offset %d: %w", offset, err)
	}

	// Read message data
	msgData := make([]byte, length)
	if _, err := io.ReadFull(r, msgData); err != nil {
		return nil, fmt.Errorf("failed to read message at offset %d: %w", offset, err)
	}

	return msgData, nil
}
