# Building a Todo App with Govinci — an in-depth tutorial

This tutorial builds a complete, natively-rendered todo application in pure Go
and ships it to the iOS simulator and the Android emulator. The finished code
lives in [`examples/todoapp`](../examples/todoapp), and every excerpt below is
taken from it verbatim — you can read along in the source or rebuild it file
by file.

The todo app is small, but it was chosen to exercise the whole of the current
framework surface. Each feature maps to a concept:

| App feature | Framework concept |
|---|---|
| The entry field | Controlled inputs, `InputWithSubmit`, the echo/rewrite contract |
| Add / toggle / delete | State via `NewState`, the rules of hooks, immutable updates |
| The task list | Virtualized `List`, `For`, `Keyed`, reconciler semantics |
| Filter chips | Derived state, style-driven selection, `Transition` |
| Delete / clear buttons | Theme base styles and when to override them |
| Everything | Accessibility props, the mobile bridge, three levels of testing |

---

## 1. The mental model

Govinci is a declarative UI framework: you describe the whole screen as a pure
function of state, and the framework figures out what actually changed. Three
types carry the model (`core/view.go`, `core/node.go`, `core/context.go`):

- **`View`** — anything with `Render(ctx *Context) *Node`. Widgets like
  `core.Text` and containers like `core.Column` return Views; your components
  are functions that return Views.
- **`Node`** — the rendered form: a type tag, props, a style, and children.
  Trees of Nodes are what gets diffed and what crosses to the native side as
  JSON.
- **`Context`** — owns everything mutable: state slots, the callback
  registry, the theme, the dirty flag. There is no global state; two contexts
  are two independent apps.

Each frame is a **render pass**: the framework rewinds the context, calls your
root function, gets a fresh Node tree, and diffs it against the previous one.
The diff — a list of patches — is what the native renderer applies:

```
        ┌──────────────  Go  ──────────────┐      ┌──── native shell ────┐
        │                                  │      │                      │
  state ─► root fn ─► Node tree ─► diff ───┼─────►│ apply patches        │
    ▲                                      │ JSON │ (SwiftUI / Compose)  │
    │                                      │      │                      │
    └── handler runs ◄─── callback ID ◄────┼──────┤ user taps / types    │
                                           │      │                      │
        └──────────────────────────────────┘      └──────────────────────┘
```

Event handlers are plain Go closures. When a builder like `Button` receives
one, it registers the closure in the context and puts a **string ID** into the
node's props (`"onClick": "cb_0"`). The native side never sees your function —
it sees the ID, and dispatches it back over the bridge when the user taps.
IDs are per-pass sequence numbers, which has one practical consequence you
will meet in the testing section: always dispatch IDs read from the *current*
tree, never a cached one.

---

## 2. Project setup: a bindable app package

An app is a Go package with two obligations (mirroring `examples/mobileapp`):

```go
package todoapp

import (
	"github.com/GraHms/govinci/core"
	"github.com/GraHms/govinci/mobile"
)

func init() {
	mobile.Register(core.NewContext(), App)
}

// AppName exists to be bindable. gobind only links a bound package when it
// references at least one bindable exported symbol; App is not bindable
// (function-typed parameters are unsupported), and without this the package —
// including the init that registers the app — would be dropped from the
// native library, leaving the bridge with a nil manager.
func AppName() string { return "Govinci Todo" }
```

The `init` is the whole integration: gomobile runs package inits when the
native library loads, so by the time the Swift or Kotlin shell asks for the
first render, the app is already registered. The `AppName` quirk matters —
forget it and the package silently drops out of the binary.

The root view always has the signature `func(*core.Context) core.View`.

---

## 3. State, and where it lives

The app has four pieces of state, all declared at the top of the root:

```go
func App(ctx *core.Context) core.View {
	todos := core.NewState(ctx, []Todo{})
	draft := core.NewState(ctx, "")
	filter := core.NewState(ctx, filterAll)
	nextID := core.NewState(ctx, 1)
	...
```

`NewState` is a hook in the React sense: state is stored in **positional
slots** on the context, matched to call sites by the order in which they run.
That gives you React's rules, and they are worth internalizing:

1. **Call hooks unconditionally, in the same order, every pass.** No
   `NewState` inside an `if` or after an early return.
2. **Don't create state inside list rows** when the list can grow, shrink,
   or reorder — a deleted row's slot would silently be read by its neighbor.
   Keep collection state in the parent and pass values plus closures down.
   This is why `todoRow` below is a pure function of its `Todo`.
3. **Bind the state to a variable.** `State[T]`'s methods have pointer
   receivers, so `core.NewState(ctx, 0).Get()` does not compile.

