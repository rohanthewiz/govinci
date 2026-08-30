package core

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

var (
	callbacks     = map[string]func(){}
	textCallbacks = map[string]func(string){}
	boolCallbacks = map[string]func(bool){}
	intCallbacks  = map[string]func(int){}
	callbackMux   sync.Mutex
	counter       int
	textCounter   int
	boolCounter   int
	intCounter    int

	usedCallbacks = map[string]bool{}
)

// BeginRenderPass resets the callback ID counters so IDs are assigned by
// render-pass sequence: the Nth callback registered in a pass is always
// "cb_N" (or "txt_cb_N"/"bool_cb_N" for its kind).
//
// This is what makes callback IDs stable across renders. Component trees are
// rebuilt from scratch on every render, and with monotonically increasing
// counters every button received a brand-new onClick ID each time — so the
// reconciler saw every interactive node's props as changed on every render,
// and renderers re-bound every listener. With per-pass sequence IDs, an
// unchanged UI re-registers the same IDs in the same order and produces zero
// prop diffs; registration simply overwrites the map entry with the latest
// closure, which is required for correctness anyway (the new closure captures
// the current state slots).
//
// The IDs have the same stability granularity as the reconciler's positional
// TargetID paths: a structural change that shifts later siblings also shifts
// their callback IDs, and the same nodes get update-props patches the
// positional differ would emit regardless. An event dispatched against a
// stale tree can therefore hit a re-used ID and run the wrong handler in the
// brief window around a structural re-render; identity-keyed IDs (planned
// with stable node identity) are the eventual fix.
//
// Must be called exactly once at the start of each render pass, before any
// component builders run. Renderers do this via render.Manager, not directly.
func BeginRenderPass() {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	counter = 0
	textCounter = 0
	boolCounter = 0
	intCounter = 0
	// Fresh liveness marks for this pass: only callbacks re-registered below
	// survive the post-render PurgeUnusedCallbacks.
	usedCallbacks = make(map[string]bool)
}

func registerCallback(fn func()) string {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	id := fmt.Sprintf("cb_%d", counter)
	counter++
	callbacks[id] = fn // overwrites last pass's closure at this position, keeping the freshest captures
	usedCallbacks[id] = true
	return id
}

func TriggerCallback(id string) {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	if fn, ok := callbacks[id]; ok {
		fn()
		usedCallbacks[id] = true
	}
}

func registerTextCallback(fn func(string)) string {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	id := fmt.Sprintf("txt_cb_%d", textCounter)
	textCounter++
	textCallbacks[id] = fn
	usedCallbacks[id] = true
	return id
}

func TriggerTextCallback(id string, val string) {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	if fn, ok := textCallbacks[id]; ok {
		fn(val)
		usedCallbacks[id] = true
	}
}

func registerBoolCallback(fn func(bool)) string {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	id := fmt.Sprintf("bool_cb_%d", boolCounter)
	boolCounter++
	boolCallbacks[id] = fn
	usedCallbacks[id] = true
	return id
}

func TriggerBoolCallback(id string, val bool) {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	if fn, ok := boolCallbacks[id]; ok {
		fn(val)
		usedCallbacks[id] = true
	}
}

func registerIntCallback(fn func(int)) string {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	id := fmt.Sprintf("int_cb_%d", intCounter)
	intCounter++
	intCallbacks[id] = fn
	usedCallbacks[id] = true
	return id
}

func TriggerIntCallback(id string, val int) {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	if fn, ok := intCallbacks[id]; ok {
		fn(val)
		usedCallbacks[id] = true
	}
}

func ReceiveEventPayload(payload map[string]any) {
	id, ok := payload["callback"].(string)
	if !ok {
		log.Println("callback ID inválido")
		return
	}
	println("updating callback", id)

	switch val := payload["value"].(type) {
	case string:
		// Tenta deserializar
		var parsed map[string]any
		if err := json.Unmarshal([]byte(val), &parsed); err == nil {
			if v, ok := parsed["value"].(string); ok {
				TriggerTextCallback(id, v)
				return
			}
			if b, ok := parsed["value"].(bool); ok {
				TriggerBoolCallback(id, b)
				return
			}
		}

		// Fallback: trata como string normal
		TriggerTextCallback(id, val)

	case bool:
		TriggerBoolCallback(id, val)
	case nil:
		TriggerCallback(id)
	default:
		TriggerCallback(id)
	}
}

func PurgeUnusedCallbacks() {
	callbackMux.Lock()
	defer callbackMux.Unlock()

	newCallbacks := make(map[string]func())
	newTextCallbacks := make(map[string]func(string))
	newBoolCallbacks := make(map[string]func(bool))
	newIntCallbacks := make(map[string]func(int))

	for id, fn := range callbacks {
		if usedCallbacks[id] {
			newCallbacks[id] = fn
		}
	}
	for id, fn := range textCallbacks {
		if usedCallbacks[id] {
			newTextCallbacks[id] = fn
		}
	}
	for id, fn := range boolCallbacks {
		if usedCallbacks[id] {
			newBoolCallbacks[id] = fn
		}
	}
	for id, fn := range intCallbacks {
		if usedCallbacks[id] {
			newIntCallbacks[id] = fn
		}
	}

	callbacks = newCallbacks
	textCallbacks = newTextCallbacks
	boolCallbacks = newBoolCallbacks
	intCallbacks = newIntCallbacks
	usedCallbacks = make(map[string]bool) // Clean up
}
