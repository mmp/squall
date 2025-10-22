// Package internal provides internal utilities for squall.
package internal

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Reader provides safe binary reading with error checking.
// All GRIB2 data is big-endian (network byte order).
type Reader struct {
	data   []byte
	offset int
}

// NewReader creates a new binary reader from a byte slice.
func NewReader(data []byte) *Reader {
	return &Reader{data: data, offset: 0}
}

// Uint8 reads an unsigned 8-bit integer.
func (r *Reader) Uint8() (uint8, error) {
	if r.offset+1 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	val := r.data[r.offset]
	r.offset++
	return val, nil
}

// Int8 reads a signed 8-bit integer.
func (r *Reader) Int8() (int8, error) {
	val, err := r.Uint8()
	return int8(val), err
}

// Uint16 reads an unsigned 16-bit big-endian integer.
func (r *Reader) Uint16() (uint16, error) {
	if r.offset+2 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	val := binary.BigEndian.Uint16(r.data[r.offset:])
	r.offset += 2
	return val, nil
}

// Int16 reads a signed 16-bit big-endian integer.
// GRIB2 uses sign-magnitude representation, not two's complement:
// - Bit 15 (0x8000) is the sign bit (1 = negative, 0 = positive)
// - Bits 0-14 (0x7FFF) are the magnitude
func (r *Reader) Int16() (int16, error) {
	val, err := r.Uint16()
	if err != nil {
		return 0, err
	}

	// Check sign bit (bit 15)
	if (val & 0x8000) != 0 {
		// Negative: extract magnitude from bits 0-14 and negate
		return -int16(val & 0x7FFF), nil
	}
	// Positive: use value as-is
	return int16(val), nil
}

// Uint32 reads an unsigned 32-bit big-endian integer.
func (r *Reader) Uint32() (uint32, error) {
	if r.offset+4 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	val := binary.BigEndian.Uint32(r.data[r.offset:])
	r.offset += 4
	return val, nil
}

// Int32 reads a signed 32-bit big-endian integer.
func (r *Reader) Int32() (int32, error) {
	val, err := r.Uint32()
	return int32(val), err
}

// Uint64 reads an unsigned 64-bit big-endian integer.
func (r *Reader) Uint64() (uint64, error) {
	if r.offset+8 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	val := binary.BigEndian.Uint64(r.data[r.offset:])
	r.offset += 8
	return val, nil
}

// Int64 reads a signed 64-bit big-endian integer.
func (r *Reader) Int64() (int64, error) {
	val, err := r.Uint64()
	return int64(val), err
}

// Float32 reads a 32-bit IEEE 754 floating-point number in big-endian format.
func (r *Reader) Float32() (float32, error) {
	bits, err := r.Uint32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(bits), nil
}

// Float64 reads a 64-bit IEEE 754 floating-point number in big-endian format.
func (r *Reader) Float64() (float64, error) {
	bits, err := r.Uint64()
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(bits), nil
}

// Bytes reads n bytes and returns a copy.
// The returned slice is a copy, not a reference to the internal buffer.
func (r *Reader) Bytes(n int) ([]byte, error) {
	if r.offset+n > len(r.data) {
		return nil, io.ErrUnexpectedEOF
	}
	// Return a copy to prevent aliasing issues
	val := make([]byte, n)
	copy(val, r.data[r.offset:r.offset+n])
	r.offset += n
	return val, nil
}

// BytesNoCopy reads n bytes and returns a slice referencing the internal buffer.
// WARNING: The returned slice is only valid until the next read operation or
// until the underlying data is modified. Use Bytes() if you need a stable copy.
func (r *Reader) BytesNoCopy(n int) ([]byte, error) {
	if r.offset+n > len(r.data) {
		return nil, io.ErrUnexpectedEOF
	}
	val := r.data[r.offset : r.offset+n]
	r.offset += n
	return val, nil
}

// String reads n bytes and returns them as a string.
func (r *Reader) String(n int) (string, error) {
	if r.offset+n > len(r.data) {
		return "", io.ErrUnexpectedEOF
	}
	val := string(r.data[r.offset : r.offset+n])
	r.offset += n
	return val, nil
}

// Skip advances the offset by n bytes without reading.
func (r *Reader) Skip(n int) error {
	if r.offset+n > len(r.data) {
		return io.ErrUnexpectedEOF
	}
	r.offset += n
	return nil
}

// Peek returns the next n bytes without advancing the offset.
func (r *Reader) Peek(n int) ([]byte, error) {
	if r.offset+n > len(r.data) {
		return nil, io.ErrUnexpectedEOF
	}
	return r.data[r.offset : r.offset+n], nil
}

// Remaining returns the number of unread bytes.
func (r *Reader) Remaining() int {
	return len(r.data) - r.offset
}

// Offset returns the current byte offset.
func (r *Reader) Offset() int {
	return r.offset
}

// SetOffset sets the current byte offset.
// Returns an error if the offset is out of bounds.
func (r *Reader) SetOffset(offset int) error {
	if offset < 0 || offset > len(r.data) {
		return fmt.Errorf("offset %d out of bounds [0, %d]", offset, len(r.data))
	}
	r.offset = offset
	return nil
}

// Len returns the total length of the data.
func (r *Reader) Len() int {
	return len(r.data)
}

