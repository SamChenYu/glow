package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/huh"
)

// setupSettingsModel builds a model in the stash state with a known config
// snapshot and a temp config path, ready to receive key messages.
func setupSettingsModel(t *testing.T) (model, string) {
	t.Helper()
	initSections()

	cfgPath := filepath.Join(t.TempDir(), "glow.yml")
	config = Config{
		GlamourStyle:    styles.AutoStyle,
		GlamourMaxWidth: 120,
		ConfiguredWidth: 80,
		ScrollSpeed:     7,
		ShowMinimap:     true,
		EnableMouse:     false,
		Pager:           false,
		ShowAllFiles:    false,
		ConfigPath:      cfgPath,
	}

	common := &commonModel{cfg: config, width: 120, height: 40}
	m := model{
		common: common,
		state:  stateShowStash,
		stash:  newStashModel(common),
	}
	return m, cfgPath
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func send(m model, msg tea.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func TestSettingsOpenAndSave(t *testing.T) {
	m, cfgPath := setupSettingsModel(t)

	m = send(m, keyMsg("s"))
	if m.state != stateShowSettings {
		t.Fatalf("expected state %v after pressing s, got %v", stateShowSettings, m.state)
	}
	if m.settings.form == nil {
		t.Fatal("settings form was not constructed")
	}

	// Verify the form was initialised from the current config snapshot.
	if m.settings.vals.style != styles.AutoStyle || m.settings.vals.scrollStr != "7" || !m.settings.vals.minimap {
		t.Fatalf("form not seeded from config: %+v", *m.settings.vals)
	}

	// Simulate the user editing a couple of values and submitting. huh's own
	// field navigation is exercised by the library; here we drive the
	// completion -> applySettings -> return-to-stash wiring that this change
	// adds, using huh's exported State field.
	m.settings.vals.minimap = false
	m.settings.vals.scrollStr = "9"
	m.settings.form.State = huh.StateCompleted

	// Any message now routes through the settings state and triggers the
	// completion handling.
	m = send(m, keyMsg("enter"))
	if m.state != stateShowStash {
		t.Fatalf("after completion expected stash state, got %v", m.state)
	}

	// Live config should reflect the edits.
	if m.common.cfg.ShowMinimap {
		t.Error("expected ShowMinimap to be applied as false")
	}
	if m.common.cfg.ScrollSpeed != 9 {
		t.Errorf("expected ScrollSpeed 9 applied live, got %d", m.common.cfg.ScrollSpeed)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("config file was not written: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"# style name or JSON path",
		`style: "auto"`,
		"minimap: false",
		"scrollSpeed: 9",
		"width: 80",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("written config missing %q\n---\n%s", want, got)
		}
	}
}

func TestSettingsCancelDoesNotWrite(t *testing.T) {
	m, cfgPath := setupSettingsModel(t)

	m = send(m, keyMsg("s"))
	if m.state != stateShowSettings {
		t.Fatalf("expected settings state, got %v", m.state)
	}

	m = send(m, keyMsg("esc"))
	if m.state != stateShowStash {
		t.Fatalf("esc should return to stash, got %v", m.state)
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("config file should not exist after cancel, stat err = %v", err)
	}
}

func TestSettingsNotOpenedWhileFiltering(t *testing.T) {
	m, _ := setupSettingsModel(t)
	m.stash.filterState = filtering

	m = send(m, keyMsg("s"))
	if m.state == stateShowSettings {
		t.Fatal("settings must not open while the filter input is active")
	}
}

func TestSettingsQDoesNotQuit(t *testing.T) {
	m, _ := setupSettingsModel(t)
	m = send(m, keyMsg("s"))
	if m.state != stateShowSettings {
		t.Fatalf("expected settings state, got %v", m.state)
	}

	next, cmd := m.Update(keyMsg("q"))
	m = next.(model)
	if m.state != stateShowSettings {
		t.Fatalf("q should keep the settings editor open, got %v", m.state)
	}
	if cmd != nil {
		if _, isQuit := cmd().(tea.QuitMsg); isQuit {
			t.Error("pressing q in the settings editor quit the whole app")
		}
	}
}

func TestWriteSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "glow.yml")
	if err := writeSettings(path, styles.DraculaStyle, true, false, 100, true, false, 3); err != nil {
		t.Fatalf("writeSettings: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		`style: "dracula"`,
		"mouse: true",
		"pager: false",
		"width: 100",
		"all: true",
		"minimap: false",
		"scrollSpeed: 3",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteSettingsEmptyPath(t *testing.T) {
	if err := writeSettings("", "auto", false, false, 0, false, true, 5); err == nil {
		t.Error("expected error when path is empty")
	}
}
