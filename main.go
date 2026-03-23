package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	purple = lipgloss.Color("#7D56F4")
	subtle = lipgloss.Color("#444444")
	white  = lipgloss.Color("#FAFAFA")
	red    = lipgloss.Color("#E06C75")
	green  = lipgloss.Color("#98C379")
	amber  = lipgloss.Color("#E5C07B")
	muted  = lipgloss.Color("#666666")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 2)

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(purple)

	inactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(subtle)

	searchBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(amber)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Padding(0, 1)

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(1, 2)

	dangerDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(red).
				Padding(1, 2)

	keyHint    = lipgloss.NewStyle().Foreground(purple).Bold(true)
	dirtyStyle = lipgloss.NewStyle().Foreground(red).Bold(true)
	savedStyle = lipgloss.NewStyle().Foreground(green)
	mutedStyle = lipgloss.NewStyle().Foreground(muted).Italic(true)
	amberStyle = lipgloss.NewStyle().Foreground(amber).Bold(true)
)

// ── Pane / Dialog / Sort modes ────────────────────────────────────────────────

type pane int

const (
	paneBrowser pane = iota
	paneEditor
)

type dialogMode int

const (
	dialogNone dialogMode = iota
	dialogNewFile
	dialogRename
	dialogConfirmDelete
)

type sortMode int

const (
	sortByName sortMode = iota
	sortByModified
)

// ── File list item ────────────────────────────────────────────────────────────

type fileItem struct {
	name     string
	path     string
	modified time.Time
}

func (f fileItem) Title() string       { return f.name }
func (f fileItem) Description() string { return f.path }
func (f fileItem) FilterValue() string { return f.name }

// ── Messages ──────────────────────────────────────────────────────────────────

type filesLoadedMsg []fileItem
type fileSavedMsg struct{}
type fileDeletedMsg struct{}
type fileRenamedMsg struct{ newName, newPath string }
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// ── Commands ──────────────────────────────────────────────────────────────────

func loadFiles() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return errMsg{err}
		}
		dir := filepath.Join(home, ".totion")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errMsg{err}
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return errMsg{err}
		}
		var items []fileItem
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				info, _ := e.Info()
				modTime := time.Time{}
				if info != nil {
					modTime = info.ModTime()
				}
				items = append(items, fileItem{
					name:     e.Name(),
					path:     filepath.Join(dir, e.Name()),
					modified: modTime,
				})
			}
		}
		return filesLoadedMsg(items)
	}
}

func createFile(name string) tea.Cmd {
	return func() tea.Msg {
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".totion")
		if !strings.HasSuffix(name, ".md") {
			name += ".md"
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0644); err != nil {
			return errMsg{err}
		}
		return loadFiles()()
	}
}

func saveFile(path, content string) tea.Cmd {
	return func() tea.Msg {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return errMsg{err}
		}
		return fileSavedMsg{}
	}
}

func deleteFile(path string) tea.Cmd {
	return func() tea.Msg {
		if err := os.Remove(path); err != nil {
			return errMsg{err}
		}
		return fileDeletedMsg{}
	}
}

func renameFile(oldPath, newName string) tea.Cmd {
	return func() tea.Msg {
		if !strings.HasSuffix(newName, ".md") {
			newName += ".md"
		}
		newPath := filepath.Join(filepath.Dir(oldPath), newName)
		if err := os.Rename(oldPath, newPath); err != nil {
			return errMsg{err}
		}
		return fileRenamedMsg{newName: newName, newPath: newPath}
	}
}

// ── Component constructors ────────────────────────────────────────────────────

func newEditor(w, h int) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Start writing..."
	ta.ShowLineNumbers = false
	ta.SetWidth(w)
	ta.SetHeight(h)
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#2D2D2D"))
	ta.FocusedStyle.Placeholder = mutedStyle
	ta.BlurredStyle.Placeholder = mutedStyle
	return ta
}

func newFileList() list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(purple).
		BorderLeftForeground(purple)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#9B85F5")).
		BorderLeftForeground(purple)
	delegate.ShowDescription = false

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Notes"
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(purple)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // we handle search ourselves
	l.SetShowHelp(false)
	return l
}

func newTextInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 80
	ti.Width = width
	return ti
}

// ── Model ─────────────────────────────────────────────────────────────────────

type model struct {
	width  int
	height int

	activePane pane
	dialog     dialogMode
	sort       sortMode

	// All files from disk (source of truth for filtering)
	allFiles []fileItem

	fileList      list.Model
	filenameInput textinput.Model // new file & rename dialogs
	searchInput   textinput.Model
	searching     bool
	editor        textarea.Model

	activeFile string
	activePath string
	dirty      bool

	statusMsg string
	quitting  bool
}

