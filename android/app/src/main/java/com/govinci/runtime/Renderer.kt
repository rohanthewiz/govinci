package com.govinci.runtime

import androidx.compose.foundation.background
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsFocusedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import coil.compose.AsyncImage

/**
 * Node-tree → Compose mapping.
 *
 * The design deliberately leans on Compose's own reconciler for everything the
 * Go reconciler doesn't do: view identity across recompositions (`key()` per
 * child), state retention in unchanged siblings, animation plumbing, and
 * accessibility semantics (via material3 components). The Go side only has to
 * keep the data tree correct; nothing here caches views or paths.
 */
val LocalGovinciRuntime = compositionLocalOf<GovinciRuntime> {
    error("GovinciRoot not mounted")
}

@Composable
fun GovinciRoot(runtime: GovinciRuntime) {
    CompositionLocalProvider(LocalGovinciRuntime provides runtime) {
        runtime.store.root?.let { RenderNode(it) }
    }
}

/**
 * `extra` carries parent-scope modifiers (Row/Column weight) that only the
 * parent can construct; see GovinciStyle.boxModifier.
 */
@Composable
fun RenderNode(node: GovinciNode, extra: Modifier = Modifier) {
    val style = node.style
    if (style?.display == "none") return // not composed at all; "hidden" keeps space

    when (node.type) {
        "Text" -> GovinciText(node, extra)
        "Button" -> GovinciButton(node, extra)

        "Input" -> GovinciTextField(node, extra)
        "InputPassword" -> GovinciTextField(node, extra, password = true)
        "NumericInput" -> GovinciTextField(node, extra, numeric = true)
        "TextArea" -> GovinciTextField(node, extra, multiline = true)
        "Checkbox" -> GovinciCheckbox(node, extra)

        "Row" -> GovinciRow(node, extra)
        "Column", "Card" -> GovinciColumn(node, extra) // Card = Column whose Go theme style carries the card look
        "Box" -> Box(style.boxModifier(extra)) { RenderChildren(node) }
        "Spacer" -> Spacer(Modifier.size(node.intProp("size").dp))
        "Scroll" -> Column(
            style.boxModifier(extra).verticalScroll(rememberScrollState())
        ) { ColumnChildren(node) }
        "SafeArea" -> Box(style.boxModifier(extra).safeDrawingPadding()) { RenderChildren(node) }

        "TabView" -> GovinciTabView(node, extra)
        "Modal" -> GovinciModal(node)
        "Image" -> AsyncImage(
            model = node.stringProp("src"),
            contentDescription = node.stringProp("alt").ifEmpty { null },
            modifier = style.boxModifier(extra),
        )

        // Camera capture needs a CameraX integration pass of its own; until
        // then render the styled surface and any overlay so layouts hold up.
        "CameraView" -> Box(style.boxModifier(extra)) { RenderChildren(node) }

        // Fragment and Theme are grouping nodes with no visual box of their
        // own: emit the children inline into whatever scope we're in.
        "Fragment", "Theme" -> RenderChildren(node)

        // Unknown node type (newer Go core than this runtime): render the
        // children so the subtree isn't a dead end.
        else -> Column(style.boxModifier(extra)) { ColumnChildren(node) }
    }
}

/** Children of a non-flex container; keyed so reorder/replace keeps sibling state. */
@Composable
private fun RenderChildren(node: GovinciNode) {
    node.children.forEachIndexed { i, child ->
        key(child.key.ifEmpty { i }) { RenderNode(child) }
    }
}

// ---------------------------------------------------------------------------
// Leaf components
// ---------------------------------------------------------------------------

@Composable
private fun GovinciText(node: GovinciNode, extra: Modifier) {
    val s = node.style
    Text(
        text = node.stringProp("content"),
        modifier = s.boxModifier(extra),
        style = textStyle(s),
    )
}

private fun textStyle(s: GovinciStyle?): TextStyle {
    if (s == null) return TextStyle.Default
    return TextStyle(
        color = s.textColor ?: Color.Unspecified,
        fontSize = if (s.fontSize > 0f) s.fontSize.sp else TextStyle.Default.fontSize,
        // Go's Weight constants are the CSS numeric scale (200/400/700), which
        // FontWeight accepts directly.
        fontWeight = if (s.fontWeight > 0) FontWeight(s.fontWeight) else null,
        lineHeight = if (s.lineHeight > 0) s.lineHeight.sp else TextStyle.Default.lineHeight,
        textAlign = when (s.align) {
            "center" -> TextAlign.Center
            "end" -> TextAlign.End
            "justify" -> TextAlign.Justify
            else -> TextAlign.Start
        },
    )
}

