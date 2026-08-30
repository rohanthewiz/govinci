package core

import (
	"fmt"
	"sync"
)

type RenderManager struct {
	mu      sync.Mutex
	subs    map[string]func()
	counter int
}

func NewRenderManager() *RenderManager {
	return &RenderManager{
		subs: make(map[string]func()),
	}
}

func (r *RenderManager) RegisterRender(fn func()) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := fmt.Sprintf("render_%d", r.counter)
	r.counter++
	r.subs[id] = fn
	return id
}

func (r *RenderManager) TriggerRender(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if fn, ok := r.subs[id]; ok {
		go fn() // async
	}
}

// defaultRenderTarget is the subscription key state mutations notify. It was
// historically hardcoded in State.Set with no matching registration (IDs from
// RegisterRender are generated as "render_N"), so state-change notifications
// went nowhere; OnStateChange below is the registration side that completes
// the circuit.
const defaultRenderTarget = "default"

// OnStateChange registers fn to run whenever state anywhere in this context
// tree is written (State.Set, or anything else calling RequestRender). Only
// one handler is held: a render driver like render.Manager owns re-rendering
// for the whole app, so later registrations replace earlier ones rather than
// fanning out duplicate render passes.
//
// fn is invoked on a fresh goroutine per notification (see TriggerRender), so
// it must be safe to call concurrently and should be cheap — the intended
// pattern is a non-blocking nudge into a coalescing channel, not a render.
func (ctx *Context) OnStateChange(fn func()) {
	ctx.renderManager.mu.Lock()
	defer ctx.renderManager.mu.Unlock()
	ctx.renderManager.subs[defaultRenderTarget] = fn
}

// RequestRender marks the tree dirty and notifies the registered render
// driver. This is the one entry point for "state changed, the UI should
// re-render" — used by State.Set and by async sources such as timers, so
// changes that happen outside a native event (where no bridge call is pending
// a response) can still reach the screen via the push channel.
func (ctx *Context) RequestRender() {
	ctx.MarkDirty()
	ctx.renderManager.TriggerRender(defaultRenderTarget)
}

func (ctx *Context) SubscribeRender(fn func()) {
	ctx.renderManager.RegisterRender(fn)
}
