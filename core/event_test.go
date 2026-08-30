package core

import "testing"

// The callback registry is package-global, so every test establishes its own
// pass boundary with BeginRenderPass before registering anything.

func TestCallbackIDsStableAcrossRenderPasses(t *testing.T) {
	BeginRenderPass()
	first := registerCallback(func() {})
	second := registerTextCallback(func(string) {})
	third := registerBoolCallback(func(bool) {})

	// Simulate the next render of the same UI: same registration order must
	// reproduce the same IDs, otherwise every interactive node's props would
	// differ between renders and the reconciler would patch all of them.
	BeginRenderPass()
	if got := registerCallback(func() {}); got != first {
		t.Errorf("plain callback ID changed across passes: %q then %q", first, got)
	}
	if got := registerTextCallback(func(string) {}); got != second {
		t.Errorf("text callback ID changed across passes: %q then %q", second, got)
	}
	if got := registerBoolCallback(func(bool) {}); got != third {
		t.Errorf("bool callback ID changed across passes: %q then %q", third, got)
	}
}

func TestCallbackKindsUseDistinctIDSpaces(t *testing.T) {
	// Each kind counts independently; IDs must never collide across the three
	// registries since TriggerCallback/TriggerTextCallback/TriggerBoolCallback
	// look up in separate maps keyed by these strings.
	BeginRenderPass()
	ids := map[string]bool{
		registerCallback(func() {}):           true,
		registerTextCallback(func(string) {}): true,
		registerBoolCallback(func(bool) {}):   true,
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 distinct IDs across kinds, got %v", ids)
	}
}

func TestReRegistrationRunsLatestClosure(t *testing.T) {
	// Correctness requirement behind the overwrite semantics: the closure from
	// the most recent render captures the current state slots, so a stable ID
	// must always dispatch to the newest registration.
	var ran string

	BeginRenderPass()
	id := registerCallback(func() { ran = "stale" })

	BeginRenderPass()
	if id2 := registerCallback(func() { ran = "fresh" }); id2 != id {
		t.Fatalf("expected re-registration at same position to reuse %q, got %q", id, id2)
	}

	TriggerCallback(id)
	if ran != "fresh" {
		t.Errorf("triggered closure = %q, want the latest registration", ran)
	}
}

func TestPurgeDropsCallbacksNotReRegistered(t *testing.T) {
	// Pass 1 renders two buttons; pass 2 renders one. After the purge, the
	// orphaned tail ID must be gone so a late event against it is a no-op
	// instead of firing a handler for a node that no longer exists.
	staleRan := false

	BeginRenderPass()
	keep := registerCallback(func() {})
	stale := registerCallback(func() { staleRan = true })

	BeginRenderPass()
	registerCallback(func() {})
	PurgeUnusedCallbacks()

	callbackMux.Lock()
	_, staleExists := callbacks[stale]
	_, keepExists := callbacks[keep]
	callbackMux.Unlock()

	if staleExists {
		t.Errorf("callback %q should have been purged after not re-registering", stale)
	}
	if !keepExists {
		t.Errorf("callback %q was re-registered this pass and must survive the purge", keep)
	}

	TriggerCallback(stale) // must be a silent no-op
	if staleRan {
		t.Errorf("purged callback still executed")
	}
}