@Composable
private fun GovinciButton(node: GovinciNode, extra: Modifier) {
    val runtime = LocalGovinciRuntime.current
    val s = node.style
    val onClick = node.stringProp("onClick")
    // Style properties the Go theme owns are fed into material3's slots
    // instead of boxModifier: Button draws its own container, so background/
    // radius/padding must go through its API to keep ripple + a11y correct.
    Button(
        onClick = { if (onClick.isNotEmpty()) runtime.click(onClick) },
        modifier = marginAndSize(s, extra),
        shape = RoundedCornerShape((s?.borderRadius ?: 8f).dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = s?.background ?: Color.Unspecified,
            contentColor = s?.textColor ?: Color.Unspecified,
        ),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(
            start = (s?.padding?.left ?: 16).dp, top = (s?.padding?.top ?: 10).dp,
            end = (s?.padding?.right ?: 16).dp, bottom = (s?.padding?.bottom ?: 10).dp,
        ),
    ) {
        Text(
            node.stringProp("label"),
            fontSize = if ((s?.fontSize ?: 0f) > 0f) s!!.fontSize.sp else 17.sp,
            fontWeight = if ((s?.fontWeight ?: 0) > 0) FontWeight(s!!.fontWeight) else null,
        )
    }
}

/** Margin + explicit dimensions only — for components that draw their own box. */
private fun marginAndSize(s: GovinciStyle?, extra: Modifier): Modifier {
    if (s == null) return extra
    // Reuse boxModifier's ordering by building a margin/size-only style.
    val trimmed = s.copy(
        background = null, borderColor = null, borderWidth = 0f,
        borderRadius = 0f, shadow = 0f,
        padding = GovinciStyle.Edges(0, 0, 0, 0),
    )
    return trimmed.boxModifier(extra)
}

@Composable
private fun GovinciCheckbox(node: GovinciNode, extra: Modifier) {
    val runtime = LocalGovinciRuntime.current
    val cb = node.stringProp("onToggle")
    Checkbox(
        checked = node.boolProp("checked"),
        onCheckedChange = { if (cb.isNotEmpty()) runtime.toggled(cb, it) },
        modifier = marginAndSize(node.style, extra),
    )
}

/**
 * The controlled-input compromise: Go owns the value, but the IME needs its
 * keystrokes echoed instantly, and the Go round trip is asynchronous. So the
 * field is locally-owned *while focused* (every edit is sent upstream but
 * late echoes never snap the cursor back), and Go-owned when not focused
 * (an async upstream change — validation rewrites, state restores — lands
 * the moment the user isn't mid-typing).
 */
@Composable
private fun GovinciTextField(
    node: GovinciNode,
    extra: Modifier,
    password: Boolean = false,
    numeric: Boolean = false,
    multiline: Boolean = false,
) {
    val runtime = LocalGovinciRuntime.current
    val s = node.style
    val upstream = node.stringProp("value")
    val onChange = node.stringProp("onChange")

    val interactions = remember { MutableInteractionSource() }
    val focused by interactions.collectIsFocusedAsState()
    var text by remember { mutableStateOf(upstream) }
    if (!focused && text != upstream) text = upstream

    val rows = node.intProp("rows")
    var modifier = s.boxModifier(extra)
    if (multiline && rows > 0) {
        // Approximate a rows-based min height from the line height.
        val line = if (s != null && s.fontSize > 0f) s.fontSize * 1.4f else 24f
        modifier = modifier.heightIn(min = (line * rows).dp)
    }

    BasicTextField(
        value = text,
        onValueChange = {
            text = it
            if (onChange.isNotEmpty()) runtime.textChanged(onChange, it)
        },
        modifier = modifier,
        interactionSource = interactions,
        textStyle = textStyle(s),
        singleLine = !multiline,
        visualTransformation =
            if (password) PasswordVisualTransformation() else VisualTransformation.None,
        keyboardOptions = KeyboardOptions(
            keyboardType = when {
                numeric -> KeyboardType.Number
                password -> KeyboardType.Password
                else -> KeyboardType.Text
            },
        ),
        decorationBox = { inner ->
            Box {
                if (text.isEmpty()) {
                    Text(
                        node.stringProp("placeholder"),
                        style = textStyle(s).copy(color = Color(0x993C3C43)),
                    )
                }
                inner()
            }
        },
    )
}

// ---------------------------------------------------------------------------
// Flex containers
// ---------------------------------------------------------------------------

@Composable
private fun GovinciRow(node: GovinciNode, extra: Modifier) {
    val s = node.style
    Row(
        modifier = s.boxModifier(extra),
        horizontalArrangement = horizontalArrangement(s),
        verticalAlignment = when (s?.alignItems) {
            "center" -> Alignment.CenterVertically
            "flex-end" -> Alignment.Bottom
            else -> Alignment.Top
        },
    ) { RowChildren(node) }
}

