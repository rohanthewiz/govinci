package core

import "strconv"

// Easing names the timing curve of a transition. The values are the CSS
// keywords — the Style.Transition field predates the native renderers and is
// CSS-shaped, so the DSL keeps that vocabulary and each renderer maps it onto
// its own curve type (Compose CubicBezierEasing, SwiftUI Animation). The
// cubic-bezier control points the CSS spec defines for each keyword are what
// the native mappings reproduce, so one Go declaration animates identically
// on Android, iOS, and the web backends.
type Easing string

const (
	EaseLinear Easing = "linear"
	Ease       Easing = "ease"
	EaseIn     Easing = "ease-in"
	EaseOut    Easing = "ease-out"
	EaseInOut  Easing = "ease-in-out"
)

// Transition declares that changes to this node's animatable properties —
// background color, size, padding, list placement — should animate over the
// given duration instead of snapping. This is the "declare in Go, drive
// natively" model: Go only ships the declaration in the style; each frame of
// the animation is produced by the platform's animation system (Compose,
// SwiftUI, CSS transitions), never by patches over the bridge.
//
// The canonical serialized form is "<ms>ms <easing>" (e.g. "250ms
// ease-in-out"), which the native parsers read; they also tolerate the CSS
// longhand ("all 0.3s ease") for styles written by hand.
func Transition(durationMs int, easing Easing) StyleProp {
	return styleFunc(func(s *Style) {
		if durationMs <= 0 {
			s.Transition = ""
			return
		}
		if easing == "" {
			easing = Ease
		}
		s.Transition = strconv.Itoa(durationMs) + "ms " + string(easing)
	})
}
