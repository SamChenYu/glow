package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const styleSample = "# Heading\n\n**Bold** and *italic* text with `inline code`.\n"

// renderStyleSamples renders a small markdown preview for each available style
// so the settings form can show what each theme looks like.
func renderStyleSamples() map[string]string {
	allStyles := []string{
		styles.AutoStyle, styles.DarkStyle, styles.LightStyle,
		styles.PinkStyle, styles.DraculaStyle, styles.TokyoNightStyle,
		styles.AsciiStyle,
	}
	samples := make(map[string]string, len(allStyles))
	for _, s := range allStyles {
		r, err := glamour.NewTermRenderer(
			glamour.WithColorProfile(lipgloss.ColorProfile()),
			glamour.WithStylePath(s),
			glamour.WithWordWrap(60),
		)
		if err != nil {
			continue
		}
		out, err := r.Render(styleSample)
		if err != nil {
			continue
		}
		samples[s] = strings.TrimRight(out, "\n")
	}
	return samples
}

// settingsValues holds the values bound to the huh form fields. It lives on the
// heap (referenced via a pointer from settingsModel) so the huh Value(&x)
// bindings stay valid even as settingsModel is copied by value through the
// parent model's update loop.
type settingsValues struct {
	style     string
	widthStr  string
	scrollStr string
	mouse     bool
	pager     bool
	all       bool
	minimap   bool
}

// settingsModel is an in-app settings editor. It wraps a huh.Form (which
// implements tea.Model) and is driven by the parent model.
type settingsModel struct {
	common *commonModel
	form   *huh.Form
	vals   *settingsValues
}

// newSettingsModel builds the settings form from the current configuration.
//
// Initial values are read from the package-level config, which holds the raw,
// as-loaded snapshot: its style is un-resolved (so "auto" is preserved rather
// than shown as "dark"/"light"), and width comes from ConfiguredWidth (the
// value stored in the file) rather than the terminal-derived GlamourMaxWidth.
func newSettingsModel(common *commonModel) settingsModel {
	vals := &settingsValues{
		style:     config.GlamourStyle,
		widthStr:  strconv.Itoa(int(config.ConfiguredWidth)), //nolint:gosec
		scrollStr: strconv.Itoa(config.ScrollSpeed),
		mouse:     config.EnableMouse,
		pager:     config.Pager,
		all:       config.ShowAllFiles,
		minimap:   config.ShowMinimap,
	}

	validateInt := func(min int) func(string) error {
		return func(s string) error {
			n, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("must be a number")
			}
			if n < min {
				return fmt.Errorf("must be >= %d", min)
			}
			return nil
		}
	}

	km := huh.NewDefaultKeyMap()
	km.Select.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Select.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑", "back"))
	km.Confirm.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Confirm.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑", "back"))
	km.Input.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Input.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑", "back"))
	// huh's default Quit binding is ctrl+c only; add esc so the user can back
	// out of the settings view and return to the file listing.
	km.Quit = key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel"))

	samples := renderStyleSamples()

	styleSelect := huh.NewSelect[string]().
		Title("Style").
		Inline(true).
		Options(
			huh.NewOption("Auto", styles.AutoStyle),
			huh.NewOption("Dark", styles.DarkStyle),
			huh.NewOption("Light", styles.LightStyle),
			huh.NewOption("Pink", styles.PinkStyle),
			huh.NewOption("Dracula", styles.DraculaStyle),
			huh.NewOption("Tokyo Night", styles.TokyoNightStyle),
			huh.NewOption("ASCII", styles.AsciiStyle),
		).
		Value(&vals.style)

	styleSelect.DescriptionFunc(func() string {
		if s, ok := samples[vals.style]; ok {
			return s
		}
		return ""
	}, &vals.style)

	form := huh.NewForm(
		huh.NewGroup(
			styleSelect,

			huh.NewInput().
				Title("Width").
				Description("Word-wrap width (0 for terminal width)").
				Value(&vals.widthStr).
				Validate(validateInt(0)),

			huh.NewConfirm().
				Title("Minimap").
				Description("Show minimap sidebar (TUI mode)").
				Affirmative("On").
				Negative("Off").
				Value(&vals.minimap),

			huh.NewConfirm().
				Title("Mouse Support").
				Description("Enable mouse wheel scrolling (TUI mode)").
				Affirmative("On").
				Negative("Off").
				Value(&vals.mouse),

			huh.NewInput().
				Title("Scroll Speed").
				Description("Lines to scroll per step (TUI mode)").
				Value(&vals.scrollStr).
				Validate(validateInt(1)),

			huh.NewConfirm().
				Title("Pager").
				Description("Use pager to display markdown").
				Affirmative("On").
				Negative("Off").
				Value(&vals.pager),

			huh.NewConfirm().
				Title("Show All Files").
				Description("Show hidden and ignored files").
				Affirmative("On").
				Negative("Off").
				Value(&vals.all),
		).Title("Settings"),
	).WithTheme(huh.ThemeCharm()).WithKeyMap(km)

	m := settingsModel{
		common: common,
		form:   form,
		vals:   vals,
	}
	m.setSize(common.width, common.height)
	return m
}

// setSize sizes the embedded form. huh honors an explicitly-set size (it only
// auto-sizes from WindowSizeMsg when width/height are zero), so we re-apply on
// every resize.
func (m *settingsModel) setSize(width, height int) {
	if width > 0 {
		m.form = m.form.WithWidth(width)
	}
	if height > 0 {
		m.form = m.form.WithHeight(height)
	}
}

func (m settingsModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m settingsModel) update(msg tea.Msg) (settingsModel, tea.Cmd) {
	f, cmd := m.form.Update(msg)
	if form, ok := f.(*huh.Form); ok {
		m.form = form
	}
	return m, cmd
}

func (m settingsModel) view() string {
	return m.form.View()
}

// completed reports whether the user submitted the settings form.
func (m settingsModel) completed() bool {
	return m.form.State == huh.StateCompleted
}

// aborted reports whether the user cancelled the settings form (e.g. via esc).
func (m settingsModel) aborted() bool {
	return m.form.State == huh.StateAborted
}

// writeSettings persists the given settings to the YAML config file at path.
// The template (keys, comments, order) mirrors the default config written by
// config_cmd.go so the two never drift.
func writeSettings(path, style string, mouse, pager bool, width int, all, minimap bool, scrollSpeed int) error {
	if path == "" {
		return fmt.Errorf("no config file path configured")
	}

	content := fmt.Sprintf(`# style name or JSON path (default "auto")
style: %q
# mouse support (TUI-mode only)
mouse: %v
# use pager to display markdown
pager: %v
# word-wrap at width
width: %d
# show all files, including hidden and ignored.
all: %v
# show minimap sidebar (TUI-mode only)
minimap: %v
# lines to scroll per step (TUI-mode only)
scrollSpeed: %d
`, style, mouse, pager, width, all, minimap, scrollSpeed)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("unable to write config file: %w", err)
	}

	return nil
}
