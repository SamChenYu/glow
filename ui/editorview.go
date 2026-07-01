package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/ansi"
)

const (
	editorGutterWidth    = 1
	editorMinPaneWidth   = 20
	editorRenderDebounce = 180 * time.Millisecond
)

// editorSubState distinguishes normal editing from the confirm-before-quit
// prompt shown when there are unsaved changes.
type editorSubState int

const (
	editorStateEditing editorSubState = iota
	editorStateConfirmQuit
)

type (
	// editorRenderTickMsg fires after the debounce interval; the preview is
	// only re-rendered if gen still matches the model's current generation.
	editorRenderTickMsg struct{ gen int }
	// editorPreviewRenderedMsg carries freshly-rendered preview content.
	editorPreviewRenderedMsg struct {
		gen     int
		content string
	}
	// editorSavedMsg tells the parent model the document was written to disk,
	// so it can keep the pager's copy in sync.
	editorSavedMsg struct{ body string }
	// editorQuitMsg tells the parent model to leave the editor.
	editorQuitMsg struct{}
)

// scrollAnchor pairs a source line with the rendered line it maps to. Anchors
// are taken at headings (whose rendered position we can locate exactly), so the
// cursor->preview mapping is exact at section boundaries and only interpolated
// within a section.
type scrollAnchor struct {
	src      int
	rendered int
}

// editorModel is an in-app split-pane markdown editor: a text editor on the
// left and a live glamour-rendered preview on the right.
type editorModel struct {
	common   *commonModel
	textarea textarea.Model
	preview  viewport.Model

	localPath string
	note      string // used for isCode detection; mirrors markdown.Note

	savedContent string // last content persisted to disk (dirty-detection baseline)
	lastRendered string // last text rendered into the preview (skip redundant renders)

	renderGen int // bumped on every content change; only the latest render applies

	// Preview scroll-follow state, rebuilt on each render.
	anchors          []scrollAnchor // (source line -> rendered line) at each heading
	fmOffset         int            // source lines removed as frontmatter (not rendered)
	postSrcLineCount int            // source line count after frontmatter

	subState editorSubState
	origin   state // where to return on exit (stash or document)

	width, height int
	statusMsg     string // transient, e.g. "Saved"
	err           error
}

func newEditorModel(common *commonModel, path, body, note string, origin state) editorModel {
	ta := textarea.New()
	ta.ShowLineNumbers = true
	ta.MaxHeight = 0 // 0 = unlimited; the default (99) would truncate large files
	ta.MaxWidth = 0  // 0 = unlimited; width is bounded via SetWidth
	ta.SetValue(body)
	// Position the cursor at the very top of the document.
	for ta.Line() > 0 {
		ta.CursorUp()
	}
	ta.CursorStart()
	// Focus here (not just in Init) so the stored model is focused: Init has a
	// value receiver, so a Focus() there would mutate a throwaway copy and the
	// textarea would silently ignore input.
	ta.Focus()

	vp := viewport.New(0, 0)
	vp.HighPerformanceRendering = false

	m := editorModel{
		common:       common,
		textarea:     ta,
		preview:      vp,
		localPath:    path,
		note:         note,
		savedContent: body,
		lastRendered: body,
		renderGen:    1,
		subState:     editorStateEditing,
		origin:       origin,
	}
	m.setSize(common.width, common.height)
	return m
}

func (m editorModel) dirty() bool { return m.textarea.Value() != m.savedContent }

func (m editorModel) Init() tea.Cmd {
	// Focus the editor and render the initial preview immediately (rather than
	// waiting out the debounce).
	return tea.Batch(m.textarea.Focus(), m.renderPreview(m.renderGen))
}

// setSize lays out the editor (left) and preview (right) as a ~50/50 split. On
// terminals too narrow for both panes, the preview is dropped and the editor
// takes the full width (mirroring how the pager suppresses the minimap).
func (m *editorModel) setSize(w, h int) {
	m.width, m.height = w, h
	contentHeight := max(1, h-statusBarHeight)

	if w < 2*editorMinPaneWidth+editorGutterWidth {
		m.textarea.SetWidth(w)
		m.textarea.SetHeight(contentHeight)
		m.preview.Width = 0
		m.preview.Height = contentHeight
		return
	}

	leftW := (w - editorGutterWidth) / 2
	rightW := w - editorGutterWidth - leftW

	m.textarea.SetWidth(leftW)
	m.textarea.SetHeight(contentHeight)
	m.preview.Width = rightW
	m.preview.Height = contentHeight
	m.syncPreviewScroll()
}

