package render_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GraHms/govinci/core"
	"github.com/GraHms/govinci/hooks"
	"github.com/GraHms/govinci/render"
)

// chanListener adapts the PatchListener push into a channel so tests can wait
// on deliveries with a timeout instead of sleeping.
type chanListener struct{ ch chan string }

func newChanListener() *chanListener {
	// Generous buffer: the pump must never block on a slow listener in tests.
	return &chanListener{ch: make(chan string, 16)}
}

func (c *chanListener) ApplyPatches(patches string) { c.ch <- patches }

// awaitPush waits for a pushed patch payload satisfying pred.
func awaitPush(t *testing.T, l *chanListener, timeout time.Duration, pred func(string) bool) string {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case p := <-l.ch:
			if pred(p) {
				return p
			}
			// Not the payload we're waiting for (e.g. an intermediate
			// coalesced push) — keep draining until the deadline.
		case <-deadline:
			t.Fatalf("timed out after %v waiting for a matching push", timeout)
		}
	}
}

func TestStateChangePushesToListener(t *testing.T) {
	m := render.New(core.NewContext(), counterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	l := newChanListener()
	m.SetListener(l)

	// The event is dispatched WITHOUT the native side calling RenderAgain —
	// exactly the shape of an async state change. The push channel alone must
	// carry the update out.
	core.TriggerCallback(onClick)

	payload := awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, "count: 1")
	})
	var patches []jsonPatch
	if err := json.Unmarshal([]byte(payload), &patches); err != nil {
		t.Fatalf("pushed payload is not patch JSON: %v\n%s", err, payload)
	}
	if len(patches) != 1 || patches[0].Type != "update-props" || patches[0].TargetID != "root/0" {
		t.Errorf("pushed patches = %+v, want single update-props on root/0", patches)
	}
}

func TestRapidStateChangesCoalesce(t *testing.T) {
	m := render.New(core.NewContext(), counterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	l := newChanListener()
	m.SetListener(l)

	const clicks = 5
	for range clicks {
		core.TriggerCallback(onClick)
	}

	// The final state must arrive; the buffered-by-one request channel means
	// the burst arrives in at most `clicks` pushes (typically far fewer), and
	// nothing after the final value.
	awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, fmt.Sprintf("count: %d", clicks))
	})
	select {
	case p := <-l.ch:
		t.Errorf("unexpected push after final state was delivered: %s", p)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestNoListenerLeavesPollingFlowIntact(t *testing.T) {
	// Regression guard for the pump/polling interaction: with no listener the
	// pump must NOT consume the diff, or a polling runtime (WASM today) would
	// call RenderAgain itself and get "[]" while the screen is stale.
	m := render.New(core.NewContext(), counterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	core.TriggerCallback(onClick)
	// Give the pump a chance to (incorrectly) swallow the change.
	time.Sleep(50 * time.Millisecond)

	patches := decodePatches(t, m.RenderAgain())
	if len(patches) != 1 || patches[0].TargetID != "root/0" {
		t.Fatalf("polling RenderAgain lost the diff to the pump: %+v", patches)
	}
}

func TestListenerAttachedLateReceivesPendingChange(t *testing.T) {
	// A state change that happens before the native side attaches must be
	// flushed on attachment (SetListener re-nudges the pump).
	m := render.New(core.NewContext(), counterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	core.TriggerCallback(onClick)

	l := newChanListener()
	m.SetListener(l)
	awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, "count: 1")
	})
}

func TestIntervalTickPushesWithoutAnyNativeEvent(t *testing.T) {
	// The headline scenario for the push channel: a timer-driven UI updating
	// with no native event in flight to piggyback on.
	tickerApp := func(ctx *core.Context) core.View {
		return core.ComponentFunc(func(ctx *core.Context) *core.Node {
			count := core.NewState(ctx, 0)
			hooks.UseInterval(ctx, func() {
				count.Set(count.Get() + 1)
			}, 20*time.Millisecond)
			return core.Text(fmt.Sprintf("count: %d", count.Get())).Render(ctx)
		})
	}

	defer hooks.ClearIntervals() // the ticker store is global; stop it for later tests

	m := render.New(core.NewContext(), tickerApp)
	defer m.Close()

	l := newChanListener()
	m.SetListener(l)
	m.RenderInitial()

	awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, "update-props") && strings.Contains(p, "count:")
	})
}
