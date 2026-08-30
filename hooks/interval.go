package hooks

import (
	"fmt"
	"github.com/GraHms/govinci/core"
	"sync"
	"time"
)

var intervalStore = struct {
	mu     sync.Mutex
	active map[string]*time.Ticker
}{
	active: make(map[string]*time.Ticker),
}

func UseInterval(ctx *core.Context, fn func(), interval time.Duration) {
	index := ctx.Cursor
	// Reserve this hook's cursor position through NewState rather than a bare
	// Cursor++: NewState keeps the slot array and the cursor aligned. A bare
	// increment leaves no slot behind, so every NewState after this hook in
	// the same render appends its backing slot one index short of its cursor —
	// from then on all later hooks read each other's slots (e.g. a bool
	// Checkbox state landing where a string Input state is read, which
	// panics on the type assertion).
	core.NewState(ctx, struct{}{})

	key := fmt.Sprintf("interval-%d", index)

	useIntervalInternal(ctx, key, fn, interval)
}

func useIntervalInternal(ctx *core.Context, key string, fn func(), interval time.Duration) {
	intervalStore.mu.Lock()
	defer intervalStore.mu.Unlock()

	if _, exists := intervalStore.active[key]; exists {
		return
	}

	ticker := time.NewTicker(interval)
	intervalStore.active[key] = ticker

	go func() {
		for range ticker.C {
			fn()
			// RequestRender (not just MarkDirty) so a timer tick reaches the
			// screen through the push channel even when no native event is in
			// flight to piggyback a re-render on.
			ctx.RequestRender()
		}
	}()
}
func ClearIntervals() {
	intervalStore.mu.Lock()
	defer intervalStore.mu.Unlock()

	for key, ticker := range intervalStore.active {
		ticker.Stop()
		delete(intervalStore.active, key)
	}
}

var timeoutStore = struct {
	mu     sync.Mutex
	active map[string]bool
}{
	active: make(map[string]bool),
}

func UseTimeout(ctx *core.Context, fn func(), delay time.Duration) {
	index := ctx.Cursor
	// Same slot-reservation rationale as UseInterval above.
	core.NewState(ctx, struct{}{})

	key := fmt.Sprintf("timeout-%d", index)

	timeoutStore.mu.Lock()
	if timeoutStore.active[key] {
		timeoutStore.mu.Unlock()
		return
	}
	timeoutStore.active[key] = true
	timeoutStore.mu.Unlock()

	go func() {
		time.Sleep(delay)
		fn()
		ctx.MarkDirty()
		timeoutStore.mu.Lock()
		delete(timeoutStore.active, key)
		timeoutStore.mu.Unlock()
	}()
}
