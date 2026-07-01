package ui

// Config contains TUI-specific configuration.
type Config struct {
	ShowAllFiles     bool
	ShowLineNumbers  bool
	Gopath           string `env:"GOPATH"`
	HomeDir          string `env:"HOME"`
	GlamourMaxWidth  uint
	GlamourStyle     string `env:"GLAMOUR_STYLE"`
	EnableMouse      bool
	PreserveNewLines bool

	// Working directory or file path
	Path string

	ShowMinimap bool

	ScrollSpeed int

	// Pager mirrors the CLI "pager" config so the in-app settings editor can
	// show and persist its current value. Not otherwise used by the TUI.
	Pager bool

	// ConfiguredWidth is the width value as stored in the config file. It is
	// kept separate from GlamourMaxWidth (which is overridden with the terminal
	// width at runtime) so the settings editor shows and persists the
	// configured value rather than the current terminal width.
	ConfiguredWidth uint

	// ConfigPath is the absolute path of the YAML config file the in-app
	// settings editor writes to. Empty disables in-app saving.
	ConfigPath string

	// For debugging the UI
	HighPerformancePager bool `env:"GLOW_HIGH_PERFORMANCE_PAGER" envDefault:"true"`
	GlamourEnabled       bool `env:"GLOW_ENABLE_GLAMOUR"         envDefault:"true"`
}