`Set` does two things: writes the slot and marks the context dirty, which
schedules a re-render. There is no manual refresh anywhere in the app.

### Mutations copy, never mutate

Every write goes through a helper that builds a fresh slice:

```go
	setDone := func(id int, done bool) {
		next := make([]Todo, len(todos.Get()))
		copy(next, todos.Get())
		for i := range next {
			if next[i].ID == id {
				next[i].Done = done
			}
		}
		todos.Set(next)
	}
```

This is not ceremony. The reconciler diffs the new tree against the previous
one; if a handler mutated a slice that the previous pass already captured,
the "before" and "after" would be the same memory and edits could vanish from
the diff. Copy-on-write keeps the previous tree honest. The same reasoning
makes `Todo` a value type, not a pointer.

Note also `nextID`: todos are identified by a monotonic ID, never by index.
Indices shift on delete, which would corrupt both row keys and the closures
captured by row buttons.

### Derived values are computed, not stored

The remaining count, the filtered slice, and the empty-state message are all
recomputed from `todos` on every pass:

```go
	visible := make([]Todo, 0, len(todos.Get()))
	for _, t := range todos.Get() {
		switch filter.Get() {
		case filterActive:
			if !t.Done {
				visible = append(visible, t)
			}
		...
```

Storing them in their own state would be a synchronization bug waiting to
happen. If a value can be computed from the source of truth, compute it —
render passes are cheap; drifted state is not.

---

## 4. The view, top to bottom

### Layout and styling

```go
	return core.SafeArea(
		core.Column(
			core.FlexGrow(1),
			core.Gap(12),

			core.Text("Todos", core.UseStyle(core.Style{
				FontSize:   28,
				FontWeight: core.Bold,
			})),
			...
```

Containers (`Column`, `Row`, `Card`, `Box`, `List`) take style props, behavior
props, and children interleaved in any order. Styles come in two equivalent
forms — modifier functions (`core.Gap(12)`) and the struct form
(`core.UseStyle(core.Style{...})`, merging non-zero fields) — use whichever
reads better at the call site.

Two theme facts to know early:

- Every widget starts from the current theme's base style for its type
  (`ctx.Theme().Components.Button`, etc.) and your props are applied *on
  top*. This is usually what you want, and occasionally what bites you —
  see the destructive buttons below.
- `Box` is the only container with no theme base; `Column` and `Row` carry
  default padding.

### The entry row: controlled input with a submit action

```go
			core.Row(
				core.Gap(8),
				core.InputWithSubmit(draft.Get(), "What needs doing?",
					func(v string) { draft.Set(v) },
					addTodo,
					core.FlexGrow(1),
				),
				core.Button("Add", addTodo,
					core.AccessibilityHint("Adds the task typed in the field"),
				),
			),
```

Inputs are **fully controlled**: the value always comes from Go, every
keystroke goes up as a text event, and the patched value comes back down.
`InputWithSubmit` adds a second handler dispatched when the user presses the
keyboard's return key (iOS) or the IME done action (Android) — it rides the
same void-callback channel as a button tap, so `addTodo` serves both commit
paths unchanged.

`addTodo` shows the whole state discipline in six lines:

```go
	addTodo := func() {
		title := strings.TrimSpace(draft.Get())
		if title == "" {
			return
		}
		todos.Set(append(append([]Todo{}, todos.Get()...), Todo{ID: nextID.Get(), Title: title}))
		nextID.Set(nextID.Get() + 1)
		draft.Set("")
	}
```

That final `draft.Set("")` is more interesting than it looks. The native text
field is locally-owned *while focused* — keystrokes echo instantly without
waiting for the Go round trip, and late echoes never snap the cursor. So how
does a clear-on-submit land while the user's cursor is still in the field?
The renderers keep a queue of every value the field itself sent upstream: an
upstream change that matches a queued entry is an echo of the user's own edit
(ignored), while one that matches nothing the field sent can only be Go
deliberately rewriting the value — and that lands immediately, focused or
not. As an app author you don't manage any of this; you just `Set` the state
and the contract holds. (If you're curious, the mechanism lives in
`GovinciTextField` in both `ios/Govinci/Runtime/Renderer.swift` and
`android/.../Renderer.kt`.)

### The filter bar: selection as style, driven by one loop

```go
func filterBar(active int, onSelect func(int)) core.View {
	return core.Row(
		core.Gap(8),
		core.For(filterLabels, func(label string, i int) core.View {
			styles := []core.StyleProp{
				core.FontSize(13),
				core.Transition(200, core.EaseInOut),
				core.AccessibilityHint("Filters the task list"),
			}
			accLabel := "Show " + strings.ToLower(label) + " tasks"
			if i == active {
				styles = append(styles,
					core.BackgroundColor(colorAccent),
					core.TextColor(colorAccentInk),
				)
				accLabel += ", selected"
			}
			styles = append(styles, core.AccessibilityLabel(accLabel))
			return core.Keyed("filter-"+label,
				core.Button(label, func() { onSelect(i) }, styles...),
			)
		}),
	)
}
```

