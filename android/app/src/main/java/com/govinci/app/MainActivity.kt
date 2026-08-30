package com.govinci.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import com.govinci.runtime.GovinciRoot
import com.govinci.runtime.GovinciRuntime

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // The runtime mounts the initial Go-rendered tree and opens the push
        // channel; after that the composition tracks the TreeStore on its own.
        // Recreation (rotation, process restore) simply remounts from Go's
        // current state — the Go side is a process-wide singleton.
        val runtime = GovinciRuntime(GomobileBridge(filesDir.absolutePath))
        runtime.start()
        setContent { GovinciRoot(runtime) }
    }
}
