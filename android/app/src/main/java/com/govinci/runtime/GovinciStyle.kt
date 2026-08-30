package com.govinci.runtime

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import org.json.JSONObject

/**
 * Kotlin mirror of Go's core.Style, decoded from the tree/patch JSON.
 *
 * Field names in the JSON are the Go struct's exported names verbatim
 * ("FontSize", "TextColor", ...) because core.Style carries no json tags.
 * Only the subset the Go DSL can actually produce today is mapped; the
 * web-oriented fields (Position, ZIndex, Transition, Animation, pseudo
 * states) have no Compose analog at this layer and are intentionally
 * ignored rather than half-implemented.
 */
data class GovinciStyle(
    val fontSize: Float,
    val fontWeight: Int,
    val textColor: Color?,
    val background: Color?,
    val padding: Edges,
    val margin: Edges,
    val borderRadius: Float,
    val shadow: Float,
    val align: String,
    val display: String,
    val width: String,
    val height: String,
    val borderColor: Color?,
    val borderWidth: Float,
    val gap: Float,
    val justifyContent: String,
    val alignItems: String,
    val flexGrow: Float,
    val lineHeight: Int,
) {
    data class Edges(val top: Int, val right: Int, val bottom: Int, val left: Int)

    companion object {
        fun parse(obj: JSONObject?): GovinciStyle? {
            if (obj == null) return null
            return GovinciStyle(
                fontSize = obj.optDouble("FontSize", 0.0).toFloat(),
                fontWeight = obj.optInt("FontWeight", 0),
                textColor = parseColor(obj.optString("TextColor")),
                background = parseColor(obj.optString("Background")),
                padding = parseEdges(obj.optJSONObject("Padding")),
                margin = parseEdges(obj.optJSONObject("Margin")),
                borderRadius = obj.optDouble("BorderRadius", 0.0).toFloat(),
                shadow = obj.optDouble("Shadow", 0.0).toFloat(),
                align = obj.optString("Align"),
                display = obj.optString("Display"),
                width = obj.optString("Width"),
                height = obj.optString("Height"),
                borderColor = parseColor(obj.optString("BorderColor")),
                borderWidth = obj.optDouble("BorderWidth", 0.0).toFloat(),
                gap = obj.optDouble("Gap", 0.0).toFloat(),
                justifyContent = obj.optString("JustifyContent"),
                alignItems = obj.optString("AlignItems"),
                flexGrow = obj.optDouble("FlexGrow", 0.0).toFloat(),
                lineHeight = obj.optInt("LineHeight", 0),
            )
        }

        /**
         * Go's EdgeInsets carries per-side values plus Horizontal/Vertical
         * shorthands; the shorthand fills any side not set explicitly, which
         * matches how the DSL's PaddingHorizontal-style helpers are used.
         */
        private fun parseEdges(obj: JSONObject?): Edges {
            if (obj == null) return Edges(0, 0, 0, 0)
            val h = obj.optInt("Horizontal", 0)
            val v = obj.optInt("Vertical", 0)
            fun side(name: String, shorthand: Int): Int {
                val explicit = obj.optInt(name, 0)
                return if (explicit != 0) explicit else shorthand
            }
            return Edges(
                top = side("Top", v),
                right = side("Right", h),
                bottom = side("Bottom", v),
                left = side("Left", h),
            )
        }

        /** Accepts CSS-style #RGB, #RRGGBB, and #RRGGBBAA (Go emits the latter two). */
        fun parseColor(hex: String?): Color? {
            if (hex.isNullOrEmpty() || !hex.startsWith("#")) return null
            val s = hex.substring(1)
            return try {
                when (s.length) {
                    3 -> {
                        val r = s[0].digitToInt(16) * 17
                        val g = s[1].digitToInt(16) * 17
                        val b = s[2].digitToInt(16) * 17
                        Color(r, g, b)
                    }
                    6 -> {
                        val v = s.toLong(16)
                        Color(0xFF000000L or v)
                    }
                    // CSS orders the alpha byte last; android.graphics wants it
                    // first, so recompose the channels rather than parse directly.
                    8 -> {
                        val v = s.toLong(16)
                        val rgb = v ushr 8
                        val a = v and 0xFF
                        Color((a shl 24) or rgb)
                    }
                    else -> null
                }
            } catch (_: NumberFormatException) {
                null
            }
        }
    }
}

/**
 * Builds this style's box modifiers in CSS box-model order, outermost first:
 * margin, size, elevation shadow, corner clip, background, border, then inner
 * padding. The order is load-bearing — e.g. padding before background would
 * paint the background inside the padding, and clip after background would
 * leave square corners painted.
 *
 * `extra` is a scope-dependent modifier the parent computed for this child
 * (today: Row/Column weight from FlexGrow, which only exists as a RowScope/
 * ColumnScope extension and so cannot be built here).
 */
fun GovinciStyle?.boxModifier(extra: Modifier = Modifier): Modifier {
    var m: Modifier = extra
    if (this == null) return m

    if (margin != Edges0) {
        m = m.padding(
            start = margin.left.dp, top = margin.top.dp,
            end = margin.right.dp, bottom = margin.bottom.dp,
        )
    }
    m = m.then(dimensionModifier(width, horizontal = true))
    m = m.then(dimensionModifier(height, horizontal = false))

    val shape = if (borderRadius > 0f) RoundedCornerShape(borderRadius.dp) else null
    if (shadow > 0f) {
        m = m.shadow(elevation = shadow.dp, shape = shape ?: RoundedCornerShape(0.dp))
    }
    if (shape != null) m = m.clip(shape)
    background?.let { m = m.background(it) }
    if (borderWidth > 0f && borderColor != null) {
        m = m.border(borderWidth.dp, borderColor, shape ?: RoundedCornerShape(0.dp))
    }
    if (padding != Edges0) {
        m = m.padding(
            start = padding.left.dp, top = padding.top.dp,
            end = padding.right.dp, bottom = padding.bottom.dp,
        )
    }
    // "hidden" keeps the node's space but not its pixels ("none" is handled
    // earlier by not composing the node at all — see RenderNode).
    if (display == "hidden") m = m.alpha(0f)
    return m
}

private val Edges0 = GovinciStyle.Edges(0, 0, 0, 0)

/**
 * Maps a Go dimension string onto a size modifier. Supported forms: "120px"
 * or a bare number (density-independent pixels), "100%" / other percentages
 * (fraction of the parent), and ""/"auto" (wrap content, i.e. no modifier).
 */
private fun dimensionModifier(value: String, horizontal: Boolean): Modifier {
    if (value.isEmpty() || value == "auto") return Modifier
    if (value.endsWith("%")) {
        val pct = value.dropLast(1).toFloatOrNull() ?: return Modifier
        val fraction = (pct / 100f).coerceIn(0f, 1f)
        return if (horizontal) Modifier.fillMaxWidth(fraction) else Modifier.fillMaxHeight(fraction)
    }
    val number = value.removeSuffix("px").toFloatOrNull() ?: return Modifier
    return if (horizontal) Modifier.width(number.dp) else Modifier.height(number.dp)
}
