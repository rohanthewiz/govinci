package core

// BehaviorProp attaches event behavior to a node. Apply takes the rendering
// Context (unlike StyleProp) because registering the handler needs the
// context's callback registry — the registry is per-app state on the context
// tree, not a package global.
type BehaviorProp interface {
	Apply(*Context, *Node)
}

type behaviorFunc func(*Context, *Node)

func (f behaviorFunc) Apply(ctx *Context, n *Node) {
	f(ctx, n)
}
func On(event string, handler func()) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["on"+event] = ctx.registerCallback(handler)
	})
}

func OnClick(handler func()) BehaviorProp {
	return On("Click", handler)
}

func OnTouch(handler func()) BehaviorProp {
	return On("Touch", handler)
}

// OnLongPress fires after the platform's long-press timeout (~500ms) without
// the finger lifting. A node may carry both OnClick and OnLongPress: the
// renderers wire them as one gesture recognizer (combinedClickable on
// Android, tap+longPress on iOS), so a long press never also fires the click.
func OnLongPress(handler func()) BehaviorProp {
	return On("LongPress", handler)
}
