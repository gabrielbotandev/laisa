package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) viewSettings() string {
	s := m.settingsEdit
	lines := []string{
		styleHelpTitle.Render("Settings (saved to config on Enter)"),
		fmt.Sprintf("%s default_model: %s", fieldMark(m.settingsField, 0), s.DefaultModel),
		fmt.Sprintf("%s default_device: %s", fieldMark(m.settingsField, 1), s.DefaultDevice),
		fmt.Sprintf("%s max_tokens: %d", fieldMark(m.settingsField, 2), s.MaxTokens),
		fmt.Sprintf("%s system_prompt: (first line shown)", fieldMark(m.settingsField, 3)),
		firstLine(s.SystemPrompt, 60),
		"",
		"↑/↓: field · type to edit · Enter: save · Esc: cancel",
	}
	return styleBorder.Width(m.width - 4).Render(strings.Join(lines, "\n"))
}

func fieldMark(current, idx int) string {
	if current == idx {
		return ">"
	}
	return " "
}

func firstLine(s string, max int) string {
	line := strings.SplitN(s, "\n", 2)[0]
	if len(line) > max {
		return line[:max] + "…"
	}
	return line
}

func (m *Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenChat
		return m, nil
	case "up":
		if m.settingsField > 0 {
			m.settingsField--
		}
	case "down":
		if m.settingsField < 3 {
			m.settingsField++
		}
	case "enter":
		if err := m.settingsEdit.Save(); err != nil {
			m.errMsg = err.Error()
		} else {
			m.cfg = m.settingsEdit
			m.runOpts.Device = m.cfg.DefaultDevice
			m.runOpts.MaxTokens = m.cfg.MaxTokens
			m.runOpts.SystemPrompt = m.cfg.SystemPrompt
			if m.cfg.DefaultModel != "" {
				m.modelName = m.cfg.DefaultModel
			}
			m.statusMsg = "Settings saved"
		}
		m.screen = ScreenChat
		return m, nil
	case "backspace":
		m.editSettingsField(true, "")
	default:
		if len(msg.Runes) > 0 {
			m.editSettingsField(false, string(msg.Runes))
		}
	}
	return m, nil
}

func (m *Model) editSettingsField(bs bool, edit string) {
	s := &m.settingsEdit
	trim := func(str *string) {
		if bs && len(*str) > 0 {
			*str = (*str)[:len(*str)-1]
		} else if !bs {
			*str += edit
		}
	}
	switch m.settingsField {
	case 0:
		trim(&s.DefaultModel)
	case 1:
		trim(&s.DefaultDevice)
	case 2:
		if bs {
			str := strconv.Itoa(s.MaxTokens)
			if len(str) > 0 {
				str = str[:len(str)-1]
				n, _ := strconv.Atoi(str)
				s.MaxTokens = n
			}
		} else {
			str := strconv.Itoa(s.MaxTokens) + edit
			n, _ := strconv.Atoi(str)
			if n > 0 {
				s.MaxTokens = n
			}
		}
	case 3:
		trim(&s.SystemPrompt)
	}
}
