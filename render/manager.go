package render

import (
	"encoding/json"
	"sync"

	"github.com/GraHms/govinci/core"
	"github.com/GraHms/govinci/reconcile"
)

// PatchListener is the Go→native push channel: the native shell implements it
// and registers via Manager.SetListener, and Go calls ApplyPatches whenever
// state changes outside a native event — timer ticks, network responses, any
// goroutine calling State.Set. Without it the bridge is strictly
// request/response and async updates never reach the screen (WASM worked
// around this by polling IsDirty).
//
// The single string-parameter method is deliberate: gomobile bind maps this
// interface onto Java/Kotlin and Objective-C/Swift directly, so an Android
// Activity or iOS view controller can implement it without JNI glue.
//
// ApplyPatches is invoked from a background goroutine. Native implementations
// must hop to their UI thread before touching views (runOnUiThread /
// DispatchQueue.main); the payload is the same patch-array JSON RenderAgain
// returns, and an empty patch set is never pushed.
type PatchListener interface {
	ApplyPatches(patches string)
}

type Manager struct {
	// mu serializes every render pass. Render passes mutate shared state that
	// cannot tolerate interleaving — the context's hook cursor, the global
	// callback registry (BeginRenderPass resets its counters), and
	// currentTree — and passes are started from two directions: the native
	// event path (bridge calls RenderAgain after dispatching a callback) and
	// the push pump below.
	mu          sync.Mutex
	currentTree *core.Node
	context     *core.Context
	renderFunc  func(*core.Context) core.View

	listenerMu sync.Mutex
	listener   PatchListener

	// renderRequests carries coalesced "state changed" nudges to the pump.
	//
	//   State.Set ──┐                       ┌──> RenderAgain ──> listener
	//   timer tick ─┼─> requestRender ──▷──┤      (mu held)
	//   State.Set ──┘   (buffer of 1,       └──> nothing pending: park
	//                    extra nudges
	//                    dropped)
	//
	// The buffer size of 1 is the coalescing mechanism: a burst of N rapid
	// state writes leaves at most one pending token, and the single pump
	// render that consumes it sees the final state — one diff, one push,
	// instead of N. A nudge arriving mid-render lands in the buffer and
	// triggers one follow-up pass, so the last write is never lost.
	renderRequests chan struct{}
	stop           chan struct{}
	stopOnce       sync.Once
}

func New(ctx *core.Context, rootView func(*core.Context) core.View) *Manager {
	if ctx.Theme() == nil {
		ctx = ctx.WithTheme(core.DefaultTheme)
	}
	m := &Manager{
		context:        ctx,
		renderFunc:     rootView,
		renderRequests: make(chan struct{}, 1),
		stop:           make(chan struct{}),
	}
	// Complete the notification circuit: State.Set / RequestRender fire the
	// context's default render target, which now nudges this manager's pump.
	ctx.OnStateChange(m.requestRender)
	go m.pump()
	return m
}

// Close stops the push pump. The Manager is normally an app-lifetime
// singleton, so this mainly matters for tests and hot-reload hosts.
func (m *Manager) Close() {
	m.stopOnce.Do(func() { close(m.stop) })
}

// SetListener attaches (or replaces) the native push target. Any state change
// that happened before attachment is flushed immediately so the listener
// never starts out behind.
func (m *Manager) SetListener(l PatchListener) {
	m.listenerMu.Lock()
	m.listener = l
	m.listenerMu.Unlock()
	m.requestRender()
}

func (m *Manager) getListener() PatchListener {
	m.listenerMu.Lock()
	defer m.listenerMu.Unlock()
	return m.listener
}

// requestRender is the cheap, non-blocking nudge described on renderRequests.
// Safe to call from any goroutine, including from within a render pass.
func (m *Manager) requestRender() {
	select {
	case m.renderRequests <- struct{}{}:
	default: // a render is already pending; this write will be included in it
	}
}

// pump is the single consumer of renderRequests: it turns state-change nudges
// into render passes and pushes non-empty diffs to the listener. One goroutine
// per Manager, started in New, stopped by Close.
func (m *Manager) pump() {
	for {
		select {
		case <-m.stop:
			return
		case <-m.renderRequests:
			// Nothing is mounted before the initial render, and with no
			// listener a pump render would consume the diff and discard it —
			// leaving a polling runtime (which calls RenderAgain itself) with
			// an empty diff and a stale screen. In both cases leave the dirty
			// flag standing; SetListener re-nudges, so nothing is lost.
			if m.getListener() == nil || !m.hasInitialRender() {
				continue
			}
			out := m.RenderAgain()
			if out == "[]" {
				continue
			}
			if l := m.getListener(); l != nil {
				l.ApplyPatches(out)
			}
		}
	}
}

func (m *Manager) hasInitialRender() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentTree != nil
}

func (r *Manager) RenderInitial() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	core.BeginRenderPass()
	r.context.Reset()
	r.currentTree = r.renderFunc(r.context).Render(r.context)
	return renderJSON(r.currentTree)
}

// RenderAgain ReRender Used after an event (input/click/state change) to get diff
func (r *Manager) RenderAgain() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	// BeginRenderPass must precede the render so callback IDs restart from
	// zero and re-registrations line up with the previous pass; the purge
	// after the diff then drops only IDs no longer registered this pass.
	core.BeginRenderPass()
	r.context.Reset()
	newTree := r.renderFunc(r.context).Render(r.context)
	patches := reconcile.Diff(r.currentTree, newTree, "root")
	r.currentTree = newTree
	r.context.ClearDirty()
	core.PurgeUnusedCallbacks()
	if patches == nil {
		// A no-change render must serialize as "[]", not "null": the native
		// runtimes iterate the decoded patch list without a null check.
		patches = []reconcile.Patch{}
	}
	return renderJSON(patches)
}

// JSON encoder
func renderJSON[T any](v T) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"failed to encode JSON"}`
	}
	return string(data)
}

func (r *Manager) RenderAndGetPatches() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	core.BeginRenderPass()
	r.context.Cursor = 0
	newTree := r.renderFunc(r.context).Render(r.context)

	if r.currentTree == nil {
		r.currentTree = newTree
		return render(newTree)
	}

	patches := reconcile.Diff(r.currentTree, newTree, "root")
	r.currentTree = newTree
	if patches == nil {
		patches = []reconcile.Patch{}
	}
	return render(patches)
}

func render[T any](tree T) string {
	data, err := json.Marshal(tree)
	if err != nil {
		return `{"error":"failed to encode render tree"}`
	}
	return string(data)
}
