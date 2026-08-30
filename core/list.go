package core

// List is the virtualized sibling of Column: a vertically scrolling container
// whose children are laid out lazily by the native renderer (Compose
// LazyColumn, SwiftUI LazyVStack), so a thousand-row feed composes only the
// rows on screen. Column + Scroll remains the right choice for short content;
// List is for long, data-driven collections.
//
// Give every child a stable identity with Keyed(id, ...) — the native lazy
// containers use the key to keep row state (and recycled views) attached to
// the same data across insertions, removals, and reorders. Unkeyed children
// fall back to positional identity, which behaves like Column but loses row
// state on reorder.
//
// It shares Column's theme base and the standard container argument contract:
// style props, behavior props (e.g. OnClick on the list surface), and child
// views in any order.
func List(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "List", ctx.Theme().Components.Column, stylePropsAndChildren)
	})
}
