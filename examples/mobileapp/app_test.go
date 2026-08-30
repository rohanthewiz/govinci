package mobileapp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/GraHms/govinci/mobile"
)

// The package init has already run mobile.Register by the time tests execute,
// so these tests exercise exactly the call sequence the Kotlin shell makes.

type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

// findProp depth-first searches the tree for the first node of the given type
// and returns the named prop (callback IDs, values).
func findProp(n *node, nodeType, prop string) (string, bool) {
	if n == nil {
		return "", false
	}
	if n.Type == nodeType {
		if v, ok := n.Props[prop].(string); ok {
			return v, true
		}
	}
	for _, c := range n.Children {
		if v, ok := findProp(c, nodeType, prop); ok {
			return v, true
		}
	}
	return "", false
}

func TestDemoAppRendersAndDispatchesEvents(t *testing.T) {
	initial := mobile.RenderInitial()

	var tree node
	if err := json.Unmarshal([]byte(initial), &tree); err != nil {
		t.Fatalf("initial tree is not valid JSON: %v", err)
	}
	if !strings.Contains(initial, "Count: 0") {
		t.Fatalf("initial tree missing counter text:\n%s", initial)
	}

	// Void event: tapping the Increment button must patch the counter text.
	onClick, ok := findProp(&tree, "Button", "onClick")
	if !ok {
		t.Fatal("no Button with onClick in initial tree")
	}
	patches := mobile.TriggerCallback(onClick)
	if !strings.Contains(patches, "Count: 1") {
		t.Errorf("click patches don't update the counter:\n%s", patches)
	}

	// Int event: switching tabs must re-render with the new selectedIndex.
	onTabChange, ok := findProp(&tree, "TabView", "onTabChange")
	if !ok {
		t.Fatal("no TabView with onTabChange in initial tree")
	}
	patches = mobile.TriggerIntCallback(onTabChange, 1)
	if !strings.Contains(patches, `"selectedIndex":1`) {
		t.Errorf("tab-change patches don't carry the new selection:\n%s", patches)
	}

	// Text event: typing into the Input must patch the greeting.
	// Re-read the tree first — the callback IDs may have shifted with the
	// tab switch above, and the shell always dispatches IDs from its
	// current tree, never a stale one.
	var current node
	if err := json.Unmarshal([]byte(mobile.RenderInitial()), &current); err != nil {
		t.Fatalf("re-rendered tree is not valid JSON: %v", err)
	}
	onChange, ok := findProp(&current, "Input", "onChange")
	if !ok {
		t.Fatal("no Input with onChange in tree")
	}
	patches = mobile.TriggerTextCallback(onChange, "Ada")
	if !strings.Contains(patches, "Hello, Ada!") {
		t.Errorf("input patches don't update the greeting:\n%s", patches)
	}
}
