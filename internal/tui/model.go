package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabrielbotandev/laisa/internal/app"
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

// genUsage holds token/context stats from the backend.
type genUsage struct {
	PromptTokens     int
	CompletionTokens int
	ContextTokens    int
	ContextLimit     int
}

// Model is the root Bubble Tea model.
type Model struct {
	screen    Screen
	cfg       app.Config
	runOpts   app.RunOptions
	modelName string
	modelPath string

	messages      []ChatMessage
	viewport      viewport.Model
	stickToBottom bool
	input         textarea.Model

	width  int
	height int

	statusMsg string
	errMsg    string
	busy      bool
	cancel    context.CancelFunc
	genCh     chan tea.Msg

	contextUsed   int
	contextLimit  int
	sessionPrompt int
	sessionReply  int

	footerCWD    string
	footerBranch string

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
	return tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
}

func initialModel(cfg app.Config, opts app.RunOptions, modelName string) Model {
	ta := textarea.New()
	ta.Prompt = "" // default is a thick left border character (looks like a white bar)
	ta.Placeholder = "Type a message… (/help for commands)"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(inputTextareaLines)
	ta.ShowLineNumbers = false
	km := textarea.DefaultKeyMap
	// Default maps ctrl+m to newline; disable so ctrl+m can open the model picker.
	km.InsertNewline = key.NewBinding(key.WithDisabled())
	ta.KeyMap = km

	vp := viewport.New(80, 20)
	vp.SetContent("")

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(1)
	delegate.SetHeight(1)
	styles := list.NewDefaultItemStyles()
	styles.NormalTitle = styles.NormalTitle.Padding(0)
	styles.SelectedTitle = styles.SelectedTitle.Padding(0, 0, 0, 1)
	styles.DimmedTitle = styles.DimmedTitle.Padding(0)
	delegate.Styles = styles

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
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
		stickToBottom: true,
		contextLimit:  32768,
		modelList:     l,
		settingsEdit:  cfg,
		settingsField: 0,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.refreshModels(),
		m.refreshFooterInfo(),
		m.refreshContextLimit(),
	)
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
	text  string
	err   error
	usage genUsage
}

type genTokenMsg struct {
	text string
}

type genReadyMsg struct {
	contextLimit int
}

type genUsageMsg struct {
	usage genUsage
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		if m.screen == ScreenChat {
			if handled, cmd := m.handleChatKeybinding(msg); handled {
				return m, cmd
			}
			if handled, cmd := m.handleViewportScroll(msg); handled {
				return m, cmd
			}
			if !m.busy {
				var c tea.Cmd
				m.input, c = m.input.Update(msg)
				return m, c
			}
			return m, nil
		}
		return m.handleKey(msg)

	case tea.MouseMsg:
		if m.screen == ScreenChat {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			m.stickToBottom = m.viewport.AtBottom()
			return m, cmd
		}

	case footerInfoMsg:
		m.footerCWD = msg.cwd
		m.footerBranch = msg.branch
		return m, nil

	case contextLimitMsg:
		if msg.limit > 0 {
			m.contextLimit = msg.limit
		}
		return m, nil

	case modelsLoadedMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.modelList.SetItems(msg.items)
		if m.modelName == "" && len(msg.names) > 0 {
			m.modelName = msg.names[0]
		}
		return m, m.refreshContextLimit()

	case genReadyMsg:
		if msg.contextLimit > 0 {
			m.contextLimit = msg.contextLimit
		}
		if m.genCh != nil {
			return m, waitGen(m.genCh)
		}
		return m, nil

	case genUsageMsg:
		m.applyUsage(msg.usage)
		if m.genCh != nil {
			return m, waitGen(m.genCh)
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
		if msg.usage.ContextLimit > 0 || msg.usage.ContextTokens > 0 {
			m.applyUsage(msg.usage)
		}
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
		m.input.Focus()
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
	fixedTop := topBarFixedLines()
	fixedBottom := inputAreaFixedLines()
	vpH := m.height - fixedTop - fixedBottom
	if vpH < 4 {
		vpH = 4
	}

	m.viewport.Width = conversationViewportWidth(m.width)
	m.viewport.Height = vpH
	m.input.SetWidth(inputTextWidth(m.width))
	m.input.SetHeight(inputTextareaLines)

	m.layoutModelPicker()
}

func (m *Model) layoutModelPicker() {
	const chrome = 5 // title, gap below title, hint, spacing
	h := m.height - chrome
	if h < 8 {
		h = 8
	}
	w := m.width
	if w < 20 {
		w = 20
	}
	m.modelList.SetSize(w, h)
}

func (m *Model) refreshViewport() {
	m.viewport.SetContent(renderTranscript(m.messages, m.viewport.Width))
	if m.stickToBottom {
		m.viewport.GotoBottom()
	}
}
