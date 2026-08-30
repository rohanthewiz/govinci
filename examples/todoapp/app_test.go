package todoapp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/GraHms/govinci/mobile"
)

// The package init has already run mobile.Register by the time tests execute,
// so these tests exercise exactly the call sequence the native shells make:
// read the current tree, dispatch a callback ID found in it, assert on the
// patches (or the next full tree). The bridge is a process-wide singleton, so
// the whole user journey lives in one ordered test rather than independent
// tests that would fight over shared state.

type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

// findNode depth-first searches for the first node satisfying pred. Predicates
// match on type plus identifying props (a Button's label) because callback IDs
// are per-pass sequence numbers — position in the tree, not identity, is the
// only stable way to locate a widget.
func findNode(n *node, pred func(*node) bool) *node {
	if n == nil {
		return nil
	}
	if pred(n) {
		return n
	}
	for _, c := range n.Children {
		if found := findNode(c, pred); found != nil {
			return found
		}
	}
	return nil
}

// currentTree re-renders and parses the full tree. Every dispatch below reads
// its callback ID from a fresh tree — the "dispatch only IDs from the live
// tree" discipline the shells follow, since any state change can renumber IDs.
func currentTree(t *testing.T) *node {
	t.Helper()
	var tree node
	if err := json.Unmarshal([]byte(mobile.RenderInitial()), &tree); err != nil {
		t.Fatalf("tree is not valid JSON: %v", err)
	}
	return &tree
}

// mustCallback locates a widget in the live tree and returns the named
// callback ID, failing the test if the widget or the prop is missing.
func mustCallback(t *testing.T, pred func(*node) bool, prop, desc string) string {
	t.Helper()
	n := findNode(currentTree(t), pred)
	if n == nil {
		t.Fatalf("no node in current tree matching: %s", desc)
	}
	id, ok := n.Props[prop].(string)
	if !ok {
		t.Fatalf("node %s has no %q callback prop: %#v", desc, prop, n.Props)
	}
	return id
}

func byType(nodeType string) func(*node) bool {
	return func(n *node) bool { return n.Type == nodeType }
}

func buttonLabeled(label string) func(*node) bool {
	return func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == label
	}
}

// addTodo drives the two-step entry flow: type into the controlled Input,
// then commit with the Add button.
func addTodo(t *testing.T, title string) {
	t.Helper()
	mobile.TriggerTextCallback(mustCallback(t, byType("Input"), "onChange", "Input"), title)
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Add"), "onClick", `Button "Add"`))
}

