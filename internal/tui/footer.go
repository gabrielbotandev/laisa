package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gabrielbotandev/laisa/internal/app"
)

type footerInfoMsg struct {
	cwd    string
	branch string
}

func (m *Model) refreshFooterInfo() tea.Cmd {
	return func() tea.Msg {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "?"
		}
		displayCWD := cwd
		home, _ := os.UserHomeDir()
		if home != "" && cwd != "?" && strings.HasPrefix(cwd, home) {
			displayCWD = "~" + strings.TrimPrefix(cwd, home)
		}

		branch := ""
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		if cwd != "?" {
			cmd.Dir = cwd
		}
		if out, err := cmd.Output(); err == nil {
			branch = strings.TrimSpace(string(out))
		}
		if branch == "" {
			branch = "(no git)"
		}

		return footerInfoMsg{cwd: displayCWD, branch: branch}
	}
}

func (m *Model) viewFooter() string {
	left := m.footerLeft()
	right := m.footerRight()
	return joinFooterRow(m.width, left, right)
}

func (m *Model) footerLeft() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s (%s)", m.footerCWD, m.footerBranch))
	b.WriteString(" · ")
	b.WriteString(fmt.Sprintf("↑%s ↓%s", formatTokenCount(m.sessionPrompt), formatTokenCount(m.sessionReply)))
	b.WriteString(" · ")
	b.WriteString(formatContextUsage(m.contextUsed, m.contextLimit))

	switch {
	case m.busy:
		b.WriteString(" · generating…")
	case m.errMsg != "":
		b.WriteString(" · ")
		b.WriteString(styleError.Render(m.errMsg))
	case m.statusMsg != "" && m.statusMsg != "Ready":
		b.WriteString(" · ")
		b.WriteString(m.statusMsg)
	}

	return b.String()
}

func (m *Model) footerRight() string {
	return fmt.Sprintf("%s · %s",
		emptyFallback(m.modelName, "no model"),
		m.runOpts.Device,
	)
}

func joinFooterRow(width int, left, right string) string {
	leftS := styleFooter.Render(left)
	rightS := styleFooter.Render(right)
	gap := width - lipgloss.Width(leftS) - lipgloss.Width(rightS)
	if gap < 1 {
		gap = 1
	}
	return leftS + strings.Repeat(" ", gap) + rightS
}

func formatTokenCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func formatContextUsage(used, limit int) string {
	if limit <= 0 {
		limit = 32768
	}
	if used < 0 {
		used = 0
	}
	pct := float64(used) / float64(limit) * 100
	return fmt.Sprintf("%.1f%%/%s", pct, formatTokenCount(limit))
}

func (m *Model) refreshContextLimit() tea.Cmd {
	return func() tea.Msg {
		path, err := app.ResolveModelOrDefault(m.modelName, m.cfg)
		if err != nil {
			return contextLimitMsg{limit: 32768}
		}
		return contextLimitMsg{limit: app.ContextLimitFromModel(path)}
	}
}

type contextLimitMsg struct {
	limit int
}

func (m *Model) resetContextStats() {
	m.contextUsed = 0
	m.sessionPrompt = 0
	m.sessionReply = 0
}

func (m *Model) applyUsage(u genUsage) {
	if u.ContextLimit > 0 {
		m.contextLimit = u.ContextLimit
	}
	if u.ContextTokens > 0 {
		m.contextUsed = u.ContextTokens
	}
	if u.PromptTokens > 0 {
		m.sessionPrompt += u.PromptTokens
	}
	if u.CompletionTokens > 0 {
		m.sessionReply += u.CompletionTokens
	}
}
