package logs

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/theme"
)

type DetailComponent interface {
	tea.Model
	layout.Sizeable
	layout.Bindings
}

type HideFullLogMsg struct{}

type fullLogKeyMap struct {
	Enter key.Binding
	Escape key.Binding
}

func (k fullLogKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Escape}
}

func (k fullLogKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{
		k.Enter,
		k.Escape,
	}}
}

type detailCmp struct {
	width, height int
	currentLog    logging.LogMessage
	viewport      viewport.Model
	keys          fullLogKeyMap
}

func (i *detailCmp) Init() tea.Cmd {
	messages := logging.List()
	if len(messages) == 0 {
		return nil
	}
	i.currentLog = messages[0]
	return nil
}

func (i *detailCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case ShowFullLogMsg:
		if msg.ID != i.currentLog.ID {
			i.currentLog = logging.LogMessage(msg)
			i.updateContent()
		}
	case selectedLogMsg:
		if msg.ID != i.currentLog.ID {
			i.currentLog = logging.LogMessage(msg)
			i.updateContent()
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, i.keys.Enter), key.Matches(msg, i.keys.Escape):
			return i, func() tea.Msg { return HideFullLogMsg{} }
		}
	}

	// This line was missing in the previous iteration, causing key events not to be processed by the viewport
	i.viewport, cmd = i.viewport.Update(msg)
	return i, cmd
}

func (i *detailCmp) updateContent() {
	var content strings.Builder
	t := theme.CurrentTheme()

	// Format the header with timestamp and level
	timeStyle := lipgloss.NewStyle().Foreground(t.TextMuted())
	levelStyle := getLevelStyle(i.currentLog.Level)

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		timeStyle.Render(i.currentLog.Time.Format(time.RFC3339)),
		"  ",
		levelStyle.Render(i.currentLog.Level),
	)

	content.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	content.WriteString("\n\n")

	// Message with styling
	messageStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Text())
	content.WriteString(messageStyle.Render("Message:"))
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Padding(0, 2).Width(i.width - 4).Render(wordwrap.String(i.currentLog.Message, i.width - 4)))
	content.WriteString("\n\n")

	// Attributes section
	if len(i.currentLog.Attributes) > 0 {
		attrHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Text())
		content.WriteString(attrHeaderStyle.Render("Attributes:"))
		content.WriteString("\n")

		// Create a table-like display for attributes
		keyStyle := lipgloss.NewStyle().Foreground(t.Primary()).Bold(true)
		valueStyle := lipgloss.NewStyle().Foreground(t.Text())

		for _, attr := range i.currentLog.Attributes {
			attrLine := fmt.Sprintf("%s: %s",
				keyStyle.Render(attr.Key),
				valueStyle.Render(attr.Value),
			)
			content.WriteString(lipgloss.NewStyle().Padding(0, 2).Width(i.width - 4).Render(wordwrap.String(attrLine, i.width - 4)))
			content.WriteString("\n")
		}
	}

	i.viewport.SetContent(content.String())
}

func getLevelStyle(level string) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true)
	t := theme.CurrentTheme()
	
	switch strings.ToLower(level) {
	case "info":
		return style.Foreground(t.Info())
	case "warn", "warning":
		return style.Foreground(t.Warning())
	case "error", "err":
		return style.Foreground(t.Error())
	case "debug":
		return style.Foreground(t.Success())
	default:
		return style.Foreground(t.Text())
	}
}

func (i *detailCmp) View() string {
	t := theme.CurrentTheme()
	return styles.ForceReplaceBackgroundWithLipgloss(i.viewport.View(), t.Background())
}

func (i *detailCmp) GetSize() (int, int) {
	return i.width, i.height
}

func (i *detailCmp) SetSize(width int, height int) tea.Cmd {
	i.width = width
	i.height = height
	i.viewport.Width = i.width
	i.viewport.Height = i.height
		i.updateContent()
	return nil
}

func (i *detailCmp) BindingKeys() []key.Binding {
	return i.keys.ShortHelp()
}

func NewLogsDetails() DetailComponent {
	return &detailCmp{
		viewport: viewport.New(0, 0),
		keys: fullLogKeyMap{
			Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "exit full view")),
			Escape: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "exit full view")),
		},
	}
}