func initialModel() model {
	return model{
		activePane:    paneBrowser,
		dialog:        dialogNone,
		sort:          sortByName,
		fileList:      newFileList(),
		filenameInput: newTextInput("my-note.md", 30),
		searchInput:   newTextInput("search notes...", 24),
		editor:        newEditor(80, 20),
		statusMsg:     "Loading notes...",
	}
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return loadFiles()
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncPaneSizes()

	case filesLoadedMsg:
		m.allFiles = []fileItem(msg)
		m.applyFilterAndSort()
		if len(msg) == 0 {
			m.statusMsg = "No notes yet — ctrl+n to create one"
		} else {
			m.statusMsg = fmt.Sprintf("%d note(s)", len(msg))
		}
		if len(msg) > 0 && m.activeFile == "" {
			m.openSelected()
		}

	case fileSavedMsg:
		m.dirty = false
		m.statusMsg = "Saved " + m.activeFile + "  " + wordCountStatus(m.editor.Value())
		// Refresh to update modified time
		return m, loadFiles()

	case fileDeletedMsg:
		m.activeFile = ""
		m.activePath = ""
		m.dirty = false
		m.editor.SetValue("")
		m.activePane = paneBrowser
		return m, loadFiles()

	case fileRenamedMsg:
		m.activeFile = msg.newName
		m.activePath = msg.newPath
		m.statusMsg = "Renamed to " + msg.newName
		return m, loadFiles()

	case errMsg:
		m.statusMsg = "Error: " + msg.Error()

	case tea.KeyMsg:
		// ── Search mode intercepts most keys ──
		if m.searching {
			return m, m.handleSearchKey(msg)
		}

		// ── Dialog: new file / rename ──
		if m.dialog == dialogNewFile || m.dialog == dialogRename {
			return m, m.handleFilenameKey(msg)
		}

		// ── Dialog: confirm delete ──
		if m.dialog == dialogConfirmDelete {
			return m, m.handleDeleteConfirmKey(msg)
		}

		// ── Global shortcuts ──
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+s":
			if m.activePath != "" {
				m.statusMsg = "Saving..."
				return m, saveFile(m.activePath, m.editor.Value())
			}

		case "ctrl+n":
			m.dialog = dialogNewFile
			m.filenameInput.SetValue("")
			m.filenameInput.Placeholder = "my-note.md"
			m.filenameInput.Focus()
			return m, textinput.Blink

		case "ctrl+r":
			if m.activeFile != "" {
				m.dialog = dialogRename
				// Pre-fill with current name (without .md for easy editing)
				current := strings.TrimSuffix(m.activeFile, ".md")
				m.filenameInput.SetValue(current)
				m.filenameInput.Placeholder = m.activeFile
				m.filenameInput.Focus()
				return m, textinput.Blink
			}

		case "ctrl+d":
			if m.activeFile != "" {
				m.dialog = dialogConfirmDelete
				return m, nil
			}

		case "ctrl+f":
			m.searching = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			return m, textinput.Blink

		case "ctrl+o":
			// Toggle sort
			if m.sort == sortByName {
				m.sort = sortByModified
				m.statusMsg = "Sorted by modified time"
			} else {
				m.sort = sortByName
				m.statusMsg = "Sorted by name"
			}
			m.applyFilterAndSort()
			return m, nil

		case "tab":
			if m.activePane == paneBrowser {
				m.activePane = paneEditor
				m.editor.Focus()
				cmds = append(cmds, textarea.Blink)
			} else {
				m.activePane = paneBrowser
				m.editor.Blur()
			}
			return m, tea.Batch(cmds...)

		case "enter":
			if m.activePane == paneBrowser {
				m.openSelected()
				return m, textarea.Blink
			}
		}
	}

	// Route to active component
	if m.dialog == dialogNone && !m.searching {
		switch m.activePane {
		case paneBrowser:
			var cmd tea.Cmd
			m.fileList, cmd = m.fileList.Update(msg)
			cmds = append(cmds, cmd)

		case paneEditor:
			prevVal := m.editor.Value()
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			cmds = append(cmds, cmd)
			if m.activePath != "" && m.editor.Value() != prevVal {
				m.dirty = true
				m.statusMsg = wordCountStatus(m.editor.Value())
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// ── Key handlers ──────────────────────────────────────────────────────────────

func (m *model) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "enter":
		m.searching = false
		m.searchInput.Blur()
		if msg.String() == "esc" {
			m.searchInput.SetValue("")
			m.applyFilterAndSort()
			m.statusMsg = fmt.Sprintf("%d note(s)", len(m.allFiles))
		}
		return nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	// Live filter on every keystroke
	m.applyFilterAndSort()
	q := m.searchInput.Value()
	visible := len(m.fileList.Items())
	if q == "" {
		m.statusMsg = fmt.Sprintf("%d note(s)", len(m.allFiles))
	} else {
		m.statusMsg = fmt.Sprintf("%d / %d match", visible, len(m.allFiles))
	}
	return cmd
}

func (m *model) handleFilenameKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.dialog = dialogNone
		m.filenameInput.Blur()
		return nil
	case "enter":
		name := strings.TrimSpace(m.filenameInput.Value())
		if name == "" {
			return nil
		}
		m.filenameInput.Blur()
		if m.dialog == dialogNewFile {
			m.dialog = dialogNone
			m.statusMsg = "Creating " + name + "..."
			return createFile(name)
		}
		// Rename
		m.dialog = dialogNone
		oldPath := m.activePath
		m.statusMsg = "Renaming..."
		return renameFile(oldPath, name)
	}
	var cmd tea.Cmd
	m.filenameInput, cmd = m.filenameInput.Update(msg)
	return cmd
}

