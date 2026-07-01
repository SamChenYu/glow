// Package ui provides the main UI for the glow application.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/log"
	"github.com/muesli/gitcha"
	te "github.com/muesli/termenv"
)

const (
	statusMessageTimeout = time.Second * 3 // how long to show status messages like "stashed!"
	ellipsis             = "…"
)

var (
	config Config

	markdownExtensions = []string{
		"*.md", "*.mdown", "*.mkdn", "*.mkd", "*.markdown",
	}
)

// NewProgram returns a new Tea program.
func NewProgram(cfg Config, content string) *tea.Program {
	log.Debug(
		"Starting glow",
		"high_perf_pager",
		cfg.HighPerformancePager,
		"glamour",
		cfg.GlamourEnabled,
	)

	config = cfg
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if cfg.EnableMouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	m := newModel(cfg, content)
	return tea.NewProgram(m, opts...)
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type (
	initLocalFileSearchMsg struct {
		cwd string
		ch  chan gitcha.SearchResult
	}
)

type (
	foundLocalFileMsg       gitcha.SearchResult
	localFileSearchFinished struct{}
	statusMessageTimeoutMsg applicationContext
)

// applicationContext indicates the area of the application something applies
// to. Occasionally used as an argument to commands and messages.
type applicationContext int

const (
	stashContext applicationContext = iota
	pagerContext
)

// state is the top-level application state.
type state int

const (
	stateShowStash state = iota
	stateShowDocument
	stateShowSettings
)

func (s state) String() string {
	return map[state]string{
		stateShowStash:    "showing file listing",
		stateShowDocument: "showing document",
		stateShowSettings: "showing settings",
	}[s]
}

// Common stuff we'll need to access in all models.
type commonModel struct {
	cfg    Config
	cwd    string
	width  int
	height int
}

type model struct {
	common   *commonModel
	state    state
	fatalErr error

	// Sub-models
	stash    stashModel
	pager    pagerModel
	settings settingsModel

	// Channel that receives paths to local markdown files
	// (via the github.com/muesli/gitcha package)
	localFileFinder chan gitcha.SearchResult
}

// unloadDocument unloads a document from the pager. Note that while this
// method alters the model we also need to send along any commands returned.
func (m *model) unloadDocument() []tea.Cmd {
	m.state = stateShowStash
	m.stash.viewState = stashStateReady
	m.pager.unload()
	m.pager.showHelp = false

	var batch []tea.Cmd
	if m.pager.viewport.HighPerformanceRendering {
		batch = append(batch, tea.ClearScrollArea) //nolint:staticcheck
	}
	if !m.stash.shouldSpin() {
		batch = append(batch, m.stash.spinner.Tick)
	}
	return batch
}

// resolveGlamourStyle turns the "auto" style into a concrete dark or light
// style based on the terminal background. Any other style is returned as-is.
func resolveGlamourStyle(s string) string {
	if s == styles.AutoStyle {
		if te.HasDarkBackground() {
			return styles.DarkStyle
		}
		return styles.LightStyle
	}
	return s
}

// applySettings persists the values from the settings form and applies them to
// the running session where feasible. It returns any commands that need to run
// (a status message, plus a live mouse-mode toggle if that changed).
func (m *model) applySettings() tea.Cmd {
	v := m.settings.vals
	width, _ := strconv.Atoi(v.widthStr)
	scroll, _ := strconv.Atoi(v.scrollStr)
	if scroll < 1 {
		scroll = 1
	}

	if err := writeSettings(m.common.cfg.ConfigPath, v.style, v.mouse, v.pager, width, v.all, v.minimap, scroll); err != nil {
		log.Error("could not save settings", "error", err)
		return m.stash.setStatusMessage(statusMessage{errorStatusMessage, "Couldn't save settings"})
	}

	mouseChanged := config.EnableMouse != v.mouse

	// Update the raw, as-configured snapshot so a subsequent open of the
	// settings form (and code that reads the package-level config) sees the
	// new values.
	config.GlamourStyle = v.style
	config.ConfiguredWidth = uint(width) //nolint:gosec
	config.ScrollSpeed = scroll
	config.EnableMouse = v.mouse
	config.Pager = v.pager
	config.ShowAllFiles = v.all
	config.ShowMinimap = v.minimap

	// Update the live config. Style is resolved so the next opened document
	// renders with the new theme. Width is intentionally not applied live: at
	// runtime common.cfg.GlamourMaxWidth tracks the terminal width.
	m.common.cfg.GlamourStyle = resolveGlamourStyle(v.style)
	m.common.cfg.ScrollSpeed = scroll
	m.common.cfg.EnableMouse = v.mouse
	m.common.cfg.Pager = v.pager
	m.common.cfg.ShowAllFiles = v.all
	m.common.cfg.ShowMinimap = v.minimap

	cmds := []tea.Cmd{m.stash.setStatusMessage(statusMessage{normalStatusMessage, "Settings saved"})}

	// Mouse mode is set once as a program option at startup, so toggle it live
	// via a command when it changed.
	if mouseChanged {
		if v.mouse {
			cmds = append(cmds, tea.EnableMouseCellMotion)
		} else {
			cmds = append(cmds, tea.DisableMouse)
		}
	}

	return tea.Batch(cmds...)
}

func newModel(cfg Config, content string) tea.Model {
	initSections()

	cfg.GlamourStyle = resolveGlamourStyle(cfg.GlamourStyle)

	common := commonModel{
		cfg: cfg,
	}

	m := model{
		common: &common,
		state:  stateShowStash,
		pager:  newPagerModel(&common),
		stash:  newStashModel(&common),
	}

	path := cfg.Path
	if path == "" && content != "" {
		m.state = stateShowDocument
		m.pager.currentDocument = markdown{Body: content}
		return m
	}

	if path == "" {
		path = "."
	}
	info, err := os.Stat(path)
	if err != nil {
		log.Error("unable to stat file", "file", path, "error", err)
		m.fatalErr = err
		return m
	}
	if info.IsDir() {
		m.state = stateShowStash
	} else {
		cwd, _ := os.Getwd()
		m.state = stateShowDocument
		m.pager.currentDocument = markdown{
			localPath: path,
			Note:      stripAbsolutePath(path, cwd),
			Modtime:   info.ModTime(),
		}
	}

	return m
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.stash.spinner.Tick}

	switch m.state {
	case stateShowStash:
		cmds = append(cmds, findLocalFiles(*m.common))
	case stateShowDocument:
		cmds = append(cmds, loadLocalMarkdown(&m.pager.currentDocument))
	}

	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If there's been an error, any key exits
	if m.fatalErr != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.state == stateShowDocument && (m.pager.state == pagerStateSearching || m.pager.state == pagerStateTOC) {
				var cmd tea.Cmd
				m.pager, cmd = m.pager.update(msg)
				return m, cmd
			}
			if m.state == stateShowDocument || m.stash.viewState == stashStateLoadingDocument {
				batch := m.unloadDocument()
				return m, tea.Batch(batch...)
			}
		case "r":
			var cmd tea.Cmd
			if m.state == stateShowStash {
				// pass through all keys if we're editing the filter
				if m.stash.filterState == filtering {
					m.stash, cmd = m.stash.update(msg)
					return m, cmd
				}
				m.stash.markdowns = nil
				return m, m.Init()
			}

		case "s":
			// Open the settings editor from the file listing only. The
			// filterState guard keeps "s" a literal character while the user
			// is typing in the filter input.
			if m.state == stateShowStash &&
				m.stash.filterState != filtering &&
				m.stash.viewState == stashStateReady {
				m.settings = newSettingsModel(m.common)
				m.state = stateShowSettings
				return m, m.settings.Init()
			}

		case "q":
			if m.state == stateShowDocument && m.pager.state == pagerStateTOC {
				var cmd tea.Cmd
				m.pager, cmd = m.pager.update(msg)
				return m, cmd
			}

			var cmd tea.Cmd

			switch m.state { //nolint:exhaustive
			case stateShowStash:
				// pass through all keys if we're editing the filter
				if m.stash.filterState == filtering {
					m.stash, cmd = m.stash.update(msg)
					return m, cmd
				}
			case stateShowSettings:
				// Let the settings form handle "q"; don't quit the app while
				// the settings editor is open.
				m.settings, cmd = m.settings.update(msg)
				return m, cmd
			}

			return m, tea.Quit

		case "left", "h", "delete":
			if m.state == stateShowDocument && m.pager.state != pagerStateTOC {
				cmds = append(cmds, m.unloadDocument()...)
				return m, tea.Batch(cmds...)
			}

		case "ctrl+z":
			return m, tea.Suspend

		// Ctrl+C always quits no matter where in the application you are.
		case "ctrl+c":
			return m, tea.Quit
		}

	// Window size is received when starting up and on every resize
	case tea.WindowSizeMsg:
		m.common.width = msg.Width
		m.common.height = msg.Height
		m.common.cfg.GlamourMaxWidth = uint(msg.Width) //nolint:gosec
		m.stash.setSize(msg.Width, msg.Height)
		m.pager.setSize(msg.Width, msg.Height)
		if m.state == stateShowSettings {
			m.settings.setSize(msg.Width, msg.Height)
		}

	case initLocalFileSearchMsg:
		m.localFileFinder = msg.ch
		m.common.cwd = msg.cwd
		cmds = append(cmds, findNextLocalFile(m))

	case fetchedMarkdownMsg:
		// We've loaded a markdown file's contents for rendering
		m.pager.currentDocument = *msg
		body := string(utils.RemoveFrontmatter([]byte(msg.Body)))
		cmds = append(cmds, renderWithGlamour(m.pager, body))

	case contentRenderedMsg:
		m.state = stateShowDocument

	case localFileSearchFinished:
		// Always pass these messages to the stash so we can keep it updated
		// about network activity, even if the user isn't currently viewing
		// the stash.
		stashModel, cmd := m.stash.update(msg)
		m.stash = stashModel
		return m, cmd

	case foundLocalFileMsg:
		newMd := localFileToMarkdown(m.common.cwd, gitcha.SearchResult(msg))
		m.stash.addMarkdowns(newMd)
		if m.stash.filterApplied() {
			newMd.buildFilterValue()
		}
		if m.stash.shouldUpdateFilter() {
			cmds = append(cmds, filterMarkdowns(m.stash))
		}
		cmds = append(cmds, findNextLocalFile(m))

	case filteredMarkdownMsg:
		if m.state == stateShowDocument {
			newStashModel, cmd := m.stash.update(msg)
			m.stash = newStashModel
			cmds = append(cmds, cmd)
		}
	}

	// Process children
	switch m.state {
	case stateShowStash:
		newStashModel, cmd := m.stash.update(msg)
		m.stash = newStashModel
		cmds = append(cmds, cmd)

	case stateShowDocument:
		newPagerModel, cmd := m.pager.update(msg)
		m.pager = newPagerModel
		cmds = append(cmds, cmd)

	case stateShowSettings:
		var cmd tea.Cmd
		m.settings, cmd = m.settings.update(msg)
		cmds = append(cmds, cmd)

		// The form finished this update: return to the file listing in the
		// same pass, because huh renders an empty view once it's quitting.
		switch {
		case m.settings.completed():
			m.state = stateShowStash
			cmds = append(cmds, m.applySettings())
		case m.settings.aborted():
			m.state = stateShowStash
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.fatalErr != nil {
		return errorView(m.fatalErr, true)
	}

	switch m.state { //nolint:exhaustive
	case stateShowDocument:
		return m.pager.View()
	case stateShowSettings:
		return m.settings.view()
	default:
		return m.stash.view()
	}
}

func errorView(err error, fatal bool) string {
	exitMsg := "press any key to "
	if fatal {
		exitMsg += "exit"
	} else {
		exitMsg += "return"
	}
	s := fmt.Sprintf("%s\n\n%v\n\n%s",
		errorTitleStyle.Render("ERROR"),
		err,
		subtleStyle.Render(exitMsg),
	)
	return "\n" + indent(s, 3)
}

// COMMANDS

func findLocalFiles(m commonModel) tea.Cmd {
	return func() tea.Msg {
		log.Info("findLocalFiles")
		var (
			cwd = m.cfg.Path
			err error
		)

		if cwd == "" {
			cwd, err = os.Getwd()
		} else {
			var info os.FileInfo
			info, err = os.Stat(cwd)
			if err == nil && info.IsDir() {
				cwd, err = filepath.Abs(cwd)
			}
		}

		// Note that this is one error check for both cases above
		if err != nil {
			log.Error("error finding local files", "error", err)
			return errMsg{err}
		}

		log.Debug("local directory is", "cwd", cwd)

		// Switch between FindFiles and FindAllFiles to bypass .gitignore rules
		var ch chan gitcha.SearchResult
		if m.cfg.ShowAllFiles {
			ch, err = gitcha.FindAllFilesExcept(cwd, markdownExtensions, nil)
		} else {
			ch, err = gitcha.FindFilesExcept(cwd, markdownExtensions, ignorePatterns(m))
		}

		if err != nil {
			log.Error("error finding local files", "error", err)
			return errMsg{err}
		}

		return initLocalFileSearchMsg{ch: ch, cwd: cwd}
	}
}

func findNextLocalFile(m model) tea.Cmd {
	return func() tea.Msg {
		res, ok := <-m.localFileFinder

		if ok {
			// Okay now find the next one
			return foundLocalFileMsg(res)
		}
		// We're done
		log.Debug("local file search finished")
		return localFileSearchFinished{}
	}
}

func waitForStatusMessageTimeout(appCtx applicationContext, t *time.Timer) tea.Cmd {
	return func() tea.Msg {
		<-t.C
		return statusMessageTimeoutMsg(appCtx)
	}
}

// ETC

// Convert a Gitcha result to an internal representation of a markdown
// document. Note that we could be doing things like checking if the file is
// a directory, but we trust that gitcha has already done that.
func localFileToMarkdown(cwd string, res gitcha.SearchResult) *markdown {
	return &markdown{
		localPath: res.Path,
		Note:      stripAbsolutePath(res.Path, cwd),
		Modtime:   res.Info.ModTime(),
	}
}

func stripAbsolutePath(fullPath, cwd string) string {
	fp, _ := filepath.EvalSymlinks(fullPath)
	cp, _ := filepath.EvalSymlinks(cwd)
	return strings.ReplaceAll(fp, cp+string(os.PathSeparator), "")
}

// Lightweight version of reflow's indent function.
func indent(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	l := strings.Split(s, "\n")
	b := strings.Builder{}
	i := strings.Repeat(" ", n)
	for _, v := range l {
		fmt.Fprintf(&b, "%s%s\n", i, v)
	}
	return b.String()
}
