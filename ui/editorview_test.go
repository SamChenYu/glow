package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour/styles"
)

func ctrlS() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlS} }

// setupEditorFromDocument builds a model showing a document backed by a temp
// file, ready to open the split editor with `e`.
func setupEditorFromDocument(t *testing.T, content string) (model, string) {
	t.Helper()
	initSections()

	path := filepath.Join(t.TempDir(), "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	config = Config{GlamourEnabled: true, GlamourStyle: styles.DarkStyle, GlamourMaxWidth: 80}
	common := &commonModel{cfg: config, width: 120, height: 40}

	m := model{
		common: common,
		state:  stateShowDocument,
		stash:  newStashModel(common),
		pager: pagerModel{
			common:          common,
			currentDocument: markdown{localPath: path, Body: content, Note: "test.md"},
		},
	}
	return m, path
}

func TestEditorOpensFromDocument(t *testing.T) {
	m, path := setupEditorFromDocument(t, "# Hi\n\noriginal\n")

	m = send(m, keyMsg("e"))
	if m.state != stateShowEditor {
		t.Fatalf("expected editor state after e, got %v", m.state)
	}
	if m.editor.localPath != path {
		t.Errorf("editor localPath = %q, want %q", m.editor.localPath, path)
	}
	if m.editor.origin != stateShowDocument {
		t.Errorf("editor origin = %v, want document", m.editor.origin)
	}
	if m.editor.textarea.Value() != "# Hi\n\noriginal\n" {
		t.Errorf("editor did not load the document body")
	}
}

func TestEditorTypeAndSave(t *testing.T) {
	m, path := setupEditorFromDocument(t, "old\n")
	m = send(m, keyMsg("e"))

	newContent := "# Changed\n\nbrand new content\n"
	m.editor.textarea.SetValue(newContent)
	if !m.editor.dirty() {
		t.Fatal("editor should be dirty after edit")
	}

	m = send(m, ctrlS())

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != newContent {
		t.Errorf("file on disk = %q, want %q", string(data), newContent)
	}
	if m.editor.dirty() {
		t.Error("editor should be clean after save")
	}
	if m.editor.statusMsg != "Saved" {
		t.Errorf("status = %q, want Saved", m.editor.statusMsg)
	}
}

func TestEditorTypedKeystrokesReachTextarea(t *testing.T) {
	// Regression: the textarea must be focused so real key input is applied
	// (not just SetValue). Types characters via KeyMsgs through the model.
	m, path := setupEditorFromDocument(t, "")
	m = send(m, keyMsg("e"))
	if !m.editor.textarea.Focused() {
		t.Fatal("editor textarea should be focused on open")
	}

	for _, r := range "hello" {
		m = send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if got := m.editor.textarea.Value(); got != "hello" {
		t.Fatalf("typed keystrokes not applied: textarea value = %q", got)
	}

	m = send(m, ctrlS())
	data, _ := os.ReadFile(path)
	if string(data) != "hello" {
		t.Errorf("saved file = %q, want %q", string(data), "hello")
	}
}

func TestEditorPreviewRenders(t *testing.T) {
	m, _ := setupEditorFromDocument(t, "# Title\n\nUniquewordxyz here\n")
	m = send(m, keyMsg("e"))

	// Drive the render pipeline synchronously (tea.Tick doesn't fire in-process).
	gen := m.editor.renderGen
	renderMsg := m.editor.renderPreview(gen)()
	m = send(m, renderMsg)

	out := m.editor.preview.View()
	if !strings.Contains(out, "Uniquewordxyz") {
		t.Errorf("preview did not render document text; got:\n%s", out)
	}
}

func TestEditorPreviewFollowsCursor(t *testing.T) {
	// A tall document (blank-line-separated paragraphs so glamour doesn't
	// reflow them into one) whose rendered output exceeds the viewport height.
	var b strings.Builder
	for i := 0; i < 150; i++ {
		fmt.Fprintf(&b, "Paragraph %d text\n\n", i)
	}
	m, _ := setupEditorFromDocument(t, b.String())
	m = send(m, keyMsg("e"))

	// Render the preview so it has content.
	m = send(m, m.editor.renderPreview(m.editor.renderGen)())

	if m.editor.preview.TotalLineCount() <= m.editor.preview.Height {
		t.Skipf("preview not taller than viewport (%d <= %d); can't test scrolling",
			m.editor.preview.TotalLineCount(), m.editor.preview.Height)
	}
	if m.editor.preview.YOffset != 0 {
		t.Fatalf("preview should start at the top, YOffset=%d", m.editor.preview.YOffset)
	}

	// Move the cursor to the bottom; the preview should scroll to follow.
	for i := 0; i < 400; i++ {
		m = send(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.editor.preview.YOffset <= 0 {
		t.Errorf("preview should scroll down to follow the cursor, YOffset=%d", m.editor.preview.YOffset)
	}

	// Move back to the top; the preview should return to the top.
	for i := 0; i < 400; i++ {
		m = send(m, tea.KeyMsg{Type: tea.KeyUp})
	}
	if m.editor.preview.YOffset != 0 {
		t.Errorf("preview should return to the top with the cursor, YOffset=%d", m.editor.preview.YOffset)
	}
}

func TestEditorPreviewAnchorsOnHeadings(t *testing.T) {
	// Long paragraphs render to many more lines than they occupy in source, so
	// a proportional mapping would place the cursor-at-heading target far from
	// the heading's real rendered line. Heading anchoring must land on it.
	long := strings.Repeat("word ", 100)
	var lines []string
	for i := 0; i < 5; i++ {
		lines = append(lines, long, "")
	}
	headingSrc := len(lines)
	lines = append(lines, "## DEEPHEADINGZZ", "")
	for i := 0; i < 5; i++ {
		lines = append(lines, fmt.Sprintf("Tail paragraph %d.", i), "")
	}
	m, _ := setupEditorFromDocument(t, strings.Join(lines, "\n"))
	m = send(m, keyMsg("e"))
	m = send(m, m.editor.renderPreview(m.editor.renderGen)())

	if m.editor.preview.TotalLineCount() <= m.editor.preview.Height {
		t.Skip("preview not taller than viewport; can't test scrolling")
	}

	// Place the cursor on the heading's source line.
	for guard := 0; m.editor.textarea.Line() < headingSrc && guard < 5000; guard++ {
		m.editor.textarea.CursorDown()
	}
	if m.editor.textarea.Line() != headingSrc {
		t.Fatalf("cursor at line %d, want heading line %d", m.editor.textarea.Line(), headingSrc)
	}
	m.editor.syncPreviewScroll()

	if !strings.Contains(m.editor.preview.View(), "DEEPHEADINGZZ") {
		t.Errorf("preview should show the heading under the cursor; view:\n%s", m.editor.preview.View())
	}
}

func TestEditorEscConfirmAndCancel(t *testing.T) {
	m, _ := setupEditorFromDocument(t, "old\n")
	m = send(m, keyMsg("e"))
	m.editor.textarea.SetValue("edited, unsaved")

	// esc with unsaved edits -> confirm prompt, still in editor.
	m = send(m, keyMsg("esc"))
	if m.state != stateShowEditor {
		t.Fatalf("esc with unsaved edits should not leave the editor, state=%v", m.state)
	}
	if m.editor.subState != editorStateConfirmQuit {
		t.Fatalf("expected confirm-quit sub-state, got %v", m.editor.subState)
	}

	// esc again -> cancel, back to editing.
	m = send(m, keyMsg("esc"))
	if m.editor.subState != editorStateEditing {
		t.Errorf("second esc should cancel back to editing, got %v", m.editor.subState)
	}
}

func TestEditorQuitReturnsToOrigin(t *testing.T) {
	m, _ := setupEditorFromDocument(t, "old\n")
	m = send(m, keyMsg("e"))
	if m.state != stateShowEditor {
		t.Fatalf("expected editor state, got %v", m.state)
	}

	// The editor emits editorQuitMsg to leave; the parent returns to origin.
	m = send(m, editorQuitMsg{})
	if m.state != stateShowDocument {
		t.Errorf("quitting the editor should return to the document, got %v", m.state)
	}
}

func TestExternalEditorStillOnCapitalE(t *testing.T) {
	m, _ := setupEditorFromDocument(t, "old\n")

	next, cmd := m.Update(keyMsg("E"))
	m = next.(model)
	if m.state != stateShowDocument {
		t.Errorf("E should not change state, got %v", m.state)
	}
	if cmd == nil {
		t.Error("E should return an external-editor command")
	}
}

func TestEditorNotOpenedWhileFiltering(t *testing.T) {
	initSections()
	config = Config{GlamourEnabled: true}
	common := &commonModel{cfg: config, width: 120, height: 40}
	m := model{common: common, state: stateShowStash, stash: newStashModel(common)}
	m.stash.filterState = filtering

	m = send(m, keyMsg("e"))
	if m.state == stateShowEditor {
		t.Fatal("e must not open the editor while filtering")
	}
}

func TestWriteDocumentRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc.md")
	body := "---\ntitle: x\n---\n# Heading\n\nbody\n"
	if err := writeDocument(path, body); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != body {
		t.Errorf("round-trip mismatch:\n got %q\nwant %q", string(data), body)
	}
}

func TestWriteDocumentEmptyPath(t *testing.T) {
	if err := writeDocument("", "x"); err == nil {
		t.Error("expected error for empty path")
	}
}