func (m *model) handleDeleteConfirmKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		m.dialog = dialogNone
		path := m.activePath
		m.statusMsg = "Deleting " + m.activeFile + "..."
		return deleteFile(path)
	case "n", "N", "esc":
		m.dialog = dialogNone
		m.statusMsg = "Delete cancelled"
	}
	return nil
}

// ── Business logic ────────────────────────────────────────────────────────────

func (m *model) applyFilterAndSort() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	// Filter
	filtered := make([]fileItem, 0, len(m.allFiles))
	for _, f := range m.allFiles {
		if query == "" || strings.Contains(strings.ToLower(f.name), query) {
			filtered = append(filtered, f)
		}
	}

	// Sort
	switch m.sort {
	case sortByName:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].name < filtered[j].name
		})
	case sortByModified:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].modified.After(filtered[j].modified)
		})
	}

	// Push to list
	items := make([]list.Item, len(filtered))
	for i, f := range filtered {
		items[i] = f
	}
	m.fileList.SetItems(items)
}

func (m *model) openSelected() {
	item, ok := m.fileList.SelectedItem().(fileItem)
	if !ok {
		return
	}
	data, err := os.ReadFile(item.path)
	if err != nil {
		m.statusMsg = "Could not open: " + err.Error()
		return
	}
	m.activeFile = item.name
	m.activePath = item.path
	m.dirty = false
	m.editor.SetValue(string(data))
	m.editor.Focus()
	m.activePane = paneEditor
	m.statusMsg = wordCountStatus(string(data))
}

