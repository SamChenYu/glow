package utils

import (
	"regexp"
	"strings"
)

var latexCommands = map[string]string{
	// Greek letters (lowercase)
	`\alpha`:      "α",
	`\beta`:       "β",
	`\gamma`:      "γ",
	`\delta`:      "δ",
	`\epsilon`:    "ϵ",
	`\varepsilon`: "ε",
	`\zeta`:       "ζ",
	`\eta`:        "η",
	`\theta`:      "θ",
	`\iota`:       "ι",
	`\kappa`:      "κ",
	`\lambda`:     "λ",
	`\mu`:         "μ",
	`\nu`:         "ν",
	`\xi`:         "ξ",
	`\pi`:         "π",
	`\rho`:        "ρ",
	`\sigma`:      "σ",
	`\tau`:        "τ",
	`\upsilon`:    "υ",
	`\phi`:        "ϕ",
	`\varphi`:     "φ",
	`\chi`:        "χ",
	`\psi`:        "ψ",
	`\omega`:      "ω",

	// Greek letters (uppercase)
	`\Gamma`:   "Γ",
	`\Delta`:   "Δ",
	`\Theta`:   "Θ",
	`\Lambda`:  "Λ",
	`\Xi`:      "Ξ",
	`\Pi`:      "Π",
	`\Sigma`:   "Σ",
	`\Upsilon`: "Υ",
	`\Phi`:     "Φ",
	`\Psi`:     "Ψ",
	`\Omega`:   "Ω",

	// Set and logic operators
	`\in`:         "∈",
	`\notin`:      "∉",
	`\subset`:     "⊂",
	`\supset`:     "⊃",
	`\subseteq`:   "⊆",
	`\supseteq`:   "⊇",
	`\cup`:        "∪",
	`\cap`:        "∩",
	`\emptyset`:   "∅",
	`\varnothing`: "∅",
	`\setminus`:   "∖",
	`\mid`:        "∣",
	`\land`:       "∧",
	`\lor`:        "∨",
	`\lnot`:       "¬",
	`\neg`:        "¬",
	`\forall`:     "∀",
	`\exists`:     "∃",
	`\nexists`:    "∄",
	`\top`:        "⊤",
	`\bot`:        "⊥",
	`\vdash`:      "⊢",
	`\models`:     "⊨",
	`\times`:      "×",
	`\circ`:       "∘",
	`\cdot`:       "·",

	// Relations
	`\neq`:   "≠",
	`\leq`:   "≤",
	`\geq`:   "≥",
	`\equiv`: "≡",
	`\approx`: "≈",
	`\sim`:    "∼",
	`\simeq`:  "≃",
	`\cong`:   "≅",
	`\propto`: "∝",
	`\ll`:     "≪",
	`\gg`:     "≫",
	`\prec`:   "≺",
	`\succ`:   "≻",

	// Arrows
	`\to`:             "→",
	`\rightarrow`:     "→",
	`\leftarrow`:      "←",
	`\leftrightarrow`: "↔",
	`\Rightarrow`:     "⇒",
	`\Leftarrow`:      "⇐",
	`\Leftrightarrow`: "⇔",
	`\mapsto`:         "↦",
	`\uparrow`:        "↑",
	`\downarrow`:      "↓",
	`\nearrow`:        "↗",
	`\searrow`:        "↘",

	// Text-mode symbols (use fullwidth $ so the inline math regex doesn't re-consume it as a delimiter)
	`\textdollar`: "＄",
	`\ast`:        "∗",

	// Miscellaneous
	`\infty`:    "∞",
	`\partial`:  "∂",
	`\nabla`:    "∇",
	`\pm`:       "±",
	`\mp`:       "∓",
	`\star`:     "⋆",
	`\dagger`:   "†",
	`\ddagger`:  "‡",
	`\ell`:      "ℓ",
	`\hbar`:     "ℏ",
	`\Re`:       "ℜ",
	`\Im`:       "ℑ",
	`\aleph`:    "ℵ",
	`\wp`:       "℘",
	`\triangle`: "△",

	// Dots
	`\ldots`: "…",
	`\cdots`: "⋯",
	`\vdots`: "⋮",
	`\ddots`: "⋱",

	// Big operators
	`\sum`:     "∑",
	`\prod`:    "∏",
	`\coprod`:  "∐",
	`\int`:     "∫",
	`\oint`:    "∮",
	`\bigcup`:  "⋃",
	`\bigcap`:  "⋂",
	`\bigoplus`: "⨁",
	`\bigotimes`: "⨂",

	// Brackets (LaTeX escaped)
	`\{`: "{",
	`\}`: "}",
	`\langle`: "⟨",
	`\rangle`: "⟩",
	`\lfloor`: "⌊",
	`\rfloor`: "⌋",
	`\lceil`:  "⌈",
	`\rceil`:  "⌉",

	// Spacing
	`\quad`:  "  ",
	`\qquad`: "    ",
}

