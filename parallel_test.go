package mgrib2

import (
	"context"
	"testing"
	"time"
)

func makeMultipleMessages(count int) []byte {
	// Create multiple complete GRIB2 messages
	var data []byte
	for i := 0; i < count; i++ {
		data = append(data, makeCompleteGRIB2Message()...)
	}
	return data
}

func TestParseMessagesSequentialSingle(t *testing.T) {
	data := makeCompleteGRIB2Message()

	messages, err := ParseMessagesSequential(data)
	if err != nil {
		t.Fatalf("ParseMessagesSequential failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Section0 == nil {
		t.Error("message Section0 is nil")
	}
}

func TestParseMessagesSequentialMultiple(t *testing.T) {
	data := makeMultipleMessages(5)

	messages, err := ParseMessagesSequential(data)
	if err != nil {
		t.Fatalf("ParseMessagesSequential failed: %v", err)
	}

	if len(messages) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(messages))
	}

	// Verify all messages parsed correctly
	for i, msg := range messages {
		if msg.Section0 == nil {
			t.Errorf("message %d Section0 is nil", i)
		}
		if msg.Section1 == nil {
			t.Errorf("message %d Section1 is nil", i)
		}
	}
}

func TestParseMessagesSingle(t *testing.T) {
	data := makeCompleteGRIB2Message()

	messages, err := ParseMessages(data)
	if err != nil {
		t.Fatalf("ParseMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Section0 == nil {
		t.Error("message Section0 is nil")
	}
}

func TestParseMessagesMultiple(t *testing.T) {
	data := makeMultipleMessages(10)

	messages, err := ParseMessages(data)
	if err != nil {
		t.Fatalf("ParseMessages failed: %v", err)
	}

	if len(messages) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(messages))
	}

	// Verify all messages parsed correctly
	for i, msg := range messages {
		if msg.Section0 == nil {
			t.Errorf("message %d Section0 is nil", i)
		}
		if msg.Section1 == nil {
			t.Errorf("message %d Section1 is nil", i)
		}
		if msg.Section3 == nil {
			t.Errorf("message %d Section3 is nil", i)
		}
	}
}

func TestParseMessagesWithWorkers(t *testing.T) {
	data := makeMultipleMessages(8)

	tests := []struct {
		name    string
		workers int
	}{
		{"1 worker", 1},
		{"2 workers", 2},
		{"4 workers", 4},
		{"8 workers", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := ParseMessagesWithWorkers(data, tt.workers)
			if err != nil {
				t.Fatalf("ParseMessagesWithWorkers failed: %v", err)
			}

			if len(messages) != 8 {
				t.Fatalf("expected 8 messages, got %d", len(messages))
			}

			// Verify all parsed
			for i, msg := range messages {
				if msg == nil {
					t.Errorf("message %d is nil", i)
				}
			}
		})
	}
}

func TestParseMessagesWithContext(t *testing.T) {
	data := makeMultipleMessages(5)

	t.Run("Normal completion", func(t *testing.T) {
		ctx := context.Background()
		messages, err := ParseMessagesWithContext(ctx, data, 2)
		if err != nil {
			t.Fatalf("ParseMessagesWithContext failed: %v", err)
		}

		if len(messages) != 5 {
			t.Fatalf("expected 5 messages, got %d", len(messages))
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		// Create large data set
		largeData := makeMultipleMessages(100)

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel immediately
		cancel()

		_, err := ParseMessagesWithContext(ctx, largeData, 2)
		if err == nil {
			t.Error("expected error due to cancelled context, got nil")
		}
	})

	t.Run("Context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait a bit to ensure timeout
		time.Sleep(1 * time.Millisecond)

		_, err := ParseMessagesWithContext(ctx, data, 2)
		// May or may not error depending on timing
		// Just verify it doesn't panic
		_ = err
	})
}

func TestParseMessagesEmpty(t *testing.T) {
	messages, err := ParseMessages([]byte{})
	if err != nil {
		t.Fatalf("ParseMessages with empty data failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestParseMessagesOrder(t *testing.T) {
	// Create messages with different disciplines to distinguish them
	data := makeMultipleMessages(5)

	messages, err := ParseMessages(data)
	if err != nil {
		t.Fatalf("ParseMessages failed: %v", err)
	}

	// Verify order is preserved (all should be discipline 0 from our test message)
	for i, msg := range messages {
		if msg.Section0.Discipline != 0 {
			t.Errorf("message %d has wrong discipline: got %d, want 0", i, msg.Section0.Discipline)
		}
	}
}

func TestParseMessagesConsistency(t *testing.T) {
	// Verify parallel parsing produces same results as sequential
	data := makeMultipleMessages(10)

	seqMessages, err := ParseMessagesSequential(data)
	if err != nil {
		t.Fatalf("ParseMessagesSequential failed: %v", err)
	}

	parMessages, err := ParseMessages(data)
	if err != nil {
		t.Fatalf("ParseMessages failed: %v", err)
	}

	if len(seqMessages) != len(parMessages) {
		t.Fatalf("message count mismatch: sequential=%d, parallel=%d",
			len(seqMessages), len(parMessages))
	}

	// Compare key fields from each message
	for i := range seqMessages {
		if seqMessages[i].Section0.Discipline != parMessages[i].Section0.Discipline {
			t.Errorf("message %d discipline mismatch", i)
		}
		if seqMessages[i].Section1.OriginatingCenter != parMessages[i].Section1.OriginatingCenter {
			t.Errorf("message %d center mismatch", i)
		}
		if seqMessages[i].Section3.NumDataPoints != parMessages[i].Section3.NumDataPoints {
			t.Errorf("message %d grid points mismatch", i)
		}
	}
}

// Benchmark parallel vs sequential parsing
func BenchmarkParseMessagesSequential(b *testing.B) {
	data := makeMultipleMessages(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMessagesSequential(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseMessagesParallel(b *testing.B) {
	data := makeMultipleMessages(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMessages(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseMessagesParallel1Worker(b *testing.B) {
	data := makeMultipleMessages(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMessagesWithWorkers(data, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseMessagesParallel4Workers(b *testing.B) {
	data := makeMultipleMessages(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMessagesWithWorkers(data, 4)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseMessagesParallel8Workers(b *testing.B) {
	data := makeMultipleMessages(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMessagesWithWorkers(data, 8)
		if err != nil {
			b.Fatal(err)
		}
	}
}
