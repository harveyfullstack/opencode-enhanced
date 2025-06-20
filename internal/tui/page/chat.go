package page

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/app"
	"github.com/opencode-ai/opencode/internal/completions"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/opencode-ai/opencode/internal/session"
	"github.com/opencode-ai/opencode/internal/tui/components/chat"
	"github.com/opencode-ai/opencode/internal/tui/components/dialog"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

var ChatPage PageID = "chat"

type chatPage struct {
	app                  *app.App
	editor               layout.Container
	messages             layout.Container
	layout               layout.SplitPaneLayout
	session              session.Session
	completionDialog     dialog.CompletionDialog
	showCompletionDialog bool
	rewindDialog         dialog.RewindDialog
	showRewindDialog     bool
	history              []string
	historyIndex         int
}

type ChatKeyMap struct {
	ShowCompletionDialog key.Binding
	NewSession           key.Binding
	Cancel               key.Binding
	HistoryUp            key.Binding
	HistoryDown          key.Binding
	RewindSession        key.Binding
}

var keyMap = ChatKeyMap{
	ShowCompletionDialog: key.NewBinding(
		key.WithKeys("@"),
		key.WithHelp("@", "Complete"),
	),
	NewSession: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new session"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	HistoryUp: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("up", "previous message"),
	),
	HistoryDown: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("down", "next message"),
	),
	RewindSession: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "rewind session"),
	),
}

func (p *chatPage) Init() tea.Cmd {
	cmds := []tea.Cmd{
		p.layout.Init(),
		p.completionDialog.Init(),
	}
	return tea.Batch(cmds...)
}

func (p *chatPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := p.layout.SetSize(msg.Width, msg.Height)
		cmds = append(cmds, cmd)
	case dialog.CompletionDialogCloseMsg:
		p.showCompletionDialog = false
	case chat.SendMsg:
		cmd := p.sendMessage(msg.Text, msg.Attachments)
		if cmd != nil {
			return p, cmd
		}
	case dialog.CommandRunCustomMsg:
		// Check if the agent is busy before executing custom commands
		if p.app.CoderAgent.IsBusy() {
			return p, util.ReportWarn("Agent is busy, please wait before executing a command...")
		}
		
		// Process the command content with arguments if any
		content := msg.Content
		if msg.Args != nil {
			// Replace all named arguments with their values
			for name, value := range msg.Args {
				placeholder := "$" + name
				content = strings.ReplaceAll(content, placeholder, value)
			}
		}
		
		// Handle custom command execution
		cmd := p.sendMessage(content, nil)
		if cmd != nil {
			return p, cmd
		}
	case chat.SessionSelectedMsg:
	
		p.session = msg
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keyMap.ShowCompletionDialog):
			p.showCompletionDialog = true
			// Continue sending keys to layout->chat
		case key.Matches(msg, keyMap.NewSession):
			p.session = session.Session{}
			p.history = []string{}
			p.historyIndex = 0
			p.showRewindDialog = false // Hide rewind dialog on new session
			return p, tea.Batch(
				util.CmdHandler(chat.SessionClearedMsg{}),
			)
		case key.Matches(msg, keyMap.Cancel):
			if p.showRewindDialog {
				p.showRewindDialog = false
				return p, nil
			}
			if p.session.ID != "" {
				// Cancel the current session's generation process
				// This allows users to interrupt long-running operations
				p.app.CoderAgent.Cancel(p.session.ID)
				return p, nil
			}
		case key.Matches(msg, keyMap.RewindSession):
			if p.showRewindDialog {
				p.showRewindDialog = false
				return p, nil
			}
			if p.session.ID != "" {
				logging.Debug("Ctrl+M pressed, attempting to show rewind dialog")
				messages, err := p.app.Messages.List(context.Background(), p.session.ID)
				if err != nil {
					logging.Error("Failed to list messages for rewind dialog", "error", err)
					return p, util.ReportError(err)
				}
				p.rewindDialog.SetMessages(messages)
				p.showRewindDialog = true
				logging.Debug("Rewind dialog set to visible", "message_count", len(messages))
			}
		}
	case dialog.RewindSelectedMsg:
		cmd := p.rewindSession(msg.MessageID)
		if cmd != nil {
			// return p, cmd // This was returning early, preventing batching.
			cmds = append(cmds, cmd)
		}
		p.showRewindDialog = false
	case dialog.RewindDialogCloseMsg:
		p.showRewindDialog = false
	case tea.MouseMsg:
		// Pass mouse events to the messages component
		m, cmd := p.messages.Update(msg)
		p.messages = m.(layout.Container)
		cmds = append(cmds, cmd)
	}
	if p.showCompletionDialog {
		context, contextCmd := p.completionDialog.Update(msg)
		p.completionDialog = context.(dialog.CompletionDialog)
		cmds = append(cmds, contextCmd)

		// Doesn't forward event if enter key is pressed
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "enter" {
				return p, tea.Batch(cmds...)
			}
		}
	} else if p.showRewindDialog {
		context, contextCmd := p.rewindDialog.Update(msg)
		p.rewindDialog = context.(dialog.RewindDialog)
		cmds = append(cmds, contextCmd)
	}

	u, cmd := p.layout.Update(msg)
	cmds = append(cmds, cmd)
	p.layout = u.(layout.SplitPaneLayout)

	return p, tea.Batch(cmds...)
}



