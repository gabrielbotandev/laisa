package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleTopBar = lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	styleShortcuts = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	styleStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	styleInput = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(false).
			BorderRight(false)

	styleConversation = lipgloss.NewStyle()

	styleUserBlock = lipgloss.NewStyle().
			Background(lipgloss.Color("238")).
			Foreground(lipgloss.Color("252")).
			Padding(1, convoPadH)

	styleAssistant = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, convoPadH)

	styleFooter = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	styleHelpTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	// Panel chrome matches the prompt input: horizontal rules only.
	styleBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(false).
			BorderRight(false).
			Padding(0, 1)
)
