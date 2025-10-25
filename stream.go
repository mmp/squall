package squall

import (
	"fmt"
	"io"

	"github.com/mmp/squall/section"
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

	boundaries, totalBytesRead, err := scanForGRIBMessages(r)
	if err != nil {
		return nil, err
	}

	// If no GRIB messages were found and we read data, return an error
	// Empty streams (0 bytes) are valid and return 0 messages
	if len(boundaries) == 0 && totalBytesRead > 0 {
		return nil, &InvalidFormatError{
			Offset:  0,
			Message: "no valid GRIB2 messages found in stream",
		}
	}

	// Restore original position
	if _, err := r.Seek(startPos, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to restore stream position: %w", err)
	}

	return boundaries, nil
}

// scanForGRIBMessages scans the stream for GRIB2 messages
func scanForGRIBMessages(r io.ReadSeeker) ([]MessageBoundary, int64, error) {
	var boundaries []MessageBoundary
	index := 0
	offset := int64(0)
	totalBytesRead := int64(0)

	// Buffer for scanning for GRIB magic numbers
	const bufSize = 4096
	buf := make([]byte, bufSize)
	sec0Buf := make([]byte, 16)

	for {
		// Read next chunk
		n, err := r.Read(buf)
		if err == io.EOF {
			if n == 0 {
				break
			}
			// Process remaining bytes
		} else if err != nil {
			return nil, 0, fmt.Errorf("failed to read at offset %d: %w", offset, err)
		}
		totalBytesRead += int64(n)

		// Search for "GRIB" in this chunk
		for i := 0; i < n-3; i++ {
			if isGRIBMarker(buf, i) {
				msgOffset := offset + int64(i)
				boundary, newOffset, found := validateAndCreateBoundary(r, msgOffset, sec0Buf, index)
				if found {
					boundaries = append(boundaries, boundary)
					index++
					offset = newOffset
					// Exit inner loop to read next chunk from new offset
					goto nextChunk
				}
			}
		}

		// Move offset forward
		offset += int64(n)
	nextChunk:
	}

	return boundaries, totalBytesRead, nil
}

// isGRIBMarker checks if the buffer contains "GRIB" at the given position
func isGRIBMarker(buf []byte, i int) bool {
	return buf[i] == 'G' && buf[i+1] == 'R' && buf[i+2] == 'I' && buf[i+3] == 'B'
}

// validateAndCreateBoundary validates a potential GRIB message and creates a boundary
func validateAndCreateBoundary(r io.ReadSeeker, msgOffset int64, sec0Buf []byte, index int) (MessageBoundary, int64, bool) {
	// Seek to this position to read full Section 0
	if _, err := r.Seek(msgOffset, io.SeekStart); err != nil {
		return MessageBoundary{}, 0, false
	}

	// Read Section 0
	if _, err := io.ReadFull(r, sec0Buf); err != nil {
		return MessageBoundary{}, 0, false
	}

	// Parse Section 0 to get message length
	sec0, err := section.ParseSection0(sec0Buf)
	if err != nil {
		return MessageBoundary{}, 0, false
	}

	// Validate message by checking end marker
	messageEnd := msgOffset + int64(sec0.MessageLength)
	if !validateEndMarker(r, messageEnd) {
		return MessageBoundary{}, 0, false
	}

	// Valid GRIB message found!
	boundary := MessageBoundary{
		Start:  int(msgOffset),
		Length: sec0.MessageLength,
		Index:  index,
	}

	// Continue scanning after this message
	if _, err := r.Seek(messageEnd, io.SeekStart); err != nil {
		return boundary, 0, true // Return boundary but signal end
	}

	return boundary, messageEnd, true
}

// validateEndMarker checks if the message has a valid "7777" end marker
func validateEndMarker(r io.ReadSeeker, messageEnd int64) bool {
	// Seek to 4 bytes before end to read "7777" marker
	if _, err := r.Seek(messageEnd-4, io.SeekStart); err != nil {
		return false
	}

	// Read end marker
	endMarker := make([]byte, 4)
	if _, err := io.ReadFull(r, endMarker); err != nil {
		return false
	}

	return string(endMarker) == "7777"
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
