package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shai/shai/internal/app"
)

// Screen identifies the active TUI view.
type Screen int

const (
	ScreenChat Screen = iota
	ScreenHelp
	ScreenModelPicker
	ScreenDownload
	ScreenSettings
)

// ChatMessage is one conversation line.
type ChatMessage struct {
	Role    string
	Content string
}

// Model is the root Bubble Tea model.
type Model struct {
	screen    Screen
	cfg       app.Config
	runOpts   app.RunOptions
	modelName string
	modelPath string

	messages []ChatMessage
	viewport viewport.Model
	input    textarea.Model

	width  int
	height int

	statusMsg string
	errMsg    string
	busy      bool
	cancel    context.CancelFunc
	genCh     chan tea.Msg

	// Model picker
	modelList list.Model

	// Download screen
	dlRepo  string
	dlName  string
	dlField int // 0=repo, 1=name

	// Settings
	settingsField int
	settingsEdit  app.Config

	genResult string
}

// NewProgram creates the TUI program.
func NewProgram(cfg app.Config, opts app.RunOptions, modelName string) *tea.Program {
	m := initialModel(cfg, opts, modelName)
	return tea.NewProgram(&m, tea.WithAltScreen())
}

func initialModel(cfg app.Config, opts app.RunOptions, modelName string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message… (/help for commands)"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)
	vp.SetContent("")

	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Models"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return Model{
		screen:        ScreenChat,
		cfg:           cfg,
		runOpts:       opts,
		modelName:     modelName,
		input:         ta,
		viewport:      vp,
		modelList:     l,
		settingsEdit:  cfg,
		settingsField: 0,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.refreshModels())
}

type modelsLoadedMsg struct {
	items []list.Item
	names []string
	err   error
}

func (m *Model) refreshModels() tea.Cmd {
	return func() tea.Msg {
		names, err := app.ListModels()
		if err != nil {
			return modelsLoadedMsg{err: err}
		}
		var items []list.Item
		for _, n := range names {
			items = append(items, modelItem{name: n})
		}
		return modelsLoadedMsg{items: items, names: names}
	}
}

type modelItem struct {
	name string
}

func (i modelItem) Title() string       { return i.name }
func (i modelItem) Description() string { return "" }
func (i modelItem) FilterValue() string { return i.name }

type genDoneMsg struct {
	text string
	err  error
}

type genTokenMsg struct {
	text string
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case modelsLoadedMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.modelList.SetItems(msg.items)
		if m.modelName == "" && len(msg.names) > 0 {
			m.modelName = msg.names[0]
		}
		return m, nil

	case genTokenMsg:
		m.appendAssistantToken(msg.text)
		if m.genCh != nil {
			return m, waitGen(m.genCh)
		}
		return m, nil

	case genDoneMsg:
		m.genCh = nil
		m.busy = false
		m.cancel = nil
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.statusMsg = "Error"
		} else {
			m.statusMsg = "Ready"
			if msg.text != "" {
				m.finalizeAssistant(msg.text)
			}
		}
		m.refreshViewport()
		return m, nil

	case downloadDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.statusMsg = fmt.Sprintf("Downloaded to %s", msg.path)
			m.screen = ScreenChat
		}
		return m, m.refreshModels()
	}

	var cmd tea.Cmd
	switch m.screen {
	case ScreenChat:
		if !m.busy {
			m.input, cmd = m.input.Update(msg)
		}
	case ScreenModelPicker:
		m.modelList, cmd = m.modelList.Update(msg)
	case ScreenDownload:
		// handled in keys
	case ScreenSettings:
		// handled in keys
	case ScreenHelp:
		// keys only
	}
	return m, cmd
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.screen {
	case ScreenHelp:
		return m.viewHelp()
	case ScreenModelPicker:
		return m.viewModelPicker()
	case ScreenDownload:
		return m.viewDownload()
	case ScreenSettings:
		return m.viewSettings()
	default:
		return m.viewChat()
	}
}

func (m *Model) layout() {
	headerH := 2
	statusH := 1
	inputH := 5
	vpH := m.height - headerH - statusH - inputH - 2
	if vpH < 4 {
		vpH = 4
	}
	m.viewport.Width = m.width - 2
	m.viewport.Height = vpH
	m.input.SetWidth(m.width - 4)

	m.modelList.SetSize(m.width-4, m.height-6)
}

func (m *Model) refreshViewport() {
	var b strings.Builder
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			b.WriteString(styleUser.Render("You: "))
			b.WriteString(msg.Content)
		case "assistant":
			b.WriteString(styleAssistant.Render("Assistant: "))
			b.WriteString(msg.Content)
		default:
			b.WriteString(msg.Content)
		}
		b.WriteByte('\n')
	}
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}
