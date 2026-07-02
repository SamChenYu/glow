package utils

import "strings"

// wideGlyphReplacer rewrites East-Asian-Wide decorative glyphs to unambiguous
// width-1 (EAW=Narrow) look-alikes.
//
// glamour measures table cell widths via charmbracelet/x/ansi.StringWidth in
// GraphemeWidth (uniseg) mode, which reserves 2 columns for these glyphs. Many
// terminal fonts advance only 1 column for them, so a cell containing such a
// glyph renders 1 column narrower than glamour padded it — shifting every table
// separator and wrapped-continuation line (which lack the glyph) by 1 column.
// A Narrow replacement is counted as 1 by glamour AND drawn as 1 by the terminal,
// so the two agree and columns stay aligned.
//
// The map is intentionally small and easy to extend with other wide markers.
// VS16 (U+FE0F) clusters must be listed as their full two-rune sequence.
var wideGlyphReplacer = strings.NewReplacer(
	"✔️", "✓", // ✔️ HEAVY CHECK MARK + VS16   -> ✓ CHECK MARK (N)
	"▫️", "☐", // ▫️ WHITE SMALL SQUARE + VS16 -> ☐ BALLOT BOX (N)
	"✅", "✓", // ✅ WHITE HEAVY CHECK MARK (W) -> ✓ CHECK MARK (N)
	"⬛", "☑", // ⬛ BLACK LARGE SQUARE (W)      -> ☑ BALLOT BOX WITH CHECK (N)
	"⬜", "☐", // ⬜ WHITE LARGE SQUARE (W)      -> ☐ BALLOT BOX (N)
)

// NormalizeTableGlyphs rewrites East-Asian-Wide decorative glyphs to width-1
// look-alikes, but only on GFM table-row lines, so tables stay column-aligned
// regardless of how the terminal font advances those glyphs. Glyphs in prose
// (e.g. ✅ in callouts) and inside fenced code blocks are left untouched.
func NormalizeTableGlyphs(content string) string {
	lines := strings.Split(content, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track fenced code blocks; never rewrite glyphs inside them.
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		// A GFM table row (including the |---| separator) starts with a pipe.
		if strings.HasPrefix(trimmed, "|") {
			lines[i] = wideGlyphReplacer.Replace(line)
		}
	}
	return strings.Join(lines, "\n")
}