// rebuildAnchors recomputes the source->rendered line anchors from the rendered
// content and the text that produced it (lastRendered). Anchors are taken at
// each heading, whose rendered line we locate exactly via findHeaderLines.
func (m *editorModel) rebuildAnchors(rendered string) {
	text := m.lastRendered
	stripped := string(utils.RemoveFrontmatter([]byte(text)))
	m.fmOffset = strings.Count(text, "\n") - strings.Count(stripped, "\n")
	m.postSrcLineCount = strings.Count(stripped, "\n")

	renderedLines := strings.Split(rendered, "\n")
	headers := parseHeaders(text) // srcLine is in frontmatter-stripped coordinates
	findHeaderLines(headers, renderedLines)

	// Always anchor the top; append each locatable heading, keeping src and
	// rendered strictly/again monotonic; finally anchor the end.
	anchors := []scrollAnchor{{src: 0, rendered: 0}}
	for _, h := range headers {
		if h.renderedLine < 0 {
			continue
		}
		last := anchors[len(anchors)-1]
		if h.srcLine > last.src && h.renderedLine >= last.rendered {
			anchors = append(anchors, scrollAnchor{src: h.srcLine, rendered: h.renderedLine})
		}
	}
	total := len(renderedLines)
	if last := anchors[len(anchors)-1]; m.postSrcLineCount > last.src && total-1 >= last.rendered {
		anchors = append(anchors, scrollAnchor{src: m.postSrcLineCount, rendered: total - 1})
	}
	m.anchors = anchors
}

// cursorRenderedLine maps the cursor's source line to a rendered line using the
// anchors, interpolating linearly within an anchor segment. With no headings
// this degrades to a single top->bottom proportional segment.
func (m *editorModel) cursorRenderedLine(total int) int {
	cur := m.textarea.Line() - m.fmOffset
	if cur < 0 {
		cur = 0
	}

	a := m.anchors
	if len(a) < 2 {
		lines := max(1, m.postSrcLineCount)
		return int(float64(cur) / float64(max(1, lines)) * float64(max(0, total-1)))
	}
	if cur <= a[0].src {
		return a[0].rendered
	}
	if cur >= a[len(a)-1].src {
		return a[len(a)-1].rendered
	}
	for i := 0; i+1 < len(a); i++ {
		lo, hi := a[i], a[i+1]
		if cur >= lo.src && cur <= hi.src {
			if hi.src == lo.src {
				return lo.rendered
			}
			frac := float64(cur-lo.src) / float64(hi.src-lo.src)
			return lo.rendered + int(frac*float64(hi.rendered-lo.rendered))
		}
	}
	return a[len(a)-1].rendered
}

// syncPreviewScroll scrolls the preview so the rendered content at the editor's
// cursor stays in view. It only scrolls when the mapped position drifts out of
// the viewport (with a margin), so it follows like an editor rather than
// jumping every keystroke.
func (m *editorModel) syncPreviewScroll() {
	total := m.preview.TotalLineCount()
	h := m.preview.Height
	if h <= 0 || total <= h {
		m.preview.SetYOffset(0)
		return
	}

	target := m.cursorRenderedLine(total)

	off := m.preview.YOffset
	margin := h / 5
	switch {
	case target < off+margin:
		off = target - margin
	case target > off+h-1-margin:
		off = target - (h - 1) + margin
	}
	if off < 0 {
		off = 0
	}
	if maxOff := total - h; off > maxOff {
		off = maxOff
	}
	m.preview.SetYOffset(off)
}

// scheduleRender bumps the generation and fires a debounce tick carrying it.
func (m *editorModel) scheduleRender() tea.Cmd {
	m.renderGen++
	gen := m.renderGen
	return tea.Tick(editorRenderDebounce, func(time.Time) tea.Msg {
		return editorRenderTickMsg{gen: gen}
	})
}

// renderPreview returns a command that renders the current text and reports it
// back tagged with gen so stale renders can be discarded.
func (m editorModel) renderPreview(gen int) tea.Cmd {
	text := m.textarea.Value()
	width := m.preview.Width
	cfg := m.common.cfg
	note := m.note
	return func() tea.Msg {
		body := string(utils.RemoveFrontmatter([]byte(text)))
		out, err := renderMarkdown(cfg, note, width, body)
		if err != nil {
			return editorPreviewRenderedMsg{gen: gen, content: "render error: " + err.Error()}
		}
		return editorPreviewRenderedMsg{gen: gen, content: out}
	}
}

