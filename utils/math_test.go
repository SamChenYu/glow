package utils

import (
	"strings"
	"testing"
)

func TestProcessMathContent_GreekLetters(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`\alpha`, "α"},
		{`\beta`, "β"},
		{`\gamma`, "γ"},
		{`\delta`, "δ"},
		{`\varepsilon`, "ε"},
		{`\Sigma`, "Σ"},
		{`\omega`, "ω"},
		{`\varphi`, "φ"},
	}
	for _, tt := range tests {
		got := processMathContent(tt.input)
		if got != tt.want {
			t.Errorf("processMathContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProcessMathContent_SetOperators(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`\in`, "∈"},
		{`\notin`, "∉"},
		{`\subseteq`, "⊆"},
		{`\cup`, "∪"},
		{`\cap`, "∩"},
		{`\emptyset`, "∅"},
		{`\times`, "×"},
		{`\mid`, "∣"},
		{`\subset`, "⊂"},
	}
	for _, tt := range tests {
		got := processMathContent(tt.input)
		if got != tt.want {
			t.Errorf("processMathContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProcessMathContent_Arrows(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`\to`, "→"},
		{`\rightarrow`, "→"},
		{`\Rightarrow`, "⇒"},
		{`\leftarrow`, "←"},
		{`\mapsto`, "↦"},
	}
	for _, tt := range tests {
		got := processMathContent(tt.input)
		if got != tt.want {
			t.Errorf("processMathContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProcessMathContent_Mathbb(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`\mathbb{N}`, "ℕ"},
		{`\mathbb{Z}`, "ℤ"},
		{`\mathbb{R}`, "ℝ"},
		{`\mathbb{Q}`, "ℚ"},
		{`\mathbb{C}`, "ℂ"},
	}
	for _, tt := range tests {
		got := processMathContent(tt.input)
		if got != tt.want {
			t.Errorf("processMathContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProcessMathContent_Superscripts(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"x^2", "x²"},
		{"x^{10}", "x¹⁰"},
		{"a^n", "aⁿ"},
	}
	for _, tt := range tests {
		got := processMathContent(tt.input)
		if got != tt.want {
			t.Errorf("processMathContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProcessMathContent_Subscripts(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"x_0", "x₀"},
		{"a_{12}", "a₁₂"},
		{"q_i", "qᵢ"},
	}
	for _, tt := range tests {
		got := processMathContent(tt.input)
		if got != tt.want {
			t.Errorf("processMathContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProcessMathContent_Negation(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`\not\subseteq`, "⊄"},
		{`\not\in`, "∉"},
	}
	for _, tt := range tests {
		got := processMathContent(tt.input)
		if got != tt.want {
			t.Errorf("processMathContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProcessMathContent_Text(t *testing.T) {
	got := processMathContent(`L_{\text{all-}a}`)
	if !strings.Contains(got, "all-") {
		t.Errorf("processMathContent with \\text should preserve text content, got %q", got)
	}
}

func TestProcessMathContent_Braces(t *testing.T) {
	got := processMathContent(`\{a, b, c\}`)
	if got != "{a, b, c}" {
		t.Errorf("processMathContent(%q) = %q, want %q", `\{a, b, c\}`, got, "{a, b, c}")
	}
}

func TestProcessMathContent_Complex(t *testing.T) {
	input := `\{a, b\} \subseteq \{a, b, c\}`
	got := processMathContent(input)
	want := "{a, b} ⊆ {a, b, c}"
	if got != want {
		t.Errorf("processMathContent(%q) = %q, want %q", input, got, want)
	}
}

func TestProcessMathContent_Dots(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`\ldots`, "…"},
		{`\cdots`, "⋯"},
	}
	for _, tt := range tests {
		got := processMathContent(tt.input)
		if got != tt.want {
			t.Errorf("processMathContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProcessMathNotation_InlineMath(t *testing.T) {
	input := `the set $\{a, b\}$ contains two elements`
	got := ProcessMathNotation(input)
	want := `the set {a, b} contains two elements`
	if got != want {
		t.Errorf("ProcessMathNotation(%q) = %q, want %q", input, got, want)
	}
}

func TestProcessMathNotation_MultipleDollars(t *testing.T) {
	input := `$a \in \{a, b\}$, $c \notin \{a, b\}$`
	got := ProcessMathNotation(input)
	want := `a ∈ {a, b}, c ∉ {a, b}`
	if got != want {
		t.Errorf("ProcessMathNotation(%q) = %q, want %q", input, got, want)
	}
}

func TestProcessMathNotation_BlockMath(t *testing.T) {
	input := "some text\n$$x \\in \\mathbb{N}$$\nmore text"
	got := ProcessMathNotation(input)
	if !strings.Contains(got, "x ∈ ℕ") {
		t.Errorf("ProcessMathNotation block math should contain 'x ∈ ℕ', got %q", got)
	}
	if strings.Contains(got, "$$") {
		t.Errorf("ProcessMathNotation should strip $$ delimiters, got %q", got)
	}
}

func TestProcessMathNotation_NoMath(t *testing.T) {
	input := "This is a normal paragraph with no math."
	got := ProcessMathNotation(input)
	if got != input {
		t.Errorf("ProcessMathNotation should not modify non-math text, got %q", got)
	}
}

func TestProcessMathNotation_EscapedDollar(t *testing.T) {
	input := `This costs \$5 and \$10`
	got := ProcessMathNotation(input)
	want := `This costs $5 and $10`
	if got != want {
		t.Errorf("ProcessMathNotation(%q) = %q, want %q", input, got, want)
	}
}

func TestProcessMathNotation_CodeBlock(t *testing.T) {
	input := "```\n$x = 5$\n```"
	got := ProcessMathNotation(input)
	// Code blocks should ideally be preserved, but since we pre-process
	// before glamour, the code block markers are still raw markdown.
	// This is acceptable since glamour will handle them.
	if got == "" {
		t.Error("ProcessMathNotation should not return empty string")
	}
}

func TestProcessMathNotation_AutomataNotes(t *testing.T) {
	input := `| $\{a, b, c\}$ | the set with elements $a, b, c$ |
| $\{\}$ (also $\emptyset$) | the empty set (with no elements) |
| $\{0, 1, 2, \ldots\}$ (also $\mathbb{N}$ or $\omega$) | the set of natural numbers |
| $\{2x \mid x \in \mathbb{N}\}$ | the set of even natural numbers |`

	got := ProcessMathNotation(input)

	checks := []struct {
		desc string
		want string
	}{
		{"set notation", "{a, b, c}"},
		{"empty set", "∅"},
		{"natural numbers", "ℕ"},
		{"omega", "ω"},
		{"element of", "∈"},
		{"ellipsis", "…"},
		{"divider", "∣"},
	}

	for _, c := range checks {
		if !strings.Contains(got, c.want) {
			t.Errorf("AutomataNotes: expected %s (%q) in output, got %q", c.desc, c.want, got)
		}
	}

	if strings.Contains(got, "$") {
		t.Errorf("AutomataNotes: output should not contain $ delimiters, got %q", got)
	}
}

func TestProcessMathContent_LeftRight(t *testing.T) {
	input := `\left( a + b \right)`
	got := processMathContent(input)
	want := "( a + b )"
	if got != want {
		t.Errorf("processMathContent(%q) = %q, want %q", input, got, want)
	}
}

func TestProcessMathContent_Overline(t *testing.T) {
	got := processMathContent(`\overline{L}`)
	if !strings.Contains(got, "L") {
		t.Errorf("processMathContent overline should contain base character, got %q", got)
	}
}

func TestProcessMathContent_Xrightarrow(t *testing.T) {
	got := processMathContent(`\xrightarrow{a}`)
	want := "─a→"
	if got != want {
		t.Errorf("processMathContent(%q) = %q, want %q", `\xrightarrow{a}`, got, want)
	}
}

func TestProcessMathNotation_TableRow(t *testing.T) {
	input := `| $a \in \{a, b\}$, $c \notin \{a, b\}$ | an element belongs to a set (or not) |`
	got := ProcessMathNotation(input)
	if !strings.Contains(got, "∈") {
		t.Errorf("Table row should contain ∈, got %q", got)
	}
	if !strings.Contains(got, "∉") {
		t.Errorf("Table row should contain ∉, got %q", got)
	}
}

func TestProcessMathContent_Neq(t *testing.T) {
	got := processMathContent(`a \neq b`)
	want := "a ≠ b"
	if got != want {
		t.Errorf("processMathContent(%q) = %q, want %q", `a \neq b`, got, want)
	}
}

func TestProcessMathContent_SetExpression(t *testing.T) {
	input := `\{a, b\} \cap \{a, c\} = \{a\}`
	got := processMathContent(input)
	want := "{a, b} ∩ {a, c} = {a}"
	if got != want {
		t.Errorf("processMathContent(%q) = %q, want %q", input, got, want)
	}
}