// BitReader provides bit-level reading for packed data.
// GRIB2 sometimes packs data at bit boundaries (e.g., 12-bit values).
type BitReader struct {
	data    []byte
	offset  int  // bit offset
	maxBits int  // total number of bits available
}

// NewBitReader creates a new bit-level reader.
func NewBitReader(data []byte) *BitReader {
	return &BitReader{
		data:    data,
		offset:  0,
		maxBits: len(data) * 8,
	}
}

// ReadBits reads up to 64 bits as an unsigned integer.
// nbits must be in the range [1, 64].
func (br *BitReader) ReadBits(nbits int) (uint64, error) {
	if nbits < 1 || nbits > 64 {
		return 0, fmt.Errorf("nbits must be in range [1, 64], got %d", nbits)
	}

	if br.offset+nbits > br.maxBits {
		return 0, io.ErrUnexpectedEOF
	}

	var result uint64

	bitsRemaining := nbits
	for bitsRemaining > 0 {
		byteIndex := br.offset / 8
		bitOffset := br.offset % 8

		// How many bits can we read from the current byte?
		bitsInCurrentByte := 8 - bitOffset
		bitsToRead := bitsRemaining
		if bitsToRead > bitsInCurrentByte {
			bitsToRead = bitsInCurrentByte
		}

		// Read the bits from the current byte
		mask := byte((1 << bitsToRead) - 1)
		shift := bitsInCurrentByte - bitsToRead
		bits := (br.data[byteIndex] >> shift) & mask

		// Add to result
		result = (result << bitsToRead) | uint64(bits)

		br.offset += bitsToRead
		bitsRemaining -= bitsToRead
	}

	return result, nil
}

// ReadSignedBits reads up to 64 bits as a signed integer.
// nbits must be in the range [1, 64].
// The value is assumed to be in two's complement format.
func (br *BitReader) ReadSignedBits(nbits int) (int64, error) {
	val, err := br.ReadBits(nbits)
	if err != nil {
		return 0, err
	}

	// Check if sign bit is set
	signBit := uint64(1) << (nbits - 1)
	if val&signBit != 0 {
		// Negative number: extend sign bits
		mask := ^uint64(0) << nbits
		val |= mask
	}

	return int64(val), nil
}

// Offset returns the current bit offset.
func (br *BitReader) Offset() int {
	return br.offset
}

// Skip skips the specified number of bits.
func (br *BitReader) Skip(nbits int) error {
	if br.offset+nbits > br.maxBits {
		return io.ErrUnexpectedEOF
	}
	br.offset += nbits
	return nil
}

// Align aligns the offset to the next byte boundary.
// If already aligned, does nothing.
func (br *BitReader) Align() {
	remainder := br.offset % 8
	if remainder != 0 {
		br.offset += 8 - remainder
	}
}

// ReadBytes reads n bytes (octets) as an unsigned integer.
// The reader must be byte-aligned when calling this method.
// Returns the value as uint64 (big-endian).
func (br *BitReader) ReadBytes(n int) (uint64, error) {
	if n < 1 || n > 8 {
		return 0, fmt.Errorf("ReadBytes n must be in range [1, 8], got %d", n)
	}

	// Ensure we're byte-aligned
	if br.offset%8 != 0 {
		return 0, fmt.Errorf("ReadBytes called on non-byte-aligned offset %d", br.offset)
	}

	if br.offset+n*8 > br.maxBits {
		return 0, io.ErrUnexpectedEOF
	}

	byteIndex := br.offset / 8
	var result uint64

	for i := 0; i < n; i++ {
		result = (result << 8) | uint64(br.data[byteIndex+i])
	}

	br.offset += n * 8
	return result, nil
}

// ReadSignedBytes reads n bytes (octets) as a signed integer (two's complement).
// The reader must be byte-aligned when calling this method.
// Returns the value as int64 (big-endian).
func (br *BitReader) ReadSignedBytes(n int) (int64, error) {
	if n < 1 || n > 8 {
		return 0, fmt.Errorf("ReadSignedBytes n must be in range [1, 8], got %d", n)
	}

	val, err := br.ReadBytes(n)
	if err != nil {
		return 0, err
	}

	// Check if sign bit is set (most significant bit of the first byte)
	signBit := uint64(1) << (n*8 - 1)
	if val&signBit != 0 {
		// Negative number: extend sign bits
		mask := ^uint64(0) << (n * 8)
		val |= mask
	}

	return int64(val), nil
}

// ReadSignedBytesSignMagnitude reads n bytes (octets) as a signed integer using sign-magnitude encoding.
// Sign-magnitude encoding: the most significant bit is the sign bit (1=negative),
// and the remaining bits represent the magnitude.
// The reader must be byte-aligned when calling this method.
// Returns the value as int64 (big-endian).
func (br *BitReader) ReadSignedBytesSignMagnitude(n int) (int64, error) {
	if n < 1 || n > 8 {
		return 0, fmt.Errorf("ReadSignedBytesSignMagnitude n must be in range [1, 8], got %d", n)
	}

	val, err := br.ReadBytes(n)
	if err != nil {
		return 0, err
	}

	// Extract sign bit (most significant bit of first byte)
	signBit := uint64(1) << (n*8 - 1)
	isNegative := (val & signBit) != 0

	// Extract magnitude (all bits except the sign bit)
	magnitude := val & ^signBit

	if isNegative {
		return -int64(magnitude), nil
	}
	return int64(magnitude), nil
}
