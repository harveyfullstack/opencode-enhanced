package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/message"
	utilComponents "github.com/opencode-ai/opencode/internal/tui/components/util"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/theme"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

type RewindItem struct {
	message message.Message
}

func (ri *RewindItem) Render(selected bool, width int) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	itemStyle := baseStyle.
		Width(width).
		Padding(0, 1)

	if selected {
		itemStyle = itemStyle.
			Background(t.Background()).
			Foreground(t.Primary()).
			Bold(true)
	}

	return itemStyle.Render(ri.message.GetTextContent())
}

func (ri *RewindItem) DisplayValue() string {
	return ri.message.GetTextContent()
}

func (ri *RewindItem) GetValue() string {
	return ri.message.ID
}

func NewRewindItem(msg message.Message) utilComponents.SimpleListItem {
	return &RewindItem{message: msg}
}

type RewindSelectedMsg struct {
	MessageID string
}

type RewindDialogCloseMsg struct{}

type RewindDialog interface {
	tea.Model
	layout.Bindings
	SetWidth(width int)
	SetMessages(messages []message.Message)
}

type rewindDialogCmp struct {
	width    int
	height   int
	listView utilComponents.SimpleList[utilComponents.SimpleListItem]
	messages []message.Message
}

type rewindDialogKeyMap struct {
	Select key.Binding
	Cancel key.Binding
}

var rewindDialogKeys = rewindDialogKeyMap{
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select message"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func (r *rewindDialogCmp) Init() tea.Cmd {
	return nil
}

func (r *rewindDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, rewindDialogKeys.Select):
			item, i := r.listView.GetSelectedItem()
			if i == -1 {
				return r, nil
			}
			return r, tea.Batch(
				util.CmdHandler(RewindSelectedMsg{MessageID: item.GetValue()}),
				r.close(),
			)
		case key.Matches(msg, rewindDialogKeys.Cancel):
			return r, r.close()
		}
	}

	u, cmd := r.listView.Update(msg)
	r.listView = u.(utilComponents.SimpleList[utilComponents.SimpleListItem])
	cmds = append(cmds, cmd)

	return r, tea.Batch(cmds...)
}

func (r *rewindDialogCmp) View() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	maxWidth := 80

	r.listView.SetMaxWidth(maxWidth)

	return baseStyle.Padding(0, 0).
		Border(lipgloss.NormalBorder()).
		BorderBottom(false).
		BorderRight(false).
		BorderLeft(false).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Width(r.width).
		Render(r.listView.View())
}

func (r *rewindDialogCmp) SetWidth(width int) {
	r.width = width
}

func (r *rewindDialogCmp) SetMessages(messages []message.Message) {
	r.messages = messages
	items := make([]utilComponents.SimpleListItem, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == message.User && strings.TrimSpace(msg.GetTextContent()) != "" {
			items = append(items, NewRewindItem(msg))
		}
	}
	r.listView.SetItems(items)
}

func (r *rewindDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(rewindDialogKeys)
}

func (r *rewindDialogCmp) close() tea.Cmd {
	r.listView.SetItems([]utilComponents.SimpleListItem{})
	return util.CmdHandler(RewindDialogCloseMsg{})
}

func NewRewindDialogCmp() RewindDialog {
	li := utilComponents.NewSimpleList[utilComponents.SimpleListItem](
		[]utilComponents.SimpleListItem{},
		7,
		"No messages found",
		false,
	)

	return &rewindDialogCmp{
		listView: li,
	}
}
