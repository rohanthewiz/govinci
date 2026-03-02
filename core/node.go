package core

type Node struct {
	Type     string
	Key      string
	Props    map[string]any
	Style    *Style
	Children []*Node
}
