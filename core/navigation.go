package core

import "sync"

// navigatorState is the per-app route stack. It hangs off the context tree
// (one instance per NewContext root, shared by all derived contexts) rather
// than living in a package variable, so two apps in one process each navigate
// independently.
//
// The mutex matters because mutation and consumption run on different
// goroutines: Push/Pop are called from event handlers (dispatched on the
// native event path or under the render manager's dispatch lock) while
// Navigator reads the top of the stack during render passes on the pump
// goroutine. The old global slice had no synchronization at all.
type navigatorState struct {
	mu    sync.Mutex
	stack []func(*Context) View
}

func newNavigatorState() *navigatorState {
	return &navigatorState{stack: make([]func(*Context) View, 0)}
}

// current seeds the stack with initial on first use, then returns the top.
func (n *navigatorState) current(initial func(*Context) View) func(*Context) View {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.stack) == 0 {
		n.stack = append(n.stack, initial)
	}
	return n.stack[len(n.stack)-1]
}

func Navigator(initial func(*Context) View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		current := ctx.nav.current(initial)
		return Render(ctx, current(ctx))
	})
}

func Push(ctx *Context, route func(*Context) View) {
	ctx.nav.mu.Lock()
	ctx.nav.stack = append(ctx.nav.stack, route)
	ctx.nav.mu.Unlock()
	ctx.MarkDirty()
}

func Pop(ctx *Context) {
	ctx.nav.mu.Lock()
	popped := len(ctx.nav.stack) > 1 // the root route is never popped
	if popped {
		ctx.nav.stack = ctx.nav.stack[:len(ctx.nav.stack)-1]
	}
	ctx.nav.mu.Unlock()
	if popped {
		ctx.MarkDirty()
	}
}

func Replace(ctx *Context, route func(*Context) View) {
	ctx.nav.mu.Lock()
	replaced := len(ctx.nav.stack) > 0
	if replaced {
		ctx.nav.stack[len(ctx.nav.stack)-1] = route
	}
	ctx.nav.mu.Unlock()
	if replaced {
		ctx.MarkDirty()
	}
}

func Reset(ctx *Context, route func(*Context) View) {
	ctx.nav.mu.Lock()
	ctx.nav.stack = []func(*Context) View{route}
	ctx.nav.mu.Unlock()
	ctx.MarkDirty()
}

func Render(ctx *Context, view View) *Node {
	ctx.Reset()
	return view.Render(ctx)
}