var mathbbMap = map[rune]string{
	'A': "𝔸", 'B': "𝔹", 'C': "ℂ", 'D': "𝔻", 'E': "𝔼",
	'F': "𝔽", 'G': "𝔾", 'H': "ℍ", 'I': "𝕀", 'J': "𝕁",
	'K': "𝕂", 'L': "𝕃", 'M': "𝕄", 'N': "ℕ", 'O': "𝕆",
	'P': "ℙ", 'Q': "ℚ", 'R': "ℝ", 'S': "𝕊", 'T': "𝕋",
	'U': "𝕌", 'V': "𝕍", 'W': "𝕎", 'X': "𝕏", 'Y': "𝕐",
	'Z': "ℤ",
}

var superscriptMap = map[rune]rune{
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
	'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	'+': '⁺', '-': '⁻', '=': '⁼', '(': '⁽', ')': '⁾',
	'n': 'ⁿ', 'i': 'ⁱ', '*': '˟',
}

var subscriptMap = map[rune]rune{
	'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄',
	'5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
	'+': '₊', '-': '₋', '=': '₌', '(': '₍', ')': '₎',
	'a': 'ₐ', 'e': 'ₑ', 'o': 'ₒ', 'x': 'ₓ',
	'i': 'ᵢ', 'j': 'ⱼ', 'k': 'ₖ',
}

var (
	mathbbRegex        = regexp.MustCompile(`\\mathbb\{([A-Z])\}`)
	textRegex          = regexp.MustCompile(`\\text\{([^}]*)\}`)
	overlineRegex      = regexp.MustCompile(`\\overline\{([^}]*)\}`)
	underlineRegex     = regexp.MustCompile(`\\underline\{([^}]*)\}`)
	underbaceRegex     = regexp.MustCompile(`\\underbrace\{([^}]*)\}`)
	xrightarrowRegex   = regexp.MustCompile(`\\xrightarrow\{([^}]*)\}`)
	superscriptRegex   = regexp.MustCompile(`\^(\{[^}]+\}|[0-9a-zA-Z*])`)
	subscriptRegex     = regexp.MustCompile(`_(\{[^}]+\}|[0-9a-zA-Z])`)
	leftRightRegex     = regexp.MustCompile(`\\(?:left|right)([^a-zA-Z])`)
	notCommandRegex    = regexp.MustCompile(`\\not\\([a-zA-Z]+)`)
	beginEndRegex      = regexp.MustCompile(`\\(?:begin|end)\{[^}]*\}`)
	inlineMathRegex    = regexp.MustCompile(`(?s)(^|[^\\])\$\$(.+?)\$\$`)
	inlineSingleRegex  = regexp.MustCompile(`(^|[^\\$])\$([^$\n]+?)\$`)
)

var negationMap = map[string]string{
	`\subseteq`: "⊄",
	`\subset`:   "⊄",
	`\supseteq`: "⊅",
	`\supset`:   "⊅",
	`\in`:       "∉",
	`\equiv`:    "≢",
	`\sim`:      "≁",
}

