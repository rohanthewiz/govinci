package core

import (
	"sync"
)

type Context struct {
	slots           []any
	Cursor          int
	theme           *Theme
	config          *AppConfig
	idGen           int
	lock            sync.Mutex
	renderManager   *RenderManager
	callbackMap     map[string]any // Stable ID to callback
	callbackCounter *int
	usedCallbacks   map[string]bool
	dirty           bool
	parent          *Context

	children       []*Context
	childrenCursor int
	scopes         map[string]*Context
}

func (ctx *Context) MarkDirty() {
	ctx.dirty = true
}

func (ctx *Context) IsDirty() bool {
	return ctx.dirty
}

func (ctx *Context) ClearDirty() {
	ctx.dirty = false
}

type AppConfig struct {
	Name        string
	Description string
	Version     string
	Locale      string
	Author      string
	Meta        map[string]string
}

func NewContext() *Context {
	cc := 0
	return &Context{
		slots:           make([]any, 0),
		Cursor:          0,
		renderManager:   NewRenderManager(),
		callbackMap:     make(map[string]any),
		callbackCounter: &cc,
		scopes:          make(map[string]*Context),
	}
}
func (ctx *Context) NewChildContext() *Context {
	return &Context{
		slots:           make([]any, 0),
		Cursor:          0,
		theme:           ctx.theme,
		config:          ctx.config,
		renderManager:   ctx.renderManager,
		callbackMap:     ctx.callbackMap,
		callbackCounter: ctx.callbackCounter,
		parent:          ctx,
		scopes:          make(map[string]*Context),
	}
}
func UseChildContext(ctx *Context) *Context {
	index := ctx.Cursor
	ctx.Cursor++

	if index >= len(ctx.slots) {
		ctx.slots = append(ctx.slots, ctx.NewChildContext())
	}
	return ctx.slots[index].(*Context)
}

type State[T any] struct {
	get func() T
	set func(T)
}

func (s *State[T]) Get() T {
	return s.get()
}

func (s *State[T]) Set(val T) {
	s.set(val)
}

func (ctx *Context) Theme() *Theme {
	if ctx.theme != nil {
		return ctx.theme
	}
	return DefaultTheme // fallback
}

func (ctx *Context) Config() *AppConfig {
	if ctx.config == nil {
		return &AppConfig{}
	}
	return ctx.config
}

func (ctx *Context) WithConfig(cfg *AppConfig) *Context {
	return &Context{
		slots:           ctx.slots,
		Cursor:          ctx.Cursor,
		theme:           ctx.theme,
		config:          cfg,
		renderManager:   ctx.renderManager,
		callbackMap:     ctx.callbackMap,
		callbackCounter: ctx.callbackCounter,
	}
}

func (ctx *Context) WithTheme(theme *Theme) *Context {
	return &Context{
		slots:           ctx.slots,
		Cursor:          ctx.Cursor,
		theme:           theme,
		config:          ctx.config,
		renderManager:   ctx.renderManager,
		callbackMap:     ctx.callbackMap,
		callbackCounter: ctx.callbackCounter,
	}
}

func NewState[T any](ctx *Context, initial T) State[T] {
	index := ctx.Cursor
	ctx.Cursor++

	if index >= len(ctx.slots) {
		//log.Printf("Allocating slot %d with value: %#v", index, initial)
		ctx.slots = append(ctx.slots, initial)
	} else {
		//log.Printf("Reusing slot %d with value: %#v", index, ctx.slots[index])
	}

	return State[T]{
		get: func() T {
			return ctx.slots[index].(T)
		},
		set: func(val T) {
			//log.Printf("Updating slot %d with value: %#v", index, val)
			ctx.slots[index] = val
			// RequestRender rather than a bare TriggerRender: it also marks
			// the tree dirty, so polling runtimes (WASM IsDirty) and the push
			// channel observe the same signal.
			ctx.RequestRender()
		},
	}
}

func (ctx *Context) With(opts ...func(*Context)) *Context {
	for _, fn := range opts {
		fn(ctx)
	}
	return ctx
}

func WithThemeOpt(t *Theme) func(*Context) {
	return func(ctx *Context) {
		ctx.theme = t
	}
}

func WithConfigOpt(c *AppConfig) func(*Context) {
	return func(ctx *Context) {
		ctx.config = c
	}
}

func (ctx *Context) Reset() {
	ctx.Cursor = 0
	ctx.usedCallbacks = make(map[string]bool)
	for _, child := range ctx.children {
		child.Reset()
	}
	for _, child := range ctx.slots {
		if c, ok := child.(*Context); ok {
			c.Reset()
		}
	}
	for _, scope := range ctx.scopes {
		scope.Reset()
	}
}

func (ctx *Context) Scope(key string) *Context {
	if child, ok := ctx.scopes[key]; ok {
		return child
	}
	child := ctx.NewChildContext()
	ctx.scopes[key] = child
	return child
}
