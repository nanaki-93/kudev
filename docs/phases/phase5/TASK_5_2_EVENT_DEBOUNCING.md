# Task 5.2: Implement Event Debouncing

## Overview

This task implements **event debouncing** to batch rapid file changes into a single rebuild trigger.

**Effort**: ~2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: Task 5.1 (Watcher)  
**Files to Create**:
- `pkg/watch/debounce.go` — Debouncing logic
- `pkg/watch/debounce_test.go` — Tests

---

## What You're Building

A debouncer that:
1. **Collects** events within a time window
2. **Resets** timer on new events
3. **Fires** callback after quiet period
4. **Skips** rebuild if hash unchanged

---

## The Problem

Without debouncing:
```
Save file → Build triggered
Save file → Build triggered
Save file → Build triggered
# 3 builds for one "save all" operation!
```

With debouncing:
```
Save file ──┐
Save file ──┼──► [Wait 500ms] ──► Single build
Save file ──┘
```

---

## Complete Implementation

```go
// pkg/watch/debounce.go

package watch

import (
    "context"
    "sync"
    "time"
    
    "github.com/your-org/kudev/pkg/logging"
)

// DebounceConfig configures the debouncer behavior.
type DebounceConfig struct {
    // Window is how long to wait for more events before triggering.
    // Default: 500ms
    Window time.Duration
}

// DefaultDebounceConfig returns sensible defaults.
func DefaultDebounceConfig() DebounceConfig {
    return DebounceConfig{
        Window: 500 * time.Millisecond,
    }
}

// Debouncer batches rapid events into single triggers.
type Debouncer struct {
    config  DebounceConfig
    logger  logging.Logger
    
    mu      sync.Mutex
    timer   *time.Timer
    events  []FileChangeEvent
}

// NewDebouncer creates a new debouncer.
func NewDebouncer(config DebounceConfig, logger logging.Logger) *Debouncer {
    return &Debouncer{
        config: config,
        logger: logger,
        events: make([]FileChangeEvent, 0),
    }
}

// Debounce takes input events and returns debounced events.
// Multiple rapid input events result in single output after quiet period.
func (d *Debouncer) Debounce(ctx context.Context, input <-chan FileChangeEvent) <-chan []FileChangeEvent {
    output := make(chan []FileChangeEvent)
    
    go d.processEvents(ctx, input, output)
    
    return output
}

// processEvents handles the debouncing logic.
func (d *Debouncer) processEvents(ctx context.Context, input <-chan FileChangeEvent, output chan<- []FileChangeEvent) {
    defer close(output)
    
    triggerChan := make(chan struct{})
    
    for {
        select {
        case <-ctx.Done():
            d.cancelTimer()
            return
            
        case event, ok := <-input:
            if !ok {
                // Input closed, flush remaining events
                d.flushEvents(output)
                return
            }
            
            d.addEvent(event, triggerChan)
            
        case <-triggerChan:
            // Timer fired, send batched events
            d.mu.Lock()
            if len(d.events) > 0 {
                eventsCopy := make([]FileChangeEvent, len(d.events))
                copy(eventsCopy, d.events)
                d.events = d.events[:0]
                d.mu.Unlock()
                
                select {
                case output <- eventsCopy:
                    d.logger.Debug("debounce triggered",
                        "eventCount", len(eventsCopy),
                    )
                case <-ctx.Done():
                    return
                }
            } else {
                d.mu.Unlock()
            }
        }
    }
}

// addEvent adds an event and resets the debounce timer.
func (d *Debouncer) addEvent(event FileChangeEvent, triggerChan chan struct{}) {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    // Add event to batch
    d.events = append(d.events, event)
    
    d.logger.Debug("event added to batch",
        "path", event.Path,
        "batchSize", len(d.events),
    )
    
    // Reset timer
    if d.timer != nil {
        d.timer.Stop()
    }
    
    d.timer = time.AfterFunc(d.config.Window, func() {
        select {
        case triggerChan <- struct{}{}:
        default:
            // Channel full, trigger already pending
        }
    })
}

// flushEvents sends any remaining events.
func (d *Debouncer) flushEvents(output chan<- []FileChangeEvent) {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    if len(d.events) > 0 {
        output <- d.events
        d.events = d.events[:0]
    }
}

// cancelTimer stops any pending timer.
func (d *Debouncer) cancelTimer() {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    if d.timer != nil {
        d.timer.Stop()
        d.timer = nil
    }
}

// Reset clears the event buffer without triggering.
func (d *Debouncer) Reset() {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    d.events = d.events[:0]
    if d.timer != nil {
        d.timer.Stop()
        d.timer = nil
    }
}
```

---

## Key Implementation Details

### 1. Timer Reset on Each Event

```go
// Stop existing timer
if d.timer != nil {
    d.timer.Stop()
}

// Start new timer
d.timer = time.AfterFunc(d.config.Window, func() {
    triggerChan <- struct{}{}
})
```

### 2. Event Batching

```go
d.mu.Lock()
d.events = append(d.events, event)
d.mu.Unlock()
```

### 3. Non-Blocking Trigger

```go
select {
case triggerChan <- struct{}{}:
default:
    // Channel full, trigger already pending
}
```

### 4. Clean Shutdown

```go
case <-ctx.Done():
    d.cancelTimer()
    return
```

---

## Testing

```go
// pkg/watch/debounce_test.go

package watch

import (
    "context"
    "testing"
    "time"
)

func TestDebouncer_BatchesEvents(t *testing.T) {
    config := DebounceConfig{Window: 100 * time.Millisecond}
    debouncer := NewDebouncer(config, &mockLogger{})
    
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
    debouncer := NewDebouncer(config, &mockLogger{})
    
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
    
    // Should have waited ~200ms after last event (~400ms total)
    if elapsed < 350*time.Millisecond {
        t.Errorf("debounce triggered too early: %v", elapsed)
    }
}

func TestDebouncer_SeparateBatches(t *testing.T) {
    config := DebounceConfig{Window: 50 * time.Millisecond}
    debouncer := NewDebouncer(config, &mockLogger{})
    
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
    debouncer := NewDebouncer(config, &mockLogger{})
    
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
    debouncer := NewDebouncer(config, &mockLogger{})
    
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
```

---

## Checklist for Task 5.2

- [ ] Create `pkg/watch/debounce.go`
- [ ] Define `DebounceConfig` struct
- [ ] Implement `DefaultDebounceConfig()` function
- [ ] Implement `Debouncer` struct
- [ ] Implement `NewDebouncer()` constructor
- [ ] Implement `Debounce()` method
- [ ] Implement `processEvents()` goroutine
- [ ] Implement `addEvent()` helper
- [ ] Implement `flushEvents()` helper
- [ ] Implement `cancelTimer()` helper
- [ ] Implement `Reset()` method
- [ ] Create `pkg/watch/debounce_test.go`
- [ ] Test event batching
- [ ] Test timer reset
- [ ] Test separate batches
- [ ] Test cancellation
- [ ] Run `go test ./pkg/watch -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Not using mutex
```go
// Wrong - race condition
d.events = append(d.events, event)

// Right - protected access
d.mu.Lock()
d.events = append(d.events, event)
d.mu.Unlock()
```

❌ **Mistake 2**: Blocking on trigger send
```go
// Wrong - can deadlock
triggerChan <- struct{}{}

// Right - non-blocking
select {
case triggerChan <- struct{}{}:
default:
}
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 5.3** → Implement Watch Orchestration

---

## References

- [time.AfterFunc](https://pkg.go.dev/time#AfterFunc)
- [Debouncing Pattern](https://en.wikipedia.org/wiki/Debouncing)

