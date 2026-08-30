package core

import (
	"fmt"
	"strconv"
)

func Input(value string, placeholder string, onChange func(string), styleProps ...StyleProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		base := ctx.Theme().Components.Input
		style := &base
		for _, sp := range styleProps {
			sp.Apply(style)
		}

		id := ctx.registerTextCallback(onChange)

		return &Node{
			Type: "Input",
			Props: map[string]any{
				"value":       value,
				"placeholder": placeholder,
				"onChange":    id,
			},
			Style: style,
		}
	})
}

func Checkbox(checked bool, onToggle func(bool), styleProps ...StyleProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		base := ctx.Theme().Components.CheckBox
		style := &base
		for _, sp := range styleProps {
			sp.Apply(style)
		}

		id := ctx.registerBoolCallback(onToggle)

		return &Node{
			Type: "Checkbox",
			Props: map[string]any{
				"checked":  checked,
				"onToggle": id,
			},
			Style: style,
		}
	})
}

func InputPassword(value string, placeholder string, onChange func(string), styleProps ...StyleProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		base := ctx.Theme().Components.Input
		style := &base
		for _, sp := range styleProps {
			sp.Apply(style)
		}

		id := ctx.registerTextCallback(onChange)

		return &Node{
			Type: "InputPassword",
			Props: map[string]any{
				"value":       value,
				"placeholder": placeholder,
				"onChange":    id,
			},
			Style: style,
		}
	})
}

func NumericInput(value int, onChange func(int), styleProps ...StyleProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		base := ctx.Theme().Components.Input
		style := &base
		for _, sp := range styleProps {
			sp.Apply(style)
		}

		id := ctx.registerTextCallback(func(val string) {
			if n, err := strconv.Atoi(val); err == nil {
				onChange(n)
			}
		})

		return &Node{
			Type: "NumericInput",
			Props: map[string]any{
				"value":    fmt.Sprintf("%d", value),
				"onChange": id,
			},
			Style: style,
		}
	})
}

func TextArea(value string, onChange func(string), rows int, styleProps ...StyleProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		base := ctx.Theme().Components.TextArea
		style := &base
		for _, sp := range styleProps {
			sp.Apply(style)
		}

		id := ctx.registerTextCallback(onChange)

		return &Node{
			Type: "TextArea",
			Props: map[string]any{
				"value":    value,
				"rows":     rows,
				"onChange": id,
			},
			Style: style,
		}
	})
}
