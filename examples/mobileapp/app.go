// Package mobileapp is the demo app bound into the Android AAR alongside the
// mobile bridge (see android/build.sh). Its only integration point is the
// init below: gomobile runs package inits when the native library loads, so
// by the time the Kotlin shell calls Mobile.renderInitial() the app is
// already registered. Replace this package in build.sh to ship your own app.
//
// The view deliberately exercises every event kind the bridge carries — void
// (Button), text (Input), bool (Checkbox), int (TabView) — plus the async
// push path (UseInterval ticking with no native event in flight), so it
// doubles as a smoke test for the Compose runtime.
package mobileapp

import (
	"fmt"
	"time"

	"github.com/GraHms/govinci/core"
	"github.com/GraHms/govinci/hooks"
	"github.com/GraHms/govinci/mobile"
)

func init() {
	mobile.Register(core.NewContext(), App)
}

// AppName exists to be bindable. gobind only imports a bound package when it
// references at least one bindable exported symbol; App above is not bindable
// (function-typed parameters are unsupported), and with zero bindable symbols
// the package — including the init that registers the app — is never linked
// into the AAR, leaving the bridge with no app (nil manager) at runtime.
func AppName() string { return "Govinci Demo" }

func App(ctx *core.Context) core.View {
	tab := core.NewState(ctx, 0)

	return core.SafeArea(
		core.TabView(
			core.SelectedIndex(tab.Get()),
			core.OnTabChange(func(i int) { tab.Set(i) }),
			core.Tabs(
				core.Tab("Counter", ""),
				core.Tab("Form", ""),
			),
			core.Content(
				counterTab(),
				formTab(),
			),
		),
	)
}

func counterTab() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		count := core.NewState(ctx, 0)
		seconds := core.NewState(ctx, 0)

		// Drives the Go→native push channel: each tick re-renders with no
		// native event pending a response.
		hooks.UseInterval(ctx, func() { seconds.Set(seconds.Get() + 1) }, time.Second)

		return core.Column(
			core.Text(fmt.Sprintf("Count: %d", count.Get()), core.UseStyle(core.Style{
				FontSize:   28,
				FontWeight: core.Bold,
			})),
			core.Button("Increment", func() { count.Set(count.Get() + 1) }),
			core.Spacer(16),
			core.Text(fmt.Sprintf("App running for %ds", seconds.Get()), core.UseStyle(core.Style{
				FontSize:  13,
				TextColor: "#3C3C4399",
			})),
		).Render(ctx)
	})
}

func formTab() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		name := core.NewState(ctx, "")
		subscribed := core.NewState(ctx, false)

		greeting := "Hello, stranger."
		if name.Get() != "" {
			greeting = "Hello, " + name.Get() + "!"
		}
		subLabel := "Not subscribed"
		if subscribed.Get() {
			subLabel = "Subscribed"
		}

		return core.Column(
			core.Input(name.Get(), "Your name", func(v string) { name.Set(v) }),
			core.Spacer(8),
			core.Text(greeting),
			core.Spacer(8),
			core.Row(
				core.Checkbox(subscribed.Get(), func(v bool) { subscribed.Set(v) }),
				core.Text(subLabel),
			),
		).Render(ctx)
	})
}
