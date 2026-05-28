package tui

// Conversation (transcript viewport) spacing in terminal cells.
const (
	convoSideMargin  = 2 // horizontal margin — centers the chat column
	convoPadH        = 2 // inner padding left/right inside the conversation
	convoPadTop      = 1 // inner padding above first message
	convoPadBottom   = 2 // inner padding below last message (gap before input)
	convoMessageGap  = 1 // extra blank lines between user/assistant blocks
)

// Input composer — top/bottom rule only, full width to match footer bar.
const (
	inputTextareaLines = 2
	inputBorderLines   = 2 // top + bottom
	inputInnerPadH     = 1
)

func conversationViewportWidth(termWidth int) int {
	w := termWidth - 2*convoSideMargin
	if w < 24 {
		return 24
	}
	return w
}

func transcriptContentWidth(viewportWidth int) int {
	w := viewportWidth - 2*convoPadH
	if w < 16 {
		return 16
	}
	return w
}

func inputTextWidth(termWidth int) int {
	w := termWidth - 2*inputInnerPadH
	if w < 16 {
		return 16
	}
	return w
}

func (m *Model) renderComposer() string {
	return styleInput.
		Width(m.width).
		Padding(0, inputInnerPadH).
		Render(m.input.View())
}

func inputAreaFixedLines() int {
	const footerLines = 1
	return inputTextareaLines + inputBorderLines + footerLines
}

func topBarFixedLines() int {
	const (
		headerLines      = 1
		shortcutLines    = 1
		headerBottomGap  = 1 // margin below shortcuts, before conversation
	)
	return headerLines + shortcutLines + headerBottomGap
}

func (m *Model) renderConversation() string {
	inner := m.viewport.View()
	return styleConversation.
		Margin(0, convoSideMargin, 0, convoSideMargin).
		Render(inner)
}
