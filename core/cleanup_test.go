package core

import "testing"

func TestCloseRunsRegisteredCleanupsOnce(t *testing.T) {
	ctx := NewContext()
	runs := 0
	ctx.OnClose(func() { runs++ })

	ctx.Close()
	if runs != 1 {
		t.Fatalf("cleanup ran %d times after first Close, want 1", runs)
	}
	// Drain semantics: an already-run cleanup is forgotten, not re-run.
	ctx.Close()
	if runs != 1 {
		t.Fatalf("cleanup ran %d times after second Close, want 1", runs)
	}
}

func TestCleanupRegistryIsSharedAcrossDerivedContexts(t *testing.T) {
	// Children, scopes, and WithTheme/WithConfig copies must all feed the
	// root's registry, so closing the root stops resources registered
	// anywhere in the tree.
	ctx := NewContext()
	runs := map[string]int{}

	ctx.NewChildContext().OnClose(func() { runs["child"]++ })
	ctx.Scope("s").OnClose(func() { runs["scope"]++ })
	ctx.WithTheme(DefaultTheme).OnClose(func() { runs["themed"]++ })
	ctx.WithConfig(&AppConfig{}).OnClose(func() { runs["configured"]++ })

	ctx.Close()
	for _, name := range []string{"child", "scope", "themed", "configured"} {
		if runs[name] != 1 {
			t.Fatalf("cleanups after Close = %v, want each derived context's exactly once", runs)
		}
	}
}

func TestCleanupRegistriesAreIsolatedPerRoot(t *testing.T) {
	a, b := NewContext(), NewContext()
	runs := map[string]int{}
	a.OnClose(func() { runs["a"]++ })
	b.OnClose(func() { runs["b"]++ })

	a.Close()
	if runs["a"] != 1 || runs["b"] != 0 {
		t.Fatalf("closing app A ran cleanups %v, want only A's", runs)
	}
}

func TestCloseThenRegisterThenCloseAgain(t *testing.T) {
	// The re-mount shape (WASM RenderInitial called twice over one ctx): a
	// resource registered after a Close must be stopped by the next Close,
	// not dropped or run immediately.
	ctx := NewContext()
	ctx.Close()

	runs := 0
	ctx.OnClose(func() { runs++ })
	if runs != 0 {
		t.Fatal("cleanup registered after a Close ran immediately")
	}
	ctx.Close()
	if runs != 1 {
		t.Fatalf("cleanup ran %d times, want 1 on the Close after registration", runs)
	}
}
