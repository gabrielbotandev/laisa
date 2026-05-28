package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gabrielbotandev/laisa/internal/app"
	"github.com/gabrielbotandev/laisa/internal/backend"
)

const chatShortcutsLine = "pgup/pgdn scroll · end latest · ctrl+m models · ctrl+d download · ctrl+s settings · ctrl+h help · ctrl+c quit"

func (m *Model) viewChat() string {
	topBar := styleTopBar.Width(m.width).Render(lipgloss.JoinVertical(lipgloss.Left,
		styleHeader.Render("Laisa"),
		styleShortcuts.Render(chatShortcutsLine),
	))

	return lipgloss.JoinVertical(lipgloss.Left,
		topBar,
		m.renderConversation(),
		m.renderComposer(),
		m.viewFooter(),
	)
}

func emptyFallback(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func usageFromBackend(u *backend.UsageStats) genUsage {
	if u == nil {
		return genUsage{}
	}
	return genUsage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		ContextTokens:    u.ContextTokens,
		ContextLimit:     u.ContextLimit,
	}
}

// handleChatKeybinding handles chat-screen shortcuts. Returns true when the key was consumed.
func (m *Model) handleChatKeybinding(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.busy && msg.String() == "esc" {
		if m.cancel != nil {
			m.cancel()
		}
		m.busy = false
		m.statusMsg = "Cancelled"
		return true, nil
	}

	switch msg.String() {
	case "ctrl+c", "ctrl+q":
		return true, tea.Quit
	case "ctrl+l":
		m.messages = nil
		m.resetContextStats()
		m.refreshViewport()
		m.errMsg = ""
		return true, nil
	case "ctrl+h":
		m.screen = ScreenHelp
		return true, nil
	case "ctrl+m", "alt+m":
		if m.busy {
			return true, nil
		}
		m.screen = ScreenModelPicker
		return true, m.refreshModels()
	case "ctrl+d":
		m.screen = ScreenDownload
		m.dlRepo = ""
		m.dlName = ""
		m.dlField = 0
		return true, nil
	case "ctrl+s":
		m.screen = ScreenSettings
		m.settingsEdit = m.cfg
		m.settingsField = 0
		return true, nil
	case "esc":
		m.input.SetValue("")
		return true, nil
	case "enter":
		if m.busy {
			return true, nil
		}
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return true, nil
		}
		m.input.SetValue("")
		if strings.HasPrefix(text, "/") {
			_, cmd := m.handleSlashCommand(text)
			return true, cmd
		}
		m.messages = append(m.messages, ChatMessage{Role: "user", Content: text})
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: ""})
		m.stickToBottom = true
		m.refreshViewport()
		m.busy = true
		m.errMsg = ""
		m.statusMsg = "Loading model…"
		return true, m.runGeneration()
	}

	return false, nil
}

// handleViewportScroll scrolls the conversation when not handled by the input.
func (m *Model) handleViewportScroll(msg tea.KeyMsg) (bool, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "pgup", "ctrl+u":
		m.viewport, cmd = m.viewport.Update(msg)
		m.stickToBottom = false
		return true, cmd
	case "pgdown":
		m.viewport, cmd = m.viewport.Update(msg)
		m.stickToBottom = m.viewport.AtBottom()
		return true, cmd
	case "home":
		m.viewport.GotoTop()
		m.stickToBottom = false
		return true, nil
	case "end":
		m.viewport.GotoBottom()
		m.stickToBottom = true
		return true, nil
	case "up":
		if msg.Alt {
			m.viewport.LineUp(1)
			m.stickToBottom = false
			return true, nil
		}
	case "down":
		if msg.Alt {
			m.viewport.LineDown(1)
			m.stickToBottom = m.viewport.AtBottom()
			return true, nil
		}
	}
	return false, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenHelp:
		if msg.String() == "esc" {
			m.screen = ScreenChat
			m.errMsg = ""
		}
		return m, nil
	case ScreenModelPicker:
		return m.handleModelPickerKey(msg)
	case ScreenDownload:
		return m.handleDownloadKey(msg)
	case ScreenSettings:
		return m.handleSettingsKey(msg)
	}
	return m, nil
}

