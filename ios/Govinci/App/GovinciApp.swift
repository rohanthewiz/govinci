import SwiftUI

/// The whole shell: build a runtime over the gomobile bridge, mount the Go
/// tree, hand the store to GovinciRoot. Everything the app *does* lives in Go
/// (the bound package registered via mobile.Register — see examples/mobileapp).
@main
struct GovinciApp: App {
    private let runtime: GovinciRuntime

    init() {
        // App.init runs on the main thread (SwiftUI's App protocol is
        // MainActor-isolated), satisfying GovinciRuntime.start's contract.
        // start() before the first body evaluation so the initial tree is
        // there on the very first frame — no empty-flash-then-mount.
        let runtime = GovinciRuntime(bridge: GomobileBridge())
        runtime.start()
        self.runtime = runtime
    }

    var body: some Scene {
        WindowGroup {
            GovinciRoot(runtime: runtime)
        }
    }
}
