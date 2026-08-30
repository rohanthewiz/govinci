package core

func Button(label string, onClick func(), styleProps ...StyleProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		base := ctx.Theme().Components.Button
		style := &base
		for _, sp := range styleProps {
			sp.Apply(style)
		}

		return &Node{
			Type: "Button",
			Props: map[string]any{
				"label":   label,
				"onClick": ctx.registerCallback(onClick),
			},
			Style: style,
		}
	})
}

func ButtonWithEvent(label string, event string, handler func(), styleProps ...StyleProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		base := ctx.Theme().Components.Button
		style := &base
		for _, sp := range styleProps {
			sp.Apply(style)
		}

		props := map[string]any{
			"label": label,
		}
		props["on"+event] = ctx.registerCallback(handler)

		return &Node{
			Type:  "Button",
			Props: props,
			Style: style,
		}
	})
}
