package dialog

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/taigrr/crush/internal/ui/common"
)

// ForkID is the identifier for the fork dialog.
const ForkID = "fork"

const (
	forkDialogMaxWidth = 60
)

// forkFilesMode selects what a fork does with the filesystem.
type forkFilesMode int

const (
	// forkFilesKeep leaves the working tree untouched (default).
	forkFilesKeep forkFilesMode = iota
	// forkFilesWorktree creates an isolated worktree for the fork.
	forkFilesWorktree
	// forkFilesRestore restores the live working tree to the fork point.
	forkFilesRestore
	forkFilesModeCount
)

func (m forkFilesMode) label() string {
	switch m {
	case forkFilesWorktree:
		return "New worktree"
	case forkFilesRestore:
		return "Restore working tree to this point (overwrites files!)"
	default:
		return "Keep as-is"
	}
}

// Fork is a dialog for forking a conversation from a specific message.
type Fork struct {
	com       *common.Common
	help      help.Model
	input     textinput.Model
	sessionID string
	messageID string

	filesMode forkFilesMode

	keyMap struct {
		Confirm        key.Binding
		ToggleWorktree key.Binding
		Close          key.Binding
	}
}

var _ Dialog = (*Fork)(nil)

// NewFork creates a new Fork dialog.
func NewFork(com *common.Common, sessionID, messageID, defaultTitle string) *Fork {
	f := &Fork{
		com:       com,
		sessionID: sessionID,
		messageID: messageID,
		filesMode: forkFilesKeep, // Default: never touch the working tree.
	}

	help := help.New()
	help.Styles = com.Styles.DialogHelpStyles()
	f.help = help

	f.input = textinput.New()
	f.input.SetVirtualCursor(false)
	f.input.Placeholder = "Session name"
	f.input.SetStyles(com.Styles.TextInput)
	f.input.SetValue(defaultTitle)
	f.input.Focus()

	f.keyMap.Confirm = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "fork"),
	)
	f.keyMap.ToggleWorktree = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "cycle files mode"),
	)
	f.keyMap.Close = CloseKey

	return f
}

// ID implements Dialog.
func (f *Fork) ID() string {
	return ForkID
}

// HandleMsg implements Dialog.
func (f *Fork) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, f.keyMap.Close):
			return ActionClose{}
		case key.Matches(msg, f.keyMap.ToggleWorktree):
			f.filesMode = (f.filesMode + 1) % forkFilesModeCount
			return nil
		case key.Matches(msg, f.keyMap.Confirm):
			title := f.input.Value()
			if title == "" {
				title = "Fork"
			}
			return ActionForkConversation{
				SessionID:          f.sessionID,
				MessageID:          f.messageID,
				NewSessionTitle:    title,
				CreateWorktree:     f.filesMode == forkFilesWorktree,
				RestoreWorkingTree: f.filesMode == forkFilesRestore,
			}
		default:
			var cmd tea.Cmd
			f.input, cmd = f.input.Update(msg)
			return ActionCmd{cmd}
		}
	}
	return nil
}

// InitialCmd implements Dialog.
func (f *Fork) InitialCmd() tea.Cmd {
	return nil
}

// Cursor implements Dialog.
func (f *Fork) Cursor() *tea.Cursor {
	return InputCursor(f.com.Styles, f.input.Cursor())
}

// Draw implements Dialog.
func (f *Fork) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	t := f.com.Styles
	width := max(0, min(forkDialogMaxWidth, area.Dx()))
	innerWidth := width - t.Dialog.View.GetHorizontalFrameSize()

	f.input.SetWidth(innerWidth - t.Dialog.InputPrompt.GetHorizontalFrameSize() - 1)
	f.help.SetWidth(innerWidth)

	rc := NewRenderContext(t, width)
	rc.Title = "Fork Conversation"
	rc.TitleInfo = "from this message"

	// Session name input.
	inputLabel := t.Dialog.SecondaryText.Render("Session name:")
	inputView := t.Dialog.InputPrompt.Render(f.input.View())
	rc.AddPart(lipgloss.JoinVertical(lipgloss.Left, inputLabel, inputView))

	// Files mode selector.
	filesLabel := t.Dialog.SecondaryText.Render("Files:")
	filesValue := t.Dialog.PrimaryText.Render(f.filesMode.label() + "  (tab to change)")
	rc.AddPart(lipgloss.JoinVertical(lipgloss.Left, filesLabel, filesValue))

	rc.Help = f.help.View(f)

	view := rc.Render()

	cur := f.Cursor()
	DrawCenterCursor(scr, area, view, cur)
	return cur
}

// ShortHelp implements help.KeyMap.
func (f *Fork) ShortHelp() []key.Binding {
	return []key.Binding{
		f.keyMap.ToggleWorktree,
		f.keyMap.Confirm,
		f.keyMap.Close,
	}
}

// FullHelp implements help.KeyMap.
func (f *Fork) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{f.keyMap.Confirm, f.keyMap.ToggleWorktree, f.keyMap.Close},
	}
}
