package utils

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNormalizeTableGlyphs_TableRowReplaced(t *testing.T) {
	tests := []struct {
		input   string
		absent  string
		present string
	}{
		{"| ⬛ | Concept | Why it matters |", "⬛", "☑"},
		{"| ▫️ | Concept | Why it matters |", "▫️", "☐"},
		{"| ⬜ | Concept | Why it matters |", "⬜", "☐"},
		{"| ✅ | done | ship it |", "✅", "✓"},
		{"  | ⬛ | indented table row |", "⬛", "☑"}, // leading whitespace still a table row
	}
	for _, tt := range tests {
		got := NormalizeTableGlyphs(tt.input)
		if strings.Contains(got, tt.absent) {
			t.Errorf("NormalizeTableGlyphs(%q) = %q, should not contain %q", tt.input, got, tt.absent)
		}
		if !strings.Contains(got, tt.present) {
			t.Errorf("NormalizeTableGlyphs(%q) = %q, should contain %q", tt.input, got, tt.present)
		}
	}
}

func TestNormalizeTableGlyphs_ProseUntouched(t *testing.T) {
	// A callout / prose line (no leading pipe) must keep its original glyphs.
	input := "✅ Checkpoint: you can explain why no program can decide halting."
	got := NormalizeTableGlyphs(input)
	if got != input {
		t.Errorf("NormalizeTableGlyphs should not touch prose, got %q", got)
	}
}

func TestNormalizeTableGlyphs_FenceUntouched(t *testing.T) {
	// A table-looking line inside a fenced code block must be left as-is.
	input := "```\n| ⬛ | not a real table, it's code |\n```"
	got := NormalizeTableGlyphs(input)
	if !strings.Contains(got, "⬛") {
		t.Errorf("NormalizeTableGlyphs should not rewrite glyphs inside code fences, got %q", got)
	}
	if strings.Contains(got, "☑") {
		t.Errorf("NormalizeTableGlyphs rewrote a glyph inside a code fence, got %q", got)
	}
}

func TestNormalizeTableGlyphs_SeparatorUnchanged(t *testing.T) {
	input := "|---|---|---|"
	got := NormalizeTableGlyphs(input)
	if got != input {
		t.Errorf("separator row should be unchanged, got %q", got)
	}
}

func TestNormalizeTableGlyphs_MultilineTable(t *testing.T) {
	input := strings.Join([]string{
		"| ⬛/▫️ | Concept | Resource |",
		"|---|---|---|",
		"| ⬛ | Product construction | Sipser §1.1 |",
		"",
		"✅ Checkpoint outside the table stays ✅.",
	}, "\n")
	got := NormalizeTableGlyphs(input)
	if strings.Contains(got, "⬛") || strings.Contains(got, "▫️") {
		t.Errorf("table rows should have wide glyphs normalized, got %q", got)
	}
	if !strings.Contains(got, "✅ Checkpoint outside the table stays ✅.") {
		t.Errorf("prose line should keep ✅, got %q", got)
	}
}

// TestNormalizeTableGlyphs_ReplacementsAreWidth1 is the crux of the fix: every
// replacement glyph must measure exactly 1 column via the same width function
// glamour/lipgloss uses, otherwise the drift would persist.
func TestNormalizeTableGlyphs_ReplacementsAreWidth1(t *testing.T) {
	for _, g := range []string{"☑", "☐", "✓"} {
		if w := lipgloss.Width(g); w != 1 {
			t.Errorf("replacement glyph %q has width %d, want 1", g, w)
		}
	}
	// Sanity check that the originals really were the width-2 offenders.
	for _, g := range []string{"⬛", "⬜", "✅"} {
		if w := lipgloss.Width(g); w != 2 {
			t.Errorf("expected wide glyph %q to have width 2 (the bug), got %d", g, w)
		}
	}
}
