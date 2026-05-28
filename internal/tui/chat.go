package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shai/shai/internal/app"
	"github.com/shai/shai/internal/backend"
)

func (m *Model) viewChat() string {
	header := styleHeader.Render(fmt.Sprintf("shai  │  model: %s  │  device: %s  │  max tokens: %d",
		emptyFallback(m.modelName, "(none)"),
		m.runOpts.Device,
		m.runOpts.MaxTokens,
	))

	chat := m.viewport.View()
	input := m.input.View()

	status := m.statusMsg
	if status == "" {
		status = "Ready"
	}
	if m.busy {
		status = "Generating…"
	}
	statusLine := styleStatus.Render(status)
	if m.errMsg != "" {
		statusLine += "  " + styleError.Render(m.errMsg)
	}

	helpHint := styleStatus.Render("Ctrl+H help · Ctrl+M models · Ctrl+D download · Ctrl+S settings")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		chat,
		styleBorder.Render(input),
		statusLine,
		helpHint,
	)
}

func emptyFallback(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
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

	// Chat screen
	if m.busy && msg.String() == "esc" {
		if m.cancel != nil {
			m.cancel()
		}
		m.busy = false
		m.statusMsg = "Cancelled"
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "ctrl+q":
		return m, tea.Quit
	case "ctrl+l":
		m.messages = nil
		m.refreshViewport()
		m.errMsg = ""
		return m, nil
	case "ctrl+h":
		m.screen = ScreenHelp
		return m, nil
	case "ctrl+m":
		m.screen = ScreenModelPicker
		return m, m.refreshModels()
	case "ctrl+d":
		m.screen = ScreenDownload
		m.dlRepo = ""
		m.dlName = ""
		m.dlField = 0
		return m, nil
	case "ctrl+s":
		m.screen = ScreenSettings
		m.settingsEdit = m.cfg
		m.settingsField = 0
		return m, nil
	case "esc":
		m.input.SetValue("")
		return m, nil
	case "enter":
		if m.busy {
			return m, nil
		}
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}
		if strings.HasPrefix(text, "/") {
			return m.handleSlashCommand(text)
		}
		m.input.SetValue("")
		m.messages = append(m.messages, ChatMessage{Role: "user", Content: text})
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: ""})
		m.refreshViewport()
		m.busy = true
		m.errMsg = ""
		m.statusMsg = "Loading model…"
		return m, m.runGeneration()
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
		m.statusMsg = fmt.Sprintf("Max tokens: %d", n)
	case "clear":
		m.messages = nil
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
		var genErr error
		genErr = backend.RunGenerate(ctx, opts, func(ev backend.Event) {
			switch ev.Type {
			case "token":
				full += ev.Text
				ch <- genTokenMsg{text: ev.Text}
			case "done":
				if ev.Text != "" {
					full = ev.Text
				}
			case "error":
				genErr = fmt.Errorf("%s", ev.Message)
			}
		})
		if genErr != nil {
			genErr = app.WrapNPUError(opts.Device, genErr)
		}
		ch <- genDoneMsg{text: full, err: genErr}
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
