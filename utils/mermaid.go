package utils

import (
	"regexp"
	"strings"

	mermaid "github.com/AlexanderGrooff/mermaid-ascii/cmd"
)

var mermaidBlockRegex = regexp.MustCompile("(?s)```mermaid\\s*\n(.*?)```")

// ProcessMermaidDiagrams finds ```mermaid code blocks and replaces them with
// ASCII-rendered diagrams. Unsupported diagram types (e.g. stateDiagram,
// classDiagram) are left as-is with a note.
func ProcessMermaidDiagrams(input string) string {
	return mermaidBlockRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := mermaidBlockRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		source := strings.TrimSpace(parts[1])

		rendered, err := mermaid.RenderDiagram(source, nil)
		if err != nil {
			// Unsupported diagram type or parse error — show source with a note
			return "```\n[mermaid diagram — not rendered: " + firstLine(err.Error()) + "]\n\n" + source + "\n```"
		}

		return "```\n" + strings.TrimRight(rendered, "\n") + "\n```"
	})
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