func (m *Model) handleSlashCommand(text string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(text)
	cmd := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	switch cmd {
	case "help", "h":
		m.screen = ScreenHelp
	case "models":
		m.screen = ScreenModelPicker
		return m, m.refreshModels()
	case "model":
		if len(args) == 0 {
			m.errMsg = "usage: /model <name>"
			return m, nil
		}
		m.modelName = args[0]
		m.errMsg = ""
		m.statusMsg = "Model set to " + args[0]
		return m, m.refreshContextLimit()
	case "download":
		if len(args) == 0 {
			m.screen = ScreenDownload
		} else {
			m.screen = ScreenDownload
			m.dlRepo = args[0]
			m.dlName = repoBase(args[0])
		}
	case "device":
		if len(args) == 0 {
			m.errMsg = "usage: /device CPU|NPU|AUTO"
			return m, nil
		}
		m.runOpts.Device = strings.ToUpper(args[0])
		m.statusMsg = "Device: " + m.runOpts.Device
	case "tokens":
		if len(args) == 0 {
			m.errMsg = "usage: /tokens <number>"
			return m, nil
		}
		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			m.errMsg = "invalid token count"
			return m, nil
		}
		m.runOpts.MaxTokens = n
		m.statusMsg = fmt.Sprintf("Max output tokens: %d", n)
	case "clear":
		m.messages = nil
		m.resetContextStats()
		m.refreshViewport()
		m.errMsg = ""
	case "config":
		out, err := m.cfg.FormatHuman()
		if err != nil {
			m.errMsg = err.Error()
		} else {
			m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: out})
			m.refreshViewport()
		}
	case "quit", "exit":
		return m, tea.Quit
	default:
		m.errMsg = "unknown command; try /help"
	}
	return m, nil
}

func (m *Model) runGeneration() tea.Cmd {
	modelPath, err := app.ResolveModelOrDefault(m.modelName, m.cfg)
	if err != nil {
		return func() tea.Msg { return genDoneMsg{err: err} }
	}

	var history []backend.Message
	for i, msg := range m.messages {
		if i == len(m.messages)-1 && msg.Role == "assistant" && msg.Content == "" {
			continue
		}
		history = append(history, backend.Message{Role: msg.Role, Content: msg.Content})
	}

	opts := backend.GenerateOpts{
		ModelPath:    modelPath,
		Device:       m.runOpts.Device,
		MaxTokens:    m.runOpts.MaxTokens,
		SystemPrompt: m.runOpts.SystemPrompt,
		Messages:     history,
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	m.genCh = make(chan tea.Msg, 64)
	ch := m.genCh
	go func() {
		var full string
		var lastUsage genUsage
		var genErr error
		genErr = backend.RunGenerate(ctx, opts, func(ev backend.Event) {
			switch ev.Type {
			case "ready":
				if ev.ContextLimit > 0 {
					ch <- genReadyMsg{contextLimit: ev.ContextLimit}
				}
			case "token":
				full += ev.Text
				ch <- genTokenMsg{text: ev.Text}
			case "usage":
				lastUsage = usageFromBackend(ev.Usage)
				ch <- genUsageMsg{usage: lastUsage}
			case "done":
				if ev.Text != "" {
					full = ev.Text
				}
				if ev.Usage != nil {
					lastUsage = usageFromBackend(ev.Usage)
				}
			case "error":
				genErr = fmt.Errorf("%s", ev.Message)
			}
		})
		if genErr != nil {
			genErr = app.WrapNPUError(opts.Device, genErr)
		}
		ch <- genDoneMsg{text: full, err: genErr, usage: lastUsage}
		close(ch)
	}()

	return waitGen(ch)
}

func waitGen(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return genDoneMsg{}
		}
		return msg
	}
}

func (m *Model) appendAssistantToken(tok string) {
	if len(m.messages) == 0 {
		return
	}
	last := &m.messages[len(m.messages)-1]
	if last.Role != "assistant" {
		return
	}
	last.Content += tok
	m.refreshViewport()
}

func (m *Model) finalizeAssistant(text string) {
	if len(m.messages) == 0 {
		return
	}
	last := &m.messages[len(m.messages)-1]
	if last.Role == "assistant" {
		last.Content = text
	}
}

func repoBase(repo string) string {
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		return repo[i+1:]
	}
	return repo
}
