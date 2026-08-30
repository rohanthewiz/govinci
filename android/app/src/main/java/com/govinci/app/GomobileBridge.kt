package com.govinci.app

import com.govinci.runtime.GovinciBridge
import mobile.Mobile
import mobile.PatchListener

/**
 * GovinciBridge implementation over the gomobile-generated classes.
 *
 * `mobile.Mobile` / `mobile.PatchListener` come from app/libs/govinci.aar,
 * produced by ../build.sh from the Go `mobile` package (see mobile/bridge.go
 * for the delivery contract). This replaces the old hand-rolled JNI
 * `external fun` bridge — gomobile owns the FFI now.
 *
 * The app itself is Go code: the bound app package registers its root view in
 * an init step (mobile.Register), which runs when the AAR's native library
 * loads, so by the time these calls happen the app is installed.
 */
class GomobileBridge : GovinciBridge {
    override fun renderInitial(): String = Mobile.renderInitial()

    override fun triggerCallback(id: String): String = Mobile.triggerCallback(id)

    override fun triggerTextCallback(id: String, value: String): String =
        Mobile.triggerTextCallback(id, value)

    override fun triggerBoolCallback(id: String, value: Boolean): String =
        Mobile.triggerBoolCallback(id, value)

    override fun triggerIntCallback(id: String, value: Long): String =
        Mobile.triggerIntCallback(id, value)

    override fun setListener(listener: (String) -> Unit) {
        Mobile.setListener(object : PatchListener {
            override fun applyPatches(patches: String) = listener(patches)
        })
    }
}
