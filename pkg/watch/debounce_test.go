package watch

import (
	"context"
	"testing"
	"time"

	"github.com/nanaki-93/kudev/test/util"
)

func TestDebouncer_BatchesEvents(t *testing.T) {
	config := DebounceConfig{Window: 100 * time.Millisecond}
	debouncer := NewDebouncer(config, &util.MockLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	input := make(chan FileChangeEvent)
	output := debouncer.Debounce(ctx, input)

	// Send rapid events
	go func() {
		input <- FileChangeEvent{Path: "file1.go", Op: "write"}
		input <- FileChangeEvent{Path: "file2.go", Op: "write"}
		input <- FileChangeEvent{Path: "file3.go", Op: "write"}
		close(input)
	}()

	// Should receive single batch
	select {
	case batch := <-output:
		if len(batch) != 3 {
			t.Errorf("expected 3 events in batch, got %d", len(batch))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for batch")
	}
}

func TestDebouncer_ResetsTimerOnNewEvent(t *testing.T) {
	config := DebounceConfig{Window: 200 * time.Millisecond}
	debouncer := NewDebouncer(config, &util.MockLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	input := make(chan FileChangeEvent)
	output := debouncer.Debounce(ctx, input)

	start := time.Now()

	// Send events with delays (but within window)
	go func() {
		input <- FileChangeEvent{Path: "file1.go", Op: "write"}
		time.Sleep(100 * time.Millisecond)
		input <- FileChangeEvent{Path: "file2.go", Op: "write"}
		time.Sleep(100 * time.Millisecond)
		input <- FileChangeEvent{Path: "file3.go", Op: "write"}
		close(input)
	}()

	// Wait for batch
	<-output
	elapsed := time.Since(start)

	// Should have waited ~200ms after last event timer not reset
	if elapsed < 200*time.Millisecond {
		t.Errorf("debounce triggered too early: %v", elapsed)
	}
}

func TestDebouncer_SeparateBatches(t *testing.T) {
	config := DebounceConfig{Window: 50 * time.Millisecond}
	debouncer := NewDebouncer(config, &util.MockLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	input := make(chan FileChangeEvent)
	output := debouncer.Debounce(ctx, input)

	// Send two separate batches
	go func() {
		// First batch
		input <- FileChangeEvent{Path: "file1.go", Op: "write"}

		// Wait for first batch to trigger
		time.Sleep(150 * time.Millisecond)

		// Second batch
		input <- FileChangeEvent{Path: "file2.go", Op: "write"}

		time.Sleep(100 * time.Millisecond)
		close(input)
	}()

	// Should receive two batches
	batchCount := 0
	for range output {
		batchCount++
	}

	if batchCount != 2 {
		t.Errorf("expected 2 batches, got %d", batchCount)
	}
}

func TestDebouncer_CancelStopsProcessing(t *testing.T) {
	config := DebounceConfig{Window: 100 * time.Millisecond}
	debouncer := NewDebouncer(config, &util.MockLogger{})

	ctx, cancel := context.WithCancel(context.Background())

	input := make(chan FileChangeEvent)
	output := debouncer.Debounce(ctx, input)

	// Send event
	input <- FileChangeEvent{Path: "file1.go", Op: "write"}

	// Cancel before window expires
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Output should close without sending
	select {
	case _, ok := <-output:
		if ok {
			t.Error("should not receive event after cancel")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("output channel should be closed")
	}
}

func TestDebouncer_Reset(t *testing.T) {
	config := DebounceConfig{Window: 100 * time.Millisecond}
	debouncer := NewDebouncer(config, &util.MockLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	input := make(chan FileChangeEvent)
	_ = debouncer.Debounce(ctx, input)

	// Add events
	input <- FileChangeEvent{Path: "file1.go", Op: "write"}

	// Reset before trigger
	time.Sleep(50 * time.Millisecond)
	debouncer.Reset()

	// Verify events cleared
	if len(debouncer.events) != 0 {
		t.Errorf("expected 0 events after reset, got %d", len(debouncer.events))
	}
}
