package reconcile

import (
	"fmt"

	"github.com/GraHms/govinci/core"
)

// Patch represents a minimal change set between two Node trees
type Patch struct {
	Type     string      // e.g., "replace", "update", "reorder"
	TargetID string      // Node ID or unique path
	Changes  interface{} // could be Props, Style, Children diff
}

// Diff compares two Node trees and returns a list of patches
func Diff(old, new *core.Node, path string) []Patch {
	if old == nil && new != nil {
		return []Patch{{
			Type:     "add",
			TargetID: path,
			Changes:  new,
		}}
	}
	if new == nil && old != nil {
		return []Patch{{
			Type:     "remove",
			TargetID: path,
		}}
	}
	if old.Type != new.Type {
		return []Patch{{
			Type:     "replace",
			TargetID: path,
			Changes:  new,
		}}
	}

	patches := []Patch{}

	if propsChanged(old.Props, new.Props) {
		patches = append(patches, Patch{
			Type:     "update-props",
			TargetID: path,
			Changes:  new.Props,
		})
	}
	if styleChanged(old.Style, new.Style) {
		patches = append(patches, Patch{
			Type:     "update-style",
			TargetID: path,
			Changes:  new.Style,
		})
	}

	// Keyed child reconciliation
	oldKeyed := make(map[string]*core.Node)
	newKeyed := make(map[string]*core.Node)
	for _, c := range old.Children {
		if c.Key != "" {
			oldKeyed[c.Key] = c
		}
	}
	for _, c := range new.Children {
		if c.Key != "" {
			newKeyed[c.Key] = c
		}
	}

	minLen := min(len(old.Children), len(new.Children))
	for i := 0; i < minLen; i++ {
		childPath := path + "/" + itoa(i)
		oldChild := old.Children[i]
		newChild := new.Children[i]

		if oldChild.Key != "" && newChild.Key != "" && oldChild.Key != newChild.Key {
			// They have keys but don't match. Check if they moved.
			// For simplicity in this naive implementation, we emit a replace if keys don't match exactly by index.
			// A true React reconciler would emit 'move' patches, but for this framework, a full replace solves the error state.
			patches = append(patches, Patch{
				Type:     "replace",
				TargetID: childPath,
				Changes:  newChild,
			})
		} else {
			patches = append(patches, Diff(oldChild, newChild, childPath)...)
		}
	}
	if len(old.Children) < len(new.Children) {
		if len(old.Children) < len(new.Children) {
			for i := len(old.Children); i < len(new.Children); i++ {
				patches = append(patches, Patch{
					Type:     "add-child",
					TargetID: path,
					Changes:  new.Children[i],
				})
			}
		}

	}
	if len(old.Children) > len(new.Children) {
		for i := len(new.Children); i < len(old.Children); i++ {
			childPath := path + "/" + itoa(i)
			patches = append(patches, Patch{
				Type:     "remove-child",
				TargetID: childPath,
			})
		}
	}

	return patches
}

func propsChanged(a, b map[string]any) bool {
	if len(a) != len(b) {
		return true
	}
	for k, v := range a {
		if b[k] != v {
			return true
		}
	}
	return false
}

func styleChanged(a, b *core.Style) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return a != b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