func (m editorModel) savedCmd() tea.Cmd {
	body := m.textarea.Value()
	return func() tea.Msg { return editorSavedMsg{body: body} }
}

func quitEditorCmd() tea.Cmd {
	return func() tea.Msg { return editorQuitMsg{} }
}

func (m editorModel) update(msg tea.Msg) (editorModel, tea.Cmd) {
	var cmds []tea.Cmd

	if key, ok := msg.(tea.KeyMsg); ok {
		if m.subState == editorStateConfirmQuit {
			switch key.String() {
			case "y", "Y", keyEnter:
				return m, quitEditorCmd()
			case "n", "N", keyEsc:
				m.subState = editorStateEditing
			}
			return m, nil
		}

		switch key.String() {
		case "ctrl+s":
			if err := writeDocument(m.localPath, m.textarea.Value()); err != nil {
				m.err = err
				m.statusMsg = "Save failed"
				return m, nil
			}
			m.savedContent = m.textarea.Value()
			m.statusMsg = "Saved"
			return m, m.savedCmd()
		case keyEsc:
			if m.dirty() {
				m.subState = editorStateConfirmQuit
				return m, nil
			}
			return m, quitEditorCmd()
		}
	}

	// Delegate to the textarea (typing, newlines, cursor movement, etc.).
	prev := m.textarea.Value()
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	if m.textarea.Value() != prev {
		m.statusMsg = ""
		cmds = append(cmds, m.scheduleRender())
	}

	switch msg := msg.(type) {
	case editorRenderTickMsg:
		if msg.gen == m.renderGen && m.textarea.Value() != m.lastRendered {
			m.lastRendered = m.textarea.Value()
			cmds = append(cmds, m.renderPreview(msg.gen))
		}
	case editorPreviewRenderedMsg:
		if msg.gen == m.renderGen {
			m.preview.SetContent(msg.content)
			m.rebuildAnchors(msg.content)
		}
	}

	// Keep the preview following the cursor (after any cursor move or re-render).
	m.syncPreviewScroll()

	return m, tea.Batch(cmds...)
}

func (m editorModel) view() string {
	contentHeight := max(1, m.height-statusBarHeight)
	editorPane := m.textarea.View()

	var body string
	if m.preview.Width > 0 {
		sepLine := lipgloss.NewStyle().Foreground(statusBarNoteFg).Render("│")
		sep := strings.TrimRight(strings.Repeat(sepLine+"\n", contentHeight), "\n")
		previewPane := lipgloss.NewStyle().
			Width(m.preview.Width).
			Height(m.preview.Height).
			MaxHeight(m.preview.Height).
			Render(m.preview.View())
		body = lipgloss.JoinHorizontal(lipgloss.Top, editorPane, sep, previewPane)
	} else {
		body = editorPane
	}

	return body + "\n" + m.statusBar()
}

func (m editorModel) statusBar() string {
	w := m.width

	if m.subState == editorStateConfirmQuit {
		msg := " Discard unsaved changes?  (y) discard · (n) cancel "
		if pw := ansi.PrintableRuneWidth(msg); pw < w {
			msg += strings.Repeat(" ", w-pw)
		}
		return statusBarMessageStyle(msg)
	}

	name := m.note
	if name == "" {
		name = m.localPath
	}
	mark := ""
	if m.dirty() {
		mark = " •"
	}
	left := statusBarNoteStyle(fmt.Sprintf(" %s%s ", name, mark))

	var right string
	if m.statusMsg != "" {
		right = statusBarMessageStyle(fmt.Sprintf(" %s ", m.statusMsg)) + statusBarHelpStyle(" esc back ")
	} else {
		right = statusBarHelpStyle(" ctrl+s save · esc back ")
	}

	pad := max(0, w-ansi.PrintableRuneWidth(left)-ansi.PrintableRuneWidth(right))
	gap := statusBarNoteStyle(strings.Repeat(" ", pad))
	return left + gap + right
}

// writeDocument persists a document's raw body to disk.
func writeDocument(path, body string) error {
	if path == "" {
		return fmt.Errorf("no file path")
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("unable to write file: %w", err)
	}
	return nil
}
