package core

import "fmt"

func FlexGrow(value float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.FlexGrow = value
	})
}
func FlexShrink(value float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.FlexShrink = value
	})
}
func FlexBasis(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.FlexBasis = value
	})
}
func AlignSelf(value AlignItems) StyleProp {
	return styleFunc(func(s *Style) {
		s.AlignSelf = value
	})
}
func FlexWrap(enabled bool) StyleProp {
	return styleFunc(func(s *Style) {
		if enabled {
			s.FlexWrap = "wrap"
		} else {
			s.FlexWrap = "nowrap"
		}
	})
}
func RowGap(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.RowGap = px
	})
}

func ColumnGap(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.ColumnGap = px
	})
}
func MinWidth(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.MinWidth = value
	})
}

func MinHeight(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.MinHeight = value
	})
}
func Overflow(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Overflow = value // "hidden", "scroll", "visible"
	})
}
func Responsive(breakpoint string, style Style) StyleProp {
	return styleFunc(func(s *Style) {
		if s.PseudoStates == nil {
			s.PseudoStates = make(map[string]Style)
		}
		s.PseudoStates[breakpoint] = style
	})
}
func FontSize(size float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.FontSize = size
	})
}

func TextColor(hex string) StyleProp {
	return styleFunc(func(s *Style) {
		s.TextColor = hex
	})
}
func Gap(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.Gap = px
	})
}

func BackgroundColor(hex string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Background = hex
	})
}
func Align(a Alignment) StyleProp {
	return styleFunc(func(s *Style) {
		s.Align = a
	})
}

func Display(mode DisplayMode) StyleProp {
	return styleFunc(func(s *Style) {
		s.Display = mode
	})
}

func Padding(all int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding = EdgeInsets{
			Top: all, Right: all, Bottom: all, Left: all,
		}
	})
}
func BorderRadius(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.BorderRadius = px
	})
}

func Shadow(elevation float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.Shadow = elevation
	})
}

func FontWeight(weight Weight) StyleProp {
	return styleFunc(func(s *Style) {
		s.FontWeight = weight
	})
}

func Width(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Width = w
	})
}
func MaxWidth(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.MaxWidth = w
	})
}
func Height(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Height = w
	})
}
func MaxHeight(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.MaxWidth = w
	})
}
func Background(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Background = w
	})
}

func LinearGradient(x, y, z string) string {
	return fmt.Sprintf(`linear-gradient(%s, #%s, #%s)`, x, y, z)
}

func Margin(all int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Margin = EdgeInsets{
			Top: all, Right: all, Bottom: all, Left: all,
		}
	})
}

func FlexDir(dir FlexDirection) StyleProp {
	return styleFunc(func(s *Style) {
		s.FlexDirection = dir
	})
}

func Justify(j JustifyContent) StyleProp {
	return styleFunc(func(s *Style) {
		s.JustifyContent = j
	})
}

func AlignItemsProp(a AlignItems) StyleProp {
	return styleFunc(func(s *Style) {
		s.AlignItems = a
	})
}

func Bottom(v string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Bottom = v
	})
}

func Left(v string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Left = v
	})
}

func Right(v string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Right = v
	})
}

func ZIndex(v int) StyleProp {
	return styleFunc(func(s *Style) {
		s.ZIndex = v
	})
}

func PaddingVertical(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding.Vertical = px
	})
}

// AccessibilityLabel gives screen readers a name for the element (TalkBack
// contentDescription, VoiceOver label). Set it on anything non-textual a user
// can perceive or activate — images, icon buttons, tappable rows.
func AccessibilityLabel(label string) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityLabel = label
	})
}

// AccessibilityHint describes the *result* of activating the element
// ("Opens the article"). VoiceOver reads it natively; TalkBack has no hint
// slot, so the Android renderer appends it to the content description.
func AccessibilityHint(hint string) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityHint = hint
	})
}

// AccessibilityHidden removes the element (and its subtree) from the
// accessibility tree — for decorative content a screen reader should skip.
func AccessibilityHidden() StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityHidden = true
	})
}

func (s Style) With(other Style) Style {
	merged := s
	UseStyle(other).Apply(&merged)
	return merged
}