@Composable
private fun GovinciColumn(node: GovinciNode, extra: Modifier) {
    val s = node.style
    Column(
        modifier = s.boxModifier(extra),
        verticalArrangement = verticalArrangement(s),
        // AlignItems governs cross-axis placement; the DSL's simpler Align
        // ("center"/"end") acts as a fallback when AlignItems is unset.
        horizontalAlignment = when (s?.alignItems?.ifEmpty { s.align }) {
            "center" -> Alignment.CenterHorizontally
            "flex-end", "end" -> Alignment.End
            else -> Alignment.Start
        },
    ) { ColumnChildren(node) }
}

/**
 * The children loops live inside RowScope/ColumnScope because FlexGrow maps
 * onto Modifier.weight, which exists only as a scope extension — the parent
 * computes it and hands it down as the child's `extra` modifier.
 */
@Composable
private fun RowScope.RowChildren(node: GovinciNode) {
    node.children.forEachIndexed { i, child ->
        key(child.key.ifEmpty { i }) {
            val grow = child.style?.flexGrow ?: 0f
            RenderNode(child, if (grow > 0f) Modifier.weight(grow) else Modifier)
        }
    }
}

@Composable
private fun ColumnScope.ColumnChildren(node: GovinciNode) {
    node.children.forEachIndexed { i, child ->
        key(child.key.ifEmpty { i }) {
            val grow = child.style?.flexGrow ?: 0f
            RenderNode(child, if (grow > 0f) Modifier.weight(grow) else Modifier)
        }
    }
}

private fun horizontalArrangement(s: GovinciStyle?): Arrangement.Horizontal =
    when (s?.justifyContent) {
        "center" -> Arrangement.Center
        "flex-end" -> Arrangement.End
        "space-between" -> Arrangement.SpaceBetween
        "space-around" -> Arrangement.SpaceAround
        "space-evenly" -> Arrangement.SpaceEvenly
        else -> if ((s?.gap ?: 0f) > 0f) Arrangement.spacedBy(s!!.gap.dp) else Arrangement.Start
    }

private fun verticalArrangement(s: GovinciStyle?): Arrangement.Vertical =
    when (s?.justifyContent) {
        "center" -> Arrangement.Center
        "flex-end" -> Arrangement.Bottom
        "space-between" -> Arrangement.SpaceBetween
        "space-around" -> Arrangement.SpaceAround
        "space-evenly" -> Arrangement.SpaceEvenly
        else -> if ((s?.gap ?: 0f) > 0f) Arrangement.spacedBy(s!!.gap.dp) else Arrangement.Top
    }

// ---------------------------------------------------------------------------
// Composite components
// ---------------------------------------------------------------------------

@Composable
private fun GovinciTabView(node: GovinciNode, extra: Modifier) {
    val runtime = LocalGovinciRuntime.current
    val selected = node.intProp("selectedIndex")
    val onTabChange = node.stringProp("onTabChange")

    @Suppress("UNCHECKED_CAST")
    val tabs = node.props["tabs"] as? List<Map<String, Any?>> ?: emptyList()

    Column(node.style.boxModifier(extra)) {
        TabRow(selectedTabIndex = selected.coerceIn(0, (tabs.size - 1).coerceAtLeast(0))) {
            tabs.forEachIndexed { i, tab ->
                Tab(
                    selected = i == selected,
                    onClick = { if (onTabChange.isNotEmpty()) runtime.intChanged(onTabChange, i) },
                    text = { Text(tab["label"] as? String ?: "") },
                )
            }
        }
        // Go renders every tab's content as a child; selection is presentation
        // state, so only the selected child is composed. key() on the index
        // gives each tab its own composition identity, so per-tab state (input
        // text, scroll) is dropped on switch — matching the replace semantics
        // the Go reconciler would apply anyway.
        node.children.getOrNull(selected)?.let { current ->
            key(selected) { RenderNode(current) }
        }
    }
}

@Composable
private fun GovinciModal(node: GovinciNode) {
    if (!node.boolProp("visible")) return
    val runtime = LocalGovinciRuntime.current
    val onDismiss = node.stringProp("onDismiss")
    Dialog(
        onDismissRequest = { if (onDismiss.isNotEmpty()) runtime.click(onDismiss) },
    ) {
        // The dialog window already scrims with the backdrop; the content gets
        // a card-like surface unless the app styled its children explicitly.
        Column(
            Modifier
                .fillMaxWidth()
                .padding(8.dp)
                .background(Color.White, RoundedCornerShape(12.dp))
                .padding(16.dp)
        ) { ColumnChildren(node) }
    }
}
