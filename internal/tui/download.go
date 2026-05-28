package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shai/shai/internal/app"
	"github.com/shai/shai/internal/backend"
)

type downloadDoneMsg struct {
	path string
	err  error
}

func (m *Model) viewDownload() string {
	repoLine := "Repo: " + m.dlRepo
	if m.dlField == 0 {
		repoLine = styleHeader.Render("> " + repoLine + "_")
	}
	name := m.dlName
	if name == "" && m.dlRepo != "" {
		name = repoBase(m.dlRepo)
	}
	nameLine := fmt.Sprintf("Local name [%s]: %s", repoBase(m.dlRepo), name)
	if m.dlField == 1 {
		nameLine = styleHeader.Render("> " + nameLine + "_")
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		styleHelpTitle.Render("Download model"),
		repoLine,
		nameLine,
		"",
		"Tab: switch field · Enter: start download · Esc: back",
	)
	if m.busy {
		body += "\n\nDownloading… (see stderr for progress)"
	}
	return styleBorder.Width(m.width - 4).Render(body)
}

func (m *Model) handleDownloadKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.busy {
			if m.cancel != nil {
				m.cancel()
			}
			m.busy = false
		}
		m.screen = ScreenChat
		return m, nil
	case "tab":
		m.dlField = 1 - m.dlField
		return m, nil
	case "enter":
		if m.busy {
			return m, nil
		}
		repo := strings.TrimSpace(m.dlRepo)
		if repo == "" {
			m.errMsg = "repo ID required"
			return m, nil
		}
		name := strings.TrimSpace(m.dlName)
		if name == "" {
			name = repoBase(repo)
		}
		return m, m.startDownload(repo, name)
	case "backspace":
		if m.dlField == 0 && len(m.dlRepo) > 0 {
			m.dlRepo = m.dlRepo[:len(m.dlRepo)-1]
		} else if m.dlField == 1 && len(m.dlName) > 0 {
			m.dlName = m.dlName[:len(m.dlName)-1]
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			if m.dlField == 0 {
				m.dlRepo += string(msg.Runes)
			} else {
				m.dlName += string(msg.Runes)
			}
		}
	}
	return m, nil
}

func (m *Model) startDownload(repo, name string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.busy = true

	return func() tea.Msg {
		modelsDir, err := app.ModelsDir()
		if err != nil {
			return downloadDoneMsg{err: err}
		}
		dest := filepath.Join(modelsDir, name)
		err = backend.RunDownload(ctx, backend.DownloadOpts{
			Repo:     repo,
			LocalDir: dest,
			Force:    false,
		}, func(ev backend.Event) {
			_ = ev
		})
		if err != nil {
			return downloadDoneMsg{err: err}
		}
		return downloadDoneMsg{path: dest}
	}
}
