package core

type View interface {
	Render(ctx *Context) *Node
}

type ComponentFunc func(ctx *Context) *Node

func (f ComponentFunc) Render(ctx *Context) *Node {
	return f(ctx)
}

func Keyed(key string, child View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		n := child.Render(ctx)
		n.Key = key
		return n
	})
}
