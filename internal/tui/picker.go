package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) viewModelPicker() string {
	return lipglossJoin(
		styleHelpTitle.Render("Select model"),
		m.modelList.View(),
		styleStatus.Render("Enter: select · Esc: back"),
	)
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
		return m, nil
	}
	var cmd tea.Cmd
	m.modelList, cmd = m.modelList.Update(msg)
	return m, cmd
}

func lipglossJoin(parts ...string) string {
	var b string
	for i, p := range parts {
		if i > 0 {
			b += "\n"
		}
		b += p
	}
	return b
}
