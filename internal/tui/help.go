package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) viewHelp() string {
	body := `Keyboard shortcuts
  Enter          Send prompt
  PgUp/PgDn      Scroll conversation
  Home/End       Top / bottom of chat
  Alt+Up/Dn      Scroll line by line
  Ctrl+C         Quit
  Esc            Clear input draft / cancel generation
  Ctrl+L         Clear conversation
  Ctrl+H         This help
  Ctrl+M         Model picker (Alt+M on some terminals)
  Ctrl+D         Download model
  Ctrl+S         Settings (system prompt)

Slash commands
  /help              Show help
  /models            Open model picker
  /model <name>      Switch model
  /download <repo>   Download from Hugging Face
  /device CPU|NPU|AUTO
  /tokens <n>        Set max output tokens
  /clear             Clear chat
  /config            Show config
  /quit              Exit

Footer shows session token totals and context usage vs model limit.

Press Esc to return to chat.`

	return lipgloss.JoinVertical(lipgloss.Left,
		styleHelpTitle.Render("Laisa help"),
		styleBorder.Width(m.width).Render(body),
	)
}
