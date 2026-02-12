package watch

import (
	"context"
	"sync"
	"time"

	"github.com/nanaki-93/kudev/pkg/logging"
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
	config DebounceConfig
	logger logging.LoggerInterface

	mu     sync.Mutex
	timer  *time.Timer
	events []FileChangeEvent
}

// NewDebouncer creates a new debouncer.
func NewDebouncer(config DebounceConfig, logger logging.LoggerInterface) *Debouncer {
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
