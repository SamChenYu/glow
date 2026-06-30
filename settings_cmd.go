package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/huh"
	"github.com/spf13/viper"
)

var settingsFlag bool

func runSettingsForm() error {
	if err := ensureConfigFile(); err != nil {
		return err
	}

	currentStyle := viper.GetString("style")
	currentMouse := viper.GetBool("mouse")
	currentPager := viper.GetBool("pager")
	widthStr := strconv.Itoa(viper.GetInt("width"))
	currentAll := viper.GetBool("all")
	currentMinimap := viper.GetBool("minimap")
	scrollStr := strconv.Itoa(viper.GetInt("scrollSpeed"))

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

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Style").
				Description("Auto · Dark · Light · Pink · Dracula · Tokyo Night · ASCII").
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
				Value(&currentStyle),

			huh.NewInput().
				Title("Width").
				Description("Word-wrap width (0 for terminal width)").
				Value(&widthStr).
				Validate(validateInt(0)),

			huh.NewConfirm().
				Title("Minimap").
				Description("Show minimap sidebar (TUI mode)").
				Affirmative("On").
				Negative("Off").
				Value(&currentMinimap),

			huh.NewConfirm().
				Title("Mouse Support").
				Description("Enable mouse wheel scrolling (TUI mode)").
				Affirmative("On").
				Negative("Off").
				Value(&currentMouse),

			huh.NewInput().
				Title("Scroll Speed").
				Description("Lines to scroll per step (TUI mode)").
				Value(&scrollStr).
				Validate(validateInt(1)),

			huh.NewConfirm().
				Title("Pager").
				Description("Use pager to display markdown").
				Affirmative("On").
				Negative("Off").
				Value(&currentPager),

			huh.NewConfirm().
				Title("Show All Files").
				Description("Show hidden and ignored files").
				Affirmative("On").
				Negative("Off").
				Value(&currentAll),
		).Title("Settings"),
	).WithTheme(huh.ThemeCharm()).WithKeyMap(km)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Settings unchanged.")
			return nil
		}
		return fmt.Errorf("settings form error: %w", err)
	}

	w, _ := strconv.Atoi(widthStr)
	ss, _ := strconv.Atoi(scrollStr)

	return writeSettings(currentStyle, currentMouse, currentPager, w, currentAll, currentMinimap, ss)
}

func writeSettings(style string, mouse, pager bool, width int, all, minimap bool, scrollSpeed int) error {
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

	if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("unable to write config file: %w", err)
	}

	fmt.Println("Settings saved to:", configFile)
	return nil
}
