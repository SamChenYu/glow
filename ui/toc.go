package ui

import (
	"strings"

	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/truncate"
)

type tocHeader struct {
	level        int
	title        string
	srcLine      int // 0-based line in the frontmatter-stripped source
	renderedLine int
}

func parseHeaders(body string) []tocHeader {
	clean := string(utils.RemoveFrontmatter([]byte(body)))
	lines := strings.Split(clean, "\n")
	var headers []tocHeader
	inCodeBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		level := 0
		for _, c := range trimmed {
			if c == '#' {
				level++
			} else {
				break
			}
		}
		if level > 0 && level <= 6 && len(trimmed) > level && trimmed[level] == ' ' {
			title := strings.TrimSpace(trimmed[level+1:])
			if title != "" {
				headers = append(headers, tocHeader{
					level:        level,
					title:        title,
					srcLine:      i,
					renderedLine: -1,
				})
			}
		}
	}
	return headers
}

func findHeaderLines(headers []tocHeader, contentLines []string) {
	searchFrom := 0
	for i := range headers {
		title := strings.ToLower(strings.TrimSpace(headers[i].title))
		for j := searchFrom; j < len(contentLines); j++ {
			plain := strings.ToLower(strings.TrimSpace(xansi.Strip(contentLines[j])))
			if strings.Contains(plain, title) {
				headers[i].renderedLine = j
				searchFrom = j + 1
				break
			}
		}
	}
}

var (
	tocTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(mintGreen)

	tocSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"}).
				Background(lipgloss.AdaptiveColor{Light: "#1C8760", Dark: "#1C8760"})

	tocNormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#DDDDDD"})

	tocFooterStyle = lipgloss.NewStyle().
			Foreground(statusBarNoteFg)
)

func (m *pagerModel) buildTOCContent() string {
	contentWidth := max(0, m.common.width-6)

	var b strings.Builder
	b.WriteString("\n  " + tocTitleStyle.Render("Table of Contents") + "\n\n")

	for i, hdr := range m.tocHeaders {
		ind := strings.Repeat("  ", hdr.level-1)
		prefix := "  "
		if i == m.tocSelectedIdx {
			prefix = "▸ "
		}

		label := ind + prefix + hdr.title
		if contentWidth > 0 {
			label = truncate.StringWithTail(label, uint(contentWidth), ellipsis) //nolint:gosec
		}
		padded := label + strings.Repeat(" ", max(0, contentWidth-lipgloss.Width(label)))

		if i == m.tocSelectedIdx {
			b.WriteString("  " + tocSelectedStyle.Render(padded) + "\n")
		} else {
			b.WriteString("  " + tocNormalStyle.Render(padded) + "\n")
		}
	}

	b.WriteString("\n" + tocFooterStyle.Render("  ↑↓/jk navigate • enter jump • esc close"))
	return b.String()
}

func (m *pagerModel) enterTOC() {
	m.tocSavedYOffset = m.viewport.YOffset
	m.tocSelectedIdx = 0
	m.state = pagerStateTOC
	m.setContent(m.buildTOCContent())
	m.viewport.SetYOffset(0)
}

func (m *pagerModel) exitTOC(jumpToHeader bool) {
	m.state = pagerStateBrowse
	m.setContent(strings.Join(m.contentLines, "\n"))
	if jumpToHeader && m.tocSelectedIdx >= 0 && m.tocSelectedIdx < len(m.tocHeaders) {
		if line := m.tocHeaders[m.tocSelectedIdx].renderedLine; line >= 0 {
			m.viewport.SetYOffset(line)
			return
		}
	}
	m.viewport.SetYOffset(m.tocSavedYOffset)
}

func (m *pagerModel) updateTOCSelection() {
	m.setContent(m.buildTOCContent())
	m.viewport.SetYOffset(0)
}