func TestTodoLifecycle(t *testing.T) {
	initial := mobile.RenderInitial()
	if !strings.Contains(initial, "No tasks yet") {
		t.Fatalf("initial tree missing empty state:\n%s", initial)
	}
	if !strings.Contains(initial, "0 items left") {
		t.Fatalf("initial tree missing zero count:\n%s", initial)
	}

	// Typing must patch the controlled Input's value back out — that
	// round-trip is what keeps the native field and Go state in sync.
	patches := mobile.TriggerTextCallback(
		mustCallback(t, byType("Input"), "onChange", "Input"), "Buy milk")
	if !strings.Contains(patches, "Buy milk") {
		t.Errorf("typing patches don't echo the draft value:\n%s", patches)
	}

	// Committing the draft must create the row, clear the input, and bump
	// the remaining count.
	patches = mobile.TriggerCallback(
		mustCallback(t, buttonLabeled("Add"), "onClick", `Button "Add"`))
	if !strings.Contains(patches, "Buy milk") {
		t.Errorf("add patches don't contain the new row:\n%s", patches)
	}
	if !strings.Contains(patches, "1 item left") {
		t.Errorf("add patches don't update the count:\n%s", patches)
	}

	// A blank draft must be a no-op: same tree before and after the tap.
	before := mobile.RenderInitial()
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Add"), "onClick", `Button "Add"`))
	if after := mobile.RenderInitial(); after != before {
		t.Errorf("adding a blank draft changed the tree:\nbefore: %s\nafter: %s", before, after)
	}

	addTodo(t, "Walk dog")
	if tree := mobile.RenderInitial(); !strings.Contains(tree, "2 items left") {
		t.Fatalf("second add didn't land:\n%s", tree)
	}

	// Bool event: checking the first row's checkbox marks "Buy milk" done,
	// which drops the remaining count and reveals the bulk-clear button.
	patches = mobile.TriggerBoolCallback(
		mustCallback(t, byType("Checkbox"), "onToggle", "Checkbox"), true)
	if !strings.Contains(patches, "1 item left") {
		t.Errorf("toggle patches don't update the count:\n%s", patches)
	}
	if tree := mobile.RenderInitial(); !strings.Contains(tree, "Clear completed") {
		t.Errorf("completing a task didn't reveal Clear completed:\n%s", tree)
	}

	// Filters are pure view state: Done shows only the completed todo,
	// Active only the open one, and the data itself is untouched.
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Done"), "onClick", `Button "Done"`))
	tree := mobile.RenderInitial()
	if !strings.Contains(tree, "Buy milk") || strings.Contains(tree, "Walk dog") {
		t.Errorf("Done filter shows the wrong rows:\n%s", tree)
	}

	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Active"), "onClick", `Button "Active"`))
	tree = mobile.RenderInitial()
	if !strings.Contains(tree, "Walk dog") || strings.Contains(tree, "Buy milk") {
		t.Errorf("Active filter shows the wrong rows:\n%s", tree)
	}

	// Deleting under the Active filter must remove "Walk dog" everywhere,
	// not just from the filtered view — the visible slice is derived, so the
	// row's closure has to address the todo by ID, not by position.
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("✕"), "onClick", `Button "✕"`))
	if tree = mobile.RenderInitial(); !strings.Contains(tree, "No active tasks") {
		t.Errorf("delete under Active filter didn't empty the view:\n%s", tree)
	}

	mobile.TriggerCallback(mustCallback(t, buttonLabeled("All"), "onClick", `Button "All"`))
	tree = mobile.RenderInitial()
	if strings.Contains(tree, "Walk dog") {
		t.Errorf("deleted todo still present in All view:\n%s", tree)
	}
	if !strings.Contains(tree, "Buy milk") {
		t.Errorf("delete removed the wrong todo:\n%s", tree)
	}

	// Bulk clear removes the remaining (done) todo and, with nothing done
	// anymore, must take its own button with it.
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Clear completed"), "onClick", `Button "Clear completed"`))
	tree = mobile.RenderInitial()
	if strings.Contains(tree, "Buy milk") {
		t.Errorf("Clear completed left a done todo behind:\n%s", tree)
	}
	if strings.Contains(tree, "Clear completed") {
		t.Errorf("Clear completed button lingers with nothing to clear:\n%s", tree)
	}
	if !strings.Contains(tree, "No tasks yet") {
		t.Errorf("emptied list doesn't show the empty state:\n%s", tree)
	}

	// Submit path: the input's onSubmit is a void callback carrying the same
	// commit as the Add button, so typing then dispatching it must create the
	// row and clear the draft without the button being involved.
	mobile.TriggerTextCallback(mustCallback(t, byType("Input"), "onChange", "Input"), "Via enter")
	patches = mobile.TriggerCallback(mustCallback(t, byType("Input"), "onSubmit", "Input onSubmit"))
	if !strings.Contains(patches, "Via enter") {
		t.Errorf("submit patches don't contain the new row:\n%s", patches)
	}
	if !strings.Contains(patches, `"value":""`) {
		t.Errorf("submit patches don't clear the draft:\n%s", patches)
	}
	if !strings.Contains(patches, "1 item left") {
		t.Errorf("submit patches don't update the count:\n%s", patches)
	}
}
