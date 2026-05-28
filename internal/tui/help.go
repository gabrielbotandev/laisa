package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) viewHelp() string {
	body := `Keyboard shortcuts
  Enter      Send prompt
  Ctrl+C     Quit
  Esc        Cancel input / back
  Ctrl+L     Clear conversation
  Ctrl+H     This help
  Ctrl+M     Model picker
  Ctrl+D     Download model
  Ctrl+S     Settings

Slash commands
  /help              Show help
  /models            Open model picker
  /model <name>      Switch model
  /download <repo>   Download from Hugging Face
  /device CPU|NPU|AUTO
  /tokens <n>        Set max tokens
  /clear             Clear conversation
  /config            Show config
  /quit              Exit

Press Esc to return to chat.`

	return lipgloss.JoinVertical(lipgloss.Left,
		styleHelpTitle.Render("shai help"),
		styleBorder.Width(m.width-4).Render(body),
	)
}