Three ideas at work:

- **Selection is a style difference, not a structure difference.** Switching
  filters patches two background colors instead of rebuilding the row, and
  `Transition(200, core.EaseInOut)` makes the highlight animate *natively* —
  Compose and SwiftUI drive the frames; no patches flow during the animation.
- **`For` + closures.** The render function's parameters are per-iteration,
  so `func() { onSelect(i) }` captures the right index without the classic
  loop-variable trap.
- **Contrast is your job when you override half a pair.** The theme's button
  base paints a white label; put a pale background under it and the label
  vanishes. The selected chip overrides background *and* text color together.
  (This screenshot-verified lesson repeats with the delete buttons.)

### The list: virtualization, keys, and what the reconciler does

```go
			core.IfElse(len(visible) == 0,
				core.Text(emptyMessage, core.UseStyle(core.Style{
					FontSize:  14,
					TextColor: colorDim,
				})),
				core.List(
					core.FlexGrow(1),
					core.For(visible, func(t Todo, _ int) core.View {
						return todoRow(t, setDone, removeTodo)
					}),
				),
			),
```

`core.List` maps to `LazyColumn` on Android and `LazyVStack` on iOS: rows are
materialized natively as they scroll into view, so a thousand-item list costs
what the visible screen costs. Use `List` for data-driven collections and
`Scroll`+`Column` for short static content.

Each row is keyed by identity:

```go
func todoRow(t Todo, setDone func(int, bool), remove func(int)) core.View {
	...
	id := t.ID

	return core.Keyed(fmt.Sprintf("todo-%d", t.ID), core.Row(
		core.Padding(8),
		core.Gap(10),
		core.Transition(200, core.EaseInOut),
		core.AccessibilityLabel(rowAccessibilityLabel(t)),

		core.Checkbox(t.Done, func(v bool) { setDone(id, v) }),
		core.Text(t.Title, core.UseStyle(core.Style{
			FontSize:  16,
			TextColor: titleColor,
		}), core.FlexGrow(1)),
		core.Button("✕", func() { remove(id) },
			core.FontSize(13),
			core.TextColor("#FFFFFF"),
			core.BackgroundColor(colorDanger),
			core.AccessibilityLabel("Delete "+t.Title),
		),
	))
}
```

What keys buy you: the reconciler matches children **by index**, and when the
nodes at an index carry different keys it replaces that slot outright. So on
insert or delete, keyed rows stay visually correct — the row that used to be
at index 3 isn't smeared into the todo now living there. The cost of a
replace is transient native state (focus, in-progress animation) for that
subtree, which is why rows should hold none — reinforcing the rule that row
state lives in the parent.

Note `id := t.ID` before the closures: the buttons address the todo by ID, so
they still hit the right one after the `visible` slice is rebuilt under a
different filter. Deleting "Walk dog" while the Active filter is on must
delete it from `todos`, not from position 0 of whatever is visible.

Also note what the row does *not* contain: gesture props. Leaf widgets
(`Text`, `Button`, `Input`) take only style props; if you need a tappable
region that isn't a button, put `core.OnClick(...)` / `core.OnLongPress(...)`
on a container (`Row`, `Box`) — see the feed tab in `examples/mobileapp` for
that pattern.

### Destructive actions and the theme base

```go
		core.Button("✕", func() { remove(id) },
			core.FontSize(13),
			core.TextColor("#FFFFFF"),
			core.BackgroundColor(colorDanger),
			...
```

First drafts of this app styled the ✕ as a red glyph and left the background
alone — which produced a red-on-blue button, because the default theme's
button base is a medium blue. The rule of thumb: **when a widget's meaning
departs from the theme's default (a destructive action on a primary-styled
base), override the full color pair, not one half.** The footer's
"Clear completed" button gets the identical treatment.

### Accessibility is part of the tree

Accessibility is declared with style props and flows through the same patch
pipeline as everything else:

- `AccessibilityLabel` on rows composes the state into a sentence
  (`"Buy milk, completed"`), on icon buttons it replaces the glyph
  (`"Delete Buy milk"` instead of "✕").
- `AccessibilityHint` describes the consequence of activating.
- `AccessibilityHidden()` removes decoration — the hairline divider — from
  the screen-reader tree.

The XCUITest below addresses rows by these labels, so element lookup in the
test suite doubles as proof the accessibility wiring works.