func (p *chatPage) sendMessage(text string, attachments []message.Attachment) tea.Cmd {
	var cmds []tea.Cmd
	isNewSession := p.session.ID == ""
	if isNewSession {
		session, err := p.app.Sessions.Create(context.Background(), "New Session")
		if err != nil {
			return util.ReportError(err)
		}

		p.session = session

	}

	if isNewSession {
		cmds = append(cmds, util.CmdHandler(chat.SessionSelectedMsg(p.session)))
	}

	_, err := p.app.CoderAgent.Run(context.Background(), p.session.ID, text, attachments...)
	if err != nil {
		return util.ReportError(err)
	}
	return tea.Batch(cmds...)
}

func (p *chatPage) rewindSession(messageID string) tea.Cmd {
	var cmds []tea.Cmd
	err := p.app.Messages.DeleteFromID(context.Background(), p.session.ID, messageID)
	if err != nil {
		return util.ReportError(err)
	}

	updatedSession, err := p.app.Sessions.Get(context.Background(), p.session.ID)
	if err != nil {
		return util.ReportError(err)
	}
	p.session = updatedSession
	cmds = append(cmds, chat.RefreshMessagesCmd())

	// After deleting messages, we might want to refresh the view or send a notification.
	// For now, returning nil. A command to update message list could be added here if needed,
	// e.g., return util.CmdHandler(chat.MessagesChangedMsg{})

	// Refresh the messages displayed in the chat
	contentModel := p.messages.GetContentModel()
	if messagesCmp, ok := contentModel.(*chat.MessagesCmp); ok {
		cmd := messagesCmp.SetSession(p.session)
		cmds = append(cmds, cmd)
	} else {
		// This case should ideally not happen if the component is set up correctly.
		// Handle error or log, depending on application's error handling strategy.
		logging.Error("Failed to type assert content model to *chat.MessagesCmp in rewindSession")
	}

	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

func (p *chatPage) SetSize(width, height int) tea.Cmd {
	return p.layout.SetSize(width, height)
}

func (p *chatPage) GetSize() (int, int) {
	return p.layout.GetSize()
}

func (p *chatPage) View() string {
	layoutView := p.layout.View()

	if p.showCompletionDialog {
		_, layoutHeight := p.layout.GetSize()
		editorWidth, editorHeight := p.editor.GetSize()

		p.completionDialog.SetWidth(editorWidth)
		overlay := p.completionDialog.View()

		layoutView = layout.PlaceOverlay(
			0,
			layoutHeight-editorHeight-lipgloss.Height(overlay),
			overlay,
			layoutView,
			false,
		)
	} else if p.showRewindDialog {
		_, layoutHeight := p.layout.GetSize()
		editorWidth, editorHeight := p.editor.GetSize()

		p.rewindDialog.SetWidth(editorWidth)
		overlay := p.rewindDialog.View()

		layoutView = layout.PlaceOverlay(
			0,
			layoutHeight-editorHeight-lipgloss.Height(overlay),
			overlay,
			layoutView,
			false,
		)
	}

	return layoutView
}

func (p *chatPage) BindingKeys() []key.Binding {
	var bindings []key.Binding
	bindings = append(bindings, p.messages.BindingKeys()...)
	bindings = append(bindings, p.editor.BindingKeys()...)
	bindings = append(bindings, layout.KeyMapToSlice(keyMap)...)
	bindings = append(bindings, p.rewindDialog.BindingKeys()...)
	return bindings
}

func NewChatPage(app *app.App) tea.Model {
	cg := completions.NewFileAndFolderContextGroup()
	completionDialog := dialog.NewCompletionDialogCmp(cg)
	rewindDialog := dialog.NewRewindDialogCmp()

	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(app),
		layout.WithPadding(1, 1, 0, 1),
	)
	editorContainer := layout.NewContainer(
		chat.NewEditorCmp(app),
		layout.WithBorder(true, false, false, false),
	)
	return &chatPage{
		app:              app,
		editor:           editorContainer,
		messages:         messagesContainer,
		completionDialog: completionDialog,
		rewindDialog:     rewindDialog,
		layout: layout.NewSplitPane(
			layout.WithLeftPanel(messagesContainer),
			layout.WithBottomPanel(editorContainer),
		),
	}
}
