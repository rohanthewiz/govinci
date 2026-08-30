package core

import "fmt"

type PropsAndChildren any

// containerNode builds a container node of the given type from a mixed
// argument list of style props, behavior props, and child views. All the
// flex-style containers (Row, Column, Card, Box, List) share it so behavior
// props work uniformly: BehaviorProps apply to the node that is actually
// returned — an earlier version of Column applied them to a throwaway props
// map, so OnClick on a container silently never reached the renderer.
//
// Ordering contract: behavior props register their callbacks in argument
// order, before any child renders (children are collected during the loop
// but rendered after it) — so a container's own callback IDs always precede
// its children's within a render pass, regardless of argument interleaving.
func containerNode(ctx *Context, typ string, base Style, items []PropsAndChildren) *Node {
	style := &base
	n := &Node{Type: typ, Style: style}
	var children []View
	for _, item := range items {
		switch v := item.(type) {
		case StyleProp:
			v.Apply(style)
		case View:
			children = append(children, v)
		case BehaviorProp:
			v.Apply(ctx, n)
		}
	}
	n.Children = renderAll(ctx, children)
	return n
}

func Row(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "Row", ctx.Theme().Components.Row, stylePropsAndChildren)
	})
}

func Card(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "Card", ctx.Theme().Components.Card, stylePropsAndChildren)
	})
}

func Spacer(size int) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return &Node{
			Type: "Spacer",
			Props: map[string]any{
				"size": size,
			},
		}
	})
}

func Scroll(children ...View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		var nodes []*Node
		for _, child := range children {
			nodes = append(nodes, child.Render(ctx))
		}
		return &Node{
			Type:     "Scroll",
			Props:    map[string]any{},
			Children: nodes,
		}
	})
}

func SafeArea(child View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return &Node{
			Type:     "SafeArea",
			Props:    map[string]any{},
			Children: []*Node{child.Render(ctx)},
		}
	})
}

func Fragment(children ...View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		if len(children) == 1 {
			return children[0].Render(ctx)
		}
		return &Node{
			Type:     "Fragment",
			Children: renderAll(ctx, children),
		}
	})
}

func Column(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "Column", ctx.Theme().Components.Column, stylePropsAndChildren)
	})
}

func Box(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		// Box has no theme base by design: it is the unopinionated container.
		return containerNode(ctx, "Box", Style{}, stylePropsAndChildren)
	})
}
func Divider(height int, color string) View {
	return Box(
		Height(fmt.Sprintf("%dpx", height)),
		BackgroundColor(color),
		Margin(8),
	)
}
func BorderColor(hex string) StyleProp {
	return styleFunc(func(s *Style) {
		s.BorderColor = hex
	})
}
func BorderWidth(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.BorderWidth = px
	})
}

func renderAll(ctx *Context, views []View) []*Node {
	var out []*Node
	for _, v := range views {
		out = append(out, v.Render(ctx))
	}
	return out
}