---

## 5. Testing at three levels

### Level 1: bridge tests in Go (seconds, no simulator)

The package init has already registered the app, so a plain Go test can make
exactly the call sequence the native shells make — read the tree, dispatch a
callback ID found in it, assert on the patches:

```go
	patches := mobile.TriggerCallback(
		mustCallback(t, buttonLabeled("Add"), "onClick", `Button "Add"`))
	if !strings.Contains(patches, "1 item left") {
		t.Errorf("add patches don't update the count:\n%s", patches)
	}
```

`examples/todoapp/app_test.go` drives the full lifecycle this way: add,
blank-submission no-op, toggle, all three filters, delete-under-filter, bulk
clear, and the Enter/submit path. Two disciplines to copy:

- **Dispatch only IDs from the live tree.** Callback IDs are per-pass
  sequence numbers; any state change can renumber them. The `mustCallback`
  helper re-reads the tree before every dispatch.
- **The bridge is a process-wide singleton**, so order your assertions as one
  user journey rather than independent tests fighting over shared state.

Run it with `go test ./examples/todoapp/`. This is your inner loop — it
catches nearly everything in well under a second.

### Level 2: iOS simulator via XCUITest

Nearly everything. The one todo-app bug that reached a device — Add not
clearing the input — lived in the *renderer*, below what bridge tests can
see: Go emitted the correct `"value":""` patch, and the focused native field
dropped it. Only a test that owns a real keyboard could catch it, which is
what `ios/GovinciUITests/TodoAppUITests.swift` does:

```swift
        field.tap()
        field.typeText("Buy milk")
        app.buttons["Add"].tap()          // field is still focused here
        ...
        let after = (field.value as? String) ?? ""
        XCTAssertTrue(after.isEmpty || after == "What needs doing?",
                      "input not cleared after add, still shows: \(after)")
```

Run it (after the build in section 6) with:

```sh
cd ios && xcodebuild test -project GovinciApp.xcodeproj -scheme GovinciApp \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  -only-testing:GovinciUITests/TodoAppUITests
```

### Level 3: Android emulator via adb

Android has no committed UI test yet; `adb` scripting covers the same ground
interactively — type, press Enter (keyevent 66), then read the accessibility
dump and assert the row exists and the placeholder is back (the placeholder
only renders when the field is empty):

```sh
adb shell input tap <field-x> <field-y>
adb shell input text 'Buy%smilk'
adb shell input keyevent 66
adb shell uiautomator dump /sdcard/ui.xml && adb shell cat /sdcard/ui.xml
```

The layered takeaway: bridge tests prove your *app logic*, device tests prove
the *renderer contract* — and each has caught real bugs the other cannot.

---

## 6. Shipping it

Both build scripts take the app package as an optional argument (default:
the `examples/mobileapp` demo).

**iOS** (needs full Xcode + gomobile):

```sh
ios/build.sh ./examples/todoapp        # gomobile bind → Govinci.xcframework
cd ios && xcodegen generate
xcodebuild -project GovinciApp.xcodeproj -scheme GovinciApp \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build
xcrun simctl install booted <path-to>/GovinciApp.app
xcrun simctl launch booted com.govinci.demo
```

**Android** (needs SDK/NDK + JDK 17; export `ANDROID_HOME` for Gradle —
`build.sh` only sets it for gomobile):

```sh
android/build.sh ./examples/todoapp    # gomobile bind → app/libs/govinci.aar
cd android && gradle assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
adb shell am start -n com.govinci.app/.MainActivity
```

The same Go code, the same state, the same tests — rendered by SwiftUI on one
platform and Jetpack Compose on the other.

---

## 7. Where to go from here

The todo app deliberately leaves framework surface unexplored:

- **Effects and timers** — `hooks.UseEffect`, `hooks.UseInterval`,
  `hooks.UseTimeout` (`examples/mobileapp`'s counter tab ticks a clock over
  the push channel with no native event in flight).
- **Navigation** — `core.Navigator` with `Push`/`Pop`/`Replace`
  (`examples/social`).
- **Container gestures** — `OnClick`/`OnLongPress`/`OnTouch` on rows and
  boxes (`examples/mobileapp`'s feed tab).
- **Theming** — a custom `core.Theme` injected with `WithThemeOpt`, and
  components reading tokens via `ctx.Theme()` (`examples/fintechapp`).
- **Persistence** — the mutation helpers are the single write choke point;
  flushing `todos` to disk from them is a contained change.

For the internals referenced throughout: [`docs/reconciliation.md`](reconciliation.md)
covers the diffing engine, and [`docs/ui-architecture.md`](ui-architecture.md)
covers the styling and theme system.
