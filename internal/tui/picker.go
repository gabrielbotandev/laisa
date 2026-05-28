package tui

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) viewModelPicker() string {
	m.layoutModelPicker()

	title := styleHelpTitle.Margin(0, 0, 1, 0).Render("Select model")
	hint := styleStatus.Render("Enter: select · Esc: back")

	return lipgloss.JoinVertical(lipgloss.Left, title, m.modelList.View(), hint)
}

func (m *Model) handleModelPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenChat
		return m, nil
	case "enter":
		item := m.modelList.SelectedItem()
		if item != nil {
			if mi, ok := item.(modelItem); ok {
				m.modelName = mi.name
				m.statusMsg = "Model: " + mi.name
			}
		}
		m.screen = ScreenChat
		return m, m.refreshContextLimit()
	}
	var cmd tea.Cmd
	m.modelList, cmd = m.modelList.Update(msg)
	return m, cmd
}