func processMathContent(math string) string {
	s := math

	// Handle \not\command (e.g., \not\subseteq)
	s = notCommandRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := notCommandRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			cmd := `\` + parts[1]
			if neg, ok := negationMap[cmd]; ok {
				return neg
			}
		}
		return match
	})

	// Handle \mathbb{X}
	s = mathbbRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := mathbbRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			r := rune(parts[1][0])
			if repl, ok := mathbbMap[r]; ok {
				return repl
			}
		}
		return match
	})

	// Handle \text{...} — just extract the text
	s = textRegex.ReplaceAllString(s, "$1")

	// Handle \overline{...} — use combining overline
	s = overlineRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := overlineRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			inner := processMathContent(parts[1])
			var result strings.Builder
			for _, r := range inner {
				result.WriteRune(r)
				result.WriteRune('̅') // combining overline
			}
			return result.String()
		}
		return match
	})

	// Handle \underline{...} — use combining underline
	s = underlineRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := underlineRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			inner := processMathContent(parts[1])
			var result strings.Builder
			for _, r := range inner {
				result.WriteRune(r)
				result.WriteRune('̲') // combining underline
			}
			return result.String()
		}
		return match
	})

	// Handle \underbrace{...} — just show content
	s = underbaceRegex.ReplaceAllString(s, "$1")

	// Handle \xrightarrow{...} — arrow with label
	s = xrightarrowRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := xrightarrowRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			label := processMathContent(parts[1])
			return "─" + label + "→"
		}
		return match
	})

	// Handle \begin{...} and \end{...} — remove
	s = beginEndRegex.ReplaceAllString(s, "")

	// Handle \left and \right — just keep the delimiter
	s = leftRightRegex.ReplaceAllString(s, "$1")

	// Replace LaTeX commands with Unicode (longest match first via sorted iteration)
	// Process longer commands first to avoid partial matches
	for _, cmd := range sortedLatexCommands {
		if strings.Contains(s, cmd) {
			s = strings.ReplaceAll(s, cmd, latexCommands[cmd])
		}
	}

	// Handle superscripts: ^{abc} or ^x
	s = superscriptRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := superscriptRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			content := parts[1]
			if strings.HasPrefix(content, "{") {
				content = content[1 : len(content)-1]
			}
			var result strings.Builder
			converted := true
			for _, r := range content {
				if sup, ok := superscriptMap[r]; ok {
					result.WriteRune(sup)
				} else {
					converted = false
					break
				}
			}
			if converted {
				return result.String()
			}
			return "^(" + content + ")"
		}
		return match
	})

	// Handle subscripts: _{abc} or _x
	s = subscriptRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := subscriptRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			content := parts[1]
			if strings.HasPrefix(content, "{") {
				content = content[1 : len(content)-1]
			}
			var result strings.Builder
			converted := true
			for _, r := range content {
				if sub, ok := subscriptMap[r]; ok {
					result.WriteRune(sub)
				} else {
					converted = false
					break
				}
			}
			if converted {
				return result.String()
			}
			return "_(" + content + ")"
		}
		return match
	})

	return strings.TrimSpace(s)
}

// ProcessMathNotation replaces LaTeX math notation with Unicode equivalents.
// It handles both inline ($...$) and block ($$...$$) math.
func ProcessMathNotation(input string) string {
	// Process block math ($$...$$) first
	s := inlineMathRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := inlineMathRegex.FindStringSubmatch(match)
		if len(parts) == 3 {
			prefix := parts[1]
			content := processMathContent(parts[2])
			return prefix + content
		}
		return match
	})

	// Process inline math ($...$)
	s = inlineSingleRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := inlineSingleRegex.FindStringSubmatch(match)
		if len(parts) == 3 {
			prefix := parts[1]
			content := processMathContent(parts[2])
			return prefix + content
		}
		return match
	})

	// Clean up escaped dollar signs
	s = strings.ReplaceAll(s, `\$`, "$")

	return s
}

// sortedLatexCommands is pre-sorted longest-first to avoid partial matches.
var sortedLatexCommands []string

func init() {
	sortedLatexCommands = make([]string, 0, len(latexCommands))
	for cmd := range latexCommands {
		sortedLatexCommands = append(sortedLatexCommands, cmd)
	}
	// Sort by length descending, then alphabetically for stability
	for i := 0; i < len(sortedLatexCommands); i++ {
		for j := i + 1; j < len(sortedLatexCommands); j++ {
			if len(sortedLatexCommands[j]) > len(sortedLatexCommands[i]) ||
				(len(sortedLatexCommands[j]) == len(sortedLatexCommands[i]) &&
					sortedLatexCommands[j] < sortedLatexCommands[i]) {
				sortedLatexCommands[i], sortedLatexCommands[j] = sortedLatexCommands[j], sortedLatexCommands[i]
			}
		}
	}
}
