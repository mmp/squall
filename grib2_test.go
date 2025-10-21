package mgrib2

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestRead(t *testing.T) {
	data := makeCompleteGRIB2Message()

	fields, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}

	field := fields[0]

	// Verify data
	if len(field.Data) != 9 {
		t.Errorf("expected 9 data values, got %d", len(field.Data))
	}

	// Verify coordinates
	if len(field.Latitudes) != 9 {
		t.Errorf("expected 9 latitudes, got %d", len(field.Latitudes))
	}
	if len(field.Longitudes) != 9 {
		t.Errorf("expected 9 longitudes, got %d", len(field.Longitudes))
	}

	// Verify metadata
	if field.Discipline == "" {
		t.Error("Discipline is empty")
	}
	if field.Center == "" {
		t.Error("Center is empty")
	}
	if field.ReferenceTime.IsZero() {
		t.Error("ReferenceTime is zero")
	}
}

func TestReadMultiple(t *testing.T) {
	data := makeMultipleMessages(5)

	fields, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(fields))
	}

	for i, field := range fields {
		if field == nil {
			t.Errorf("field %d is nil", i)
			continue
		}
		if len(field.Data) == 0 {
			t.Errorf("field %d has no data", i)
		}
	}
}

func TestReadWithOptionsWorkers(t *testing.T) {
	data := makeMultipleMessages(10)

	fields, err := ReadWithOptions(bytes.NewReader(data), WithWorkers(4))
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 10 {
		t.Fatalf("expected 10 fields, got %d", len(fields))
	}
}

func TestReadWithOptionsSequential(t *testing.T) {
	data := makeMultipleMessages(5)

	fields, err := ReadWithOptions(bytes.NewReader(data), WithSequential())
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(fields))
	}
}

func TestReadWithOptionsContext(t *testing.T) {
	data := makeCompleteGRIB2Message()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fields, err := ReadWithOptions(bytes.NewReader(data), WithContext(ctx))
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
}

func TestReadWithOptionsFilter(t *testing.T) {
	data := makeMultipleMessages(10)

	// Filter to only include even-indexed messages (as a test)
	count := 0
	filter := func(msg *Message) bool {
		count++
		return count%2 == 0
	}

	fields, err := ReadWithOptions(bytes.NewReader(data), WithFilter(filter))
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 5 {
		t.Fatalf("expected 5 fields (50%% filtered), got %d", len(fields))
	}
}

func TestReadWithOptionsParameterCategory(t *testing.T) {
	data := makeCompleteGRIB2Message()

	// Filter for temperature (category 0)
	fields, err := ReadWithOptions(bytes.NewReader(data), WithParameterCategory(0))
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}

	// Filter for non-existent category
	fields, err = ReadWithOptions(bytes.NewReader(data), WithParameterCategory(99))
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 0 {
		t.Fatalf("expected 0 fields (filtered out), got %d", len(fields))
	}
}

func TestReadWithOptionsDiscipline(t *testing.T) {
	data := makeCompleteGRIB2Message()

	// Filter for meteorological (discipline 0)
	fields, err := ReadWithOptions(bytes.NewReader(data), WithDiscipline(0))
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}

	// Filter for non-existent discipline
	fields, err = ReadWithOptions(bytes.NewReader(data), WithDiscipline(99))
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 0 {
		t.Fatalf("expected 0 fields (filtered out), got %d", len(fields))
	}
}

func TestReadWithOptionsCenter(t *testing.T) {
	data := makeCompleteGRIB2Message()

	// Filter for NCEP (center 7)
	fields, err := ReadWithOptions(bytes.NewReader(data), WithCenter(7))
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
}

func TestGRIB2MinMaxValue(t *testing.T) {
	data := makeCompleteGRIB2Message()

	fields, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	field := fields[0]

	// Data values are 250.0, 251.0, ..., 258.0
	min := field.MinValue()
	if min != 250.0 {
		t.Errorf("MinValue: got %.1f, want 250.0", min)
	}

	max := field.MaxValue()
	if max != 258.0 {
		t.Errorf("MaxValue: got %.1f, want 258.0", max)
	}
}

func TestGRIB2CountValid(t *testing.T) {
	data := makeCompleteGRIB2Message()

	fields, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	field := fields[0]

	count := field.CountValid()
	if count != 9 {
		t.Errorf("CountValid: got %d, want 9", count)
	}
}

func TestGRIB2String(t *testing.T) {
	data := makeCompleteGRIB2Message()

	fields, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	field := fields[0]

	str := field.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	if len(str) < 20 {
		t.Errorf("String() too short: %q", str)
	}
}

func TestGRIB2GetMessage(t *testing.T) {
	data := makeCompleteGRIB2Message()

	fields, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	field := fields[0]

	msg := field.GetMessage()
	if msg == nil {
		t.Fatal("GetMessage() returned nil")
	}

	if msg.Section0 == nil {
		t.Error("Message Section0 is nil")
	}
}

func TestReadEmpty(t *testing.T) {
	fields, err := Read(bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("Read with empty data failed: %v", err)
	}

	if len(fields) != 0 {
		t.Errorf("expected 0 fields, got %d", len(fields))
	}
}

func TestReadInvalid(t *testing.T) {
	_, err := Read(bytes.NewReader([]byte("invalid data")))
	if err == nil {
		t.Error("expected error for invalid data, got nil")
	}
}

func TestReadWithOptionsCombined(t *testing.T) {
	data := makeMultipleMessages(10)

	// Combine multiple options
	fields, err := ReadWithOptions(bytes.NewReader(data),
		WithWorkers(2),
		WithParameterCategory(0),
		WithDiscipline(0),
	)
	if err != nil {
		t.Fatalf("ReadWithOptions failed: %v", err)
	}

	// All test messages have category 0 and discipline 0
	if len(fields) != 10 {
		t.Fatalf("expected 10 fields, got %d", len(fields))
	}
}

func BenchmarkRead(b *testing.B) {
	data := makeMultipleMessages(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Read(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadWithWorkers4(b *testing.B) {
	data := makeMultipleMessages(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ReadWithOptions(bytes.NewReader(data), WithWorkers(4))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadSequential(b *testing.B) {
	data := makeMultipleMessages(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ReadWithOptions(bytes.NewReader(data), WithSequential())
		if err != nil {
			b.Fatal(err)
		}
	}
}
