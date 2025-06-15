package page

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/tui/components/logs"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
)

var LogsPage PageID = "logs"

type LogPage interface {
	tea.Model
	layout.Sizeable
	layout.Bindings
}
type logsPage struct {
	width, height int
	table         layout.Container
	details       layout.Container
	isFullView    bool
}

func (p *logsPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		return p, p.SetSize(msg.Width, msg.Height)
	case logs.ShowFullLogMsg:
		p.isFullView = true
		var cmd tea.Cmd
		updatedModel, cmd := p.details.Update(msg)
		p.details = updatedModel.(layout.Container)
		cmds = append(cmds, cmd)
		return p, p.SetSize(p.width, p.height)
	case logs.HideFullLogMsg:
		p.isFullView = false
		return p, p.SetSize(p.width, p.height)
	}

	if !p.isFullView {
		var tableCmd tea.Cmd
		updatedTableModel, tableCmd := p.table.Update(msg)
		p.table = updatedTableModel.(layout.Container)
		cmds = append(cmds, tableCmd)
	}
	// Ensure details are updated regardless of full view, as it might receive messages like selectedLogMsg
	var detailsCmd tea.Cmd
	updatedDetailsModel, detailsCmd := p.details.Update(msg)
	p.details = updatedDetailsModel.(layout.Container)
	cmds = append(cmds, detailsCmd)

	return p, tea.Batch(cmds...)
}

func (p *logsPage) View() string {
	style := styles.BaseStyle().Width(p.width).Height(p.height)
	if p.isFullView {
		return style.Render(p.details.View())
	}
	return style.Render(lipgloss.JoinVertical(lipgloss.Top,
		p.table.View(),
		p.details.View(),
	))
}

func (p *logsPage) BindingKeys() []key.Binding {
	if p.isFullView {
		return p.details.BindingKeys()
	}
	return p.table.BindingKeys()
}

// GetSize implements LogPage.
func (p *logsPage) GetSize() (int, int) {
	return p.width, p.height
}

// SetSize implements LogPage.
func (p *logsPage) SetSize(width int, height int) tea.Cmd {
	p.width = width
	p.height = height
	if p.isFullView {
		return p.details.SetSize(width, height)
	}
	return tea.Batch(
		p.table.SetSize(width, height/2),
		p.details.SetSize(width, height/2),
	)
}

func (p *logsPage) Init() tea.Cmd {
	return tea.Batch(
		p.table.Init(),
		p.details.Init(),
	)
}

func NewLogsPage() LogPage {
	return &logsPage{
		table:   layout.NewContainer(logs.NewLogsTable(), layout.WithBorderAll()),
		details: layout.NewContainer(logs.NewLogsDetails(), layout.WithBorderAll()),
	}
}
