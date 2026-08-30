package core

import "sync"

// cleanupRegistry collects stop functions for background resources started on
// behalf of a context tree — interval tickers, pending timeouts, anything a
// hook spins up that outlives the render pass that created it. One registry
// exists per NewContext root and is shared by pointer with every derived
// context (children, scopes, WithTheme/WithConfig copies), the same pattern
// as renderManager and the callback registry: resources registered anywhere
// in the tree are stopped together, and two apps in one process can never
// stop each other's.
//
// Close has drain semantics rather than terminal semantics: it runs and
// forgets the functions registered so far, but the registry stays usable.
// That is deliberate — a host that re-mounts an app over the same context
// (the WASM runtime re-invoking RenderInitial, a hot-reload harness) closes
// the old manager and renders again, and the re-render's hooks must be able
// to register fresh resources for the next Close to stop.
type cleanupRegistry struct {
	mu  sync.Mutex
	fns []func()
}

func newCleanupRegistry() *cleanupRegistry {
	return &cleanupRegistry{}
}

// OnClose registers fn to run when this context tree is closed. Hooks use it
// to hand ownership of their background resources to whoever drives the app's
// lifecycle (normally render.Manager, whose Close closes its context).
func (ctx *Context) OnClose(fn func()) {
	r := ctx.cleanup
	r.mu.Lock()
	r.fns = append(r.fns, fn)
	r.mu.Unlock()
}

// Close stops every background resource registered on this context tree since
// the last Close (see the drain semantics on cleanupRegistry). The tree itself
// remains renderable afterwards; a subsequent render pass simply re-registers
// whatever resources it still needs.
//
// The registered functions run outside the registry lock: a cleanup that
// itself registers or closes (however unlikely) must not deadlock, mirroring
// the dispatch-outside-the-lock rule the callback registry follows.
func (ctx *Context) Close() {
	r := ctx.cleanup
	r.mu.Lock()
	fns := r.fns
	r.fns = nil
	r.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}