func (m *model) syncPaneSizes() {
	browserW := m.width / 3
	editorW := m.width - browserW - 4

	listH := m.height - 6
	if listH < 1 {
		listH = 1
	}
	m.fileList.SetSize(browserW-4, listH)

	editorInnerW := editorW - 4
	editorInnerH := m.height - 7
	if editorInnerW < 1 {
		editorInnerW = 1
	}
	if editorInnerH < 1 {
		editorInnerH = 1
	}
	m.editor.SetWidth(editorInnerW)
	m.editor.SetHeight(editorInnerH)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func wordCountStatus(content string) string {
	words := len(strings.Fields(content))
	lines := len(strings.Split(strings.TrimRight(content, "\n"), "\n"))
	return fmt.Sprintf("%d words · %d lines", words, lines)
}

func sortLabel(s sortMode) string {
	switch s {
	case sortByModified:
		return "modified"
	default:
		return "name"
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	if m.width == 0 {
		return "Loading..."
	}

	header := headerStyle.Width(m.width).Render("  Totion 🧠")
	body := m.renderBody()
	statusBar := m.renderStatusBar()

	view := lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)

	switch m.dialog {
	case dialogNewFile:
		return m.overlayDialog(view, m.renderFilenameDialog("New note", purple))
	case dialogRename:
		return m.overlayDialog(view, m.renderFilenameDialog("Rename note", amber))
	case dialogConfirmDelete:
		return m.overlayDialog(view, m.renderDeleteDialog())
	}
	return view
}

func (m model) renderStatusBar() string {
	saveLabel := statusBarStyle.Render(" save  ")
	if m.dirty {
		saveLabel = dirtyStyle.Render(" save*  ")
	}

	sortIndicator := amberStyle.Render("ctrl+o") +
		statusBarStyle.Render(" sort:"+sortLabel(m.sort)+"  ")

	keys := lipgloss.JoinHorizontal(lipgloss.Left,
		keyHint.Render("tab"), statusBarStyle.Render(" switch  "),
		keyHint.Render("ctrl+f"), statusBarStyle.Render(" search  "),
		keyHint.Render("ctrl+n"), statusBarStyle.Render(" new  "),
		keyHint.Render("ctrl+r"), statusBarStyle.Render(" rename  "),
		keyHint.Render("ctrl+d"), statusBarStyle.Render(" delete  "),
		keyHint.Render("ctrl+s"), saveLabel,
		sortIndicator,
		keyHint.Render("ctrl+c"), statusBarStyle.Render(" quit"),
	)

	status := statusBarStyle.Render(m.statusMsg)
	keysW := lipgloss.Width(keys)
	statusW := lipgloss.Width(status)
	gap := m.width - keysW - statusW
	if gap < 1 {
		gap = 1
	}
	return keys + strings.Repeat(" ", gap) + status
}

func (m model) renderBody() string {
	browserW := m.width / 3
	editorW := m.width - browserW - 2

	bodyH := m.height - 4
	if bodyH < 3 {
		bodyH = 3
	}

	// ── Browser pane ──
	var browserBorder lipgloss.Style
	switch {
	case m.searching:
		browserBorder = searchBorderStyle
	case m.activePane == paneBrowser:
		browserBorder = activeBorderStyle
	default:
		browserBorder = inactiveBorderStyle
	}

	// Search bar (shown above the list when active or has a query)
	listView := m.fileList.View()
	query := m.searchInput.Value()
	if m.searching || query != "" {
		searchBar := amberStyle.Render("/ ") + m.searchInput.View()
		listView = lipgloss.JoinVertical(lipgloss.Left, searchBar, listView)
	}

	browserPane := browserBorder.
		Width(browserW - 2).
		Height(bodyH).
		Render(listView)

	// ── Editor pane ──
	editorBorder := inactiveBorderStyle
	if m.activePane == paneEditor {
		editorBorder = activeBorderStyle
	}

	titleText := m.activeFile
	if titleText == "" {
		titleText = "No file open"
	}
	dirtyMark := ""
	if m.dirty {
		dirtyMark = dirtyStyle.Render(" ●")
	} else if m.activeFile != "" {
		dirtyMark = savedStyle.Render(" ✓")
	}

	// Show modified time in title if sorting by modified
	modSuffix := ""
	if m.sort == sortByModified && m.activeFile != "" {
		for _, f := range m.allFiles {
			if f.name == m.activeFile && !f.modified.IsZero() {
				modSuffix = mutedStyle.Render("  " + f.modified.Format("02 Jan 15:04"))
				break
			}
		}
	}

	editorTitle := lipgloss.NewStyle().Bold(true).Foreground(purple).Render(titleText) +
		dirtyMark + modSuffix
	divider := mutedStyle.Render(strings.Repeat("─", editorW-6))

	var editorArea string
	if m.activeFile == "" {
		editorArea = mutedStyle.Render("Select a file or press ctrl+n to create one.")
	} else {
		editorArea = m.editor.View()
	}

	editorInner := lipgloss.JoinVertical(lipgloss.Left,
		editorTitle,
		divider,
		editorArea,
	)
	editorPane := editorBorder.
		Width(editorW - 2).
		Height(bodyH).
		Render(editorInner)

	return lipgloss.JoinHorizontal(lipgloss.Top, browserPane, editorPane)
}

// ── Dialogs ───────────────────────────────────────────────────────────────────

func (m model) renderFilenameDialog(title string, color lipgloss.Color) string {
	heading := lipgloss.NewStyle().Bold(true).Foreground(color).Render(title)
	hint := mutedStyle.Render("enter  confirm  •  esc  cancel")
	content := lipgloss.JoinVertical(lipgloss.Left,
		heading, "", "Filename:", m.filenameInput.View(), "", hint,
	)
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(1, 2)
	return style.Render(content)
}

func (m model) renderDeleteDialog() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(red).Render("Delete note")
	msg := fmt.Sprintf("Delete %q permanently?", m.activeFile)
	hint := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Foreground(red).Bold(true).Render("y"),
		mutedStyle.Render("  yes  "),
		lipgloss.NewStyle().Foreground(purple).Bold(true).Render("n"),
		mutedStyle.Render("  cancel"),
	)
	content := lipgloss.JoinVertical(lipgloss.Left, title, "", msg, "", hint)
	return dangerDialogStyle.Render(content)
}

func (m model) overlayDialog(bg, dialog string) string {
	bgLines := strings.Split(bg, "\n")
	dialogLines := strings.Split(dialog, "\n")

	dialogW := lipgloss.Width(dialog)
	dialogH := len(dialogLines)

	startY := (m.height - dialogH) / 2
	startX := (m.width - dialogW) / 2
	if startX < 0 {
		startX = 0
	}

	for i, dLine := range dialogLines {
		y := startY + i
		if y < 0 || y >= len(bgLines) {
			continue
		}
		bgPlain := []rune(stripANSI(bgLines[y]))
		var prefix string
		if startX <= len(bgPlain) {
			prefix = string(bgPlain[:startX])
		} else {
			prefix = bgLines[y] + strings.Repeat(" ", startX-lipgloss.Width(bgLines[y]))
		}
		bgLines[y] = prefix + dLine
	}
	return strings.Join(bgLines, "\n")
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}