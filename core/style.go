package core

type Style struct {
	FontSize     float64
	FontWeight   Weight
	TextColor    string
	Background   string
	Padding      EdgeInsets
	Margin       EdgeInsets
	BorderRadius float64
	Shadow       float64
	Align        Alignment
	Display      DisplayMode
	Width        string
	Height       string
	BorderColor  string
	BorderWidth  float64
	Position     Position
	Top          string
	Left         string
	Right        string
	Bottom       string
	ZIndex       int
	Overflow     string // "hidden", "scroll", "visible"
	WhiteSpace   string // "nowrap", "normal", "pre-line"
	LineHeight   int
	MaxWidth     string
	Gap          float64
	Transition   string // "all 0.3s ease"
	Animation    string // "bounce 2s infinite"

	HoverStyle   *Style
	FocusStyle   *Style
	PseudoStates map[string]Style // ":hover", ":focus"

	FlexDirection  FlexDirection
	JustifyContent JustifyContent
	AlignItems     AlignItems
	MinHeight      string
	MinWidth       string
	ColumnGap      float64
	RowGap         float64
	FlexWrap       string
	AlignSelf      AlignItems
	FlexBasis      string
	FlexShrink     float64
	FlexGrow       float64

	// Accessibility semantics. These live on Style rather than Props so every
	// builder that takes StyleProps — leaves and containers alike — supports
	// them without a signature change, and so the reconciler's value-compared
	// update-style patches carry changes to them like any visual property.
	// Renderers map them onto the platform's semantics layer: contentDescription
	// / clearAndSetSemantics on Android, accessibilityLabel / accessibilityHint /
	// accessibilityHidden on iOS.
	AccessibilityLabel  string
	AccessibilityHint   string
	AccessibilityHidden bool
}

type Weight int

const (
	Light  Weight = 200
	Normal Weight = 400
	Bold   Weight = 700
)

type EdgeInsets struct {
	Top, Right, Bottom, Left int
	Horizontal, Vertical     int
}
type StyleProp interface {
	Apply(*Style)
}
type styleFunc func(*Style)

func (f styleFunc) Apply(s *Style) {
	f(s)
}

func UseStyle(s Style) StyleProp {
	return styleFunc(func(target *Style) {
		// Merge only non-zero fields
		if s.FontSize != 0 {
			target.FontSize = s.FontSize
		}
		if s.FontWeight != 0 {
			target.FontWeight = s.FontWeight
		}
		if s.TextColor != "" {
			target.TextColor = s.TextColor
		}
		if s.Background != "" {
			target.Background = s.Background
		}
		if s.BorderRadius != 0 {
			target.BorderRadius = s.BorderRadius
		}
		if s.Shadow != 0 {
			target.Shadow = s.Shadow
		}
		if s.Align != "" {
			target.Align = s.Align
		}
		if s.Display != "" {
			target.Display = s.Display
		}
		if s.Padding != (EdgeInsets{}) {
			target.Padding = s.Padding
		}
		if s.Margin != (EdgeInsets{}) {
			target.Margin = s.Margin
		}
		if s.Bottom != "" {
			target.Bottom = s.Bottom
		}
		if s.Left != "" {
			target.Left = s.Left
		}
		if s.Right != "" {
			target.Right = s.Right
		}
		if s.ZIndex != 0 {
			target.ZIndex = s.ZIndex
		}

	})
}
func PrimaryColor() string { return "#007AFF" }
func DangerColor() string  { return "#FF3B30" }
func RoundedShadowBox() StyleProp {
	return UseStyle(Style{
		BorderRadius: 12,
		Shadow:       2,
		Background:   "#FFFFFF",
	})
}

var TextInputStyle = UseStyle(Style{
	FontSize:     16,
	TextColor:    "#000000",
	Background:   "#FFFFFF",
	Padding:      EdgeInsets{Top: 10, Bottom: 10, Left: 12, Right: 12},
	BorderRadius: 8,
	Shadow:       1,
})

func PaddingTop(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding.Top = px
	})
}

func PaddingHorizontal(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding.Horizontal = px
	})
}

type ResponsiveStyle map[string]Style // "mobile", "tablet", "desktop"

type Alignment string

const (
	AlignStart    Alignment = "start"
	AlignCenter   Alignment = "center"
	AlignEnd      Alignment = "end"
	AlignStretch  Alignment = "stretch"
	AlignBaseline Alignment = "baseline"
	AlignJustify  Alignment = "justify"
)

type DisplayMode string

const (
	DisplayVisible DisplayMode = "visible"
	DisplayHidden  DisplayMode = "hidden"
	DisplayNone    DisplayMode = "none"
	DisplayInline  DisplayMode = "inline"
	DisplayBlock   DisplayMode = "block"
)

type JustifyContent string
type FlexDirection string
type AlignItems string

const (
	JustifyStart   JustifyContent = "flex-start"
	JustifyCenter  JustifyContent = "center"
	JustifyEnd     JustifyContent = "flex-end"
	JustifyBetween JustifyContent = "space-between"
	JustifyAround  JustifyContent = "space-around"
	JustifyEvenly  JustifyContent = "space-evenly"

	AlignItemsStart   AlignItems = "flex-start"
	AlignItemsCenter  AlignItems = "center"
	AlignItemsEnd     AlignItems = "flex-end"
	AlignItemsStretch AlignItems = "stretch"

	FlexRow     FlexDirection = "row"
	FlexColumn  FlexDirection = "column"
	DisplayFlex               = "flex"
)

type Position string

const (
	PositionRelative Position = "relative"
	PositionAbsolute Position = "absolute"
	PositionFixed    Position = "fixed"
	PositionSticky   Position = "sticky"
)

//func Responsive(breakpoint string, style Style) StyleProp {
//	return styleFunc(func(s *Style) {
//		if s.Responsive == nil {
//			s.Responsive = make(ResponsiveStyle)
//		}
//		s.Responsive[breakpoint] = style
//	})
//}
