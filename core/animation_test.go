package core

import "testing"

func TestTransitionPropCanonicalForm(t *testing.T) {
	cases := []struct {
		ms     int
		easing Easing
		want   string
	}{
		{250, EaseInOut, "250ms ease-in-out"},
		{300, "", "300ms ease"}, // easing defaults to CSS "ease"
		{1000, EaseLinear, "1000ms linear"},
		{0, EaseIn, ""},  // non-positive duration clears the transition
		{-5, EaseIn, ""}, // (a zero-duration "transition" is just a snap)
	}
	for _, c := range cases {
		s := &Style{Transition: "stale"}
		Transition(c.ms, c.easing).Apply(s)
		if s.Transition != c.want {
			t.Errorf("Transition(%d, %q) = %q, want %q", c.ms, c.easing, s.Transition, c.want)
		}
	}
}

func TestTransitionRidesTheNodeStyle(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := Row(Transition(250, EaseInOut), Text("x")).Render(ctx)
	if n.Style.Transition != "250ms ease-in-out" {
		t.Fatalf("node style transition = %q", n.Style.Transition)
	}
}
