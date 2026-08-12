package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	lexer "github.com/joaoofreitas/blin/internal/blin-lang"
)

type editorFinishedMsg struct{ err error }

type viewState int

const (
	stateFiles viewState = iota
	stateMarkdown
	stateCreate
	stateEdit
	stateTimeTracking
)

const (
	sidebarWidth      = 26
	cardWidth         = 38
	cardContentWidth  = cardWidth - 2 // horizontal card padding
	previewHeight     = 10
	cardHeight        = previewHeight + 4 // header, divider, and two borders
	allProjects       = "All Projects"
	dueNotes          = "Due"
	timeTrackingNotes = "Time Tracking"
)

type Note struct {
	Filename    string
	Preview     string
	Time        time.Time
	Due         time.Time
	TimeTracked map[string]TimeTotal
	Projects    []string
	Tags        []string
	FileRefs    []string
}

type TimeTotal struct {
	Hours float64
	Last  time.Time
}

type model struct {
	state    viewState
	allNotes []Note
	filtered []Note

	projects   []string
	projectIdx int

	tags        []string
	tagIdx      int
	selectedTag string

	gridCursor int
	focus      int // 0 = Sidebar, 1 = Grid

	viewport             viewport.Model
	ready                bool
	folder               string
	width                int
	height               int
	nameInput            textinput.Model
	contentInput         textarea.Model
	createFocus          int
	originalEditFilename string
	timeTable            table.Model
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	ti := textinput.New()
	ti.Placeholder = "Note name (e.g. my-note)"
	ti.CharLimit = 156
	ti.Width = 40

	ta := textarea.New()
	ta.Placeholder = "Write your note here..."

	m := model{
		state:        stateFiles,
		folder:       cwd,
		nameInput:    ti,
		contentInput: ta,
		focus:        0, // Start focused on Sidebar
	}
	m.loadAll()

	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) buildTimeTrackingTable() {
	totals := make(map[string]TimeTotal)
	for _, note := range m.allNotes {
		for id, total := range note.TimeTracked {
			aggregate := totals[id]
			aggregate.Hours += total.Hours
			if total.Last.After(aggregate.Last) {
				aggregate.Last = total.Last
			}
			totals[id] = aggregate
		}
	}

	ids := make([]string, 0, len(totals))
	for id := range totals {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if totals[ids[i]].Last.Equal(totals[ids[j]].Last) {
			return ids[i] < ids[j]
		}
		return totals[ids[i]].Last.After(totals[ids[j]].Last)
	})
	rows := make([]table.Row, 0, len(ids))
	for _, id := range ids {
		total := totals[id]
		rows = append(rows, table.Row{
			id,
			fmt.Sprintf("%.2f h", total.Hours),
			total.Last.Format("2006-01-02"),
		})
	}

	styles := table.DefaultStyles()
	styles.Header = styles.Header.Foreground(lipgloss.Color("203")).Bold(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("214")).Bold(true)

	height := max(m.height-4, 1)

	m.timeTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "ID", Width: 24},
			{Title: "TOTAL", Width: 12},
			{Title: "LAST TRACKED", Width: 14},
		}),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
		table.WithWidth(m.width),
		table.WithStyles(styles),
	)
}

func renderPreview(content []byte) string {
	tokens := lexer.New(string(content)).Run()
	var sb strings.Builder

	tagStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("203"))
	projStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("43"))
	dateStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("150"))
	refStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("31"))

	for _, tok := range tokens {
		switch tok.Type {
		case lexer.TokenTag:
			sb.WriteString(tagStyle.Render(strings.TrimPrefix(tok.Value, "=")))
		case lexer.TokenProject:
			sb.WriteString(projStyle.Render(strings.TrimPrefix(tok.Value, "=")))
		case lexer.TokenDate, lexer.TokenDue, lexer.TokenTime:
			sb.WriteString(dateStyle.Render(tok.Value))
		case lexer.TokenBlin:
			sb.WriteString(refStyle.Render(tok.Value))
		case lexer.TokenText:
			sb.WriteString(tok.Value)
		}
	}
	str := strings.TrimSpace(sb.String())

	lines := strings.Split(str, "\n")
	if len(lines) > 5 {
		str = strings.Join(lines[:5], "\n") + "\n..."
	}
	return str
}

func injectEmphasis(content []byte) string {
	tokens := lexer.New(string(content)).Run()
	var sb strings.Builder
	for _, tok := range tokens {
		switch tok.Type {
		case lexer.TokenTag, lexer.TokenProject:
			sb.WriteString("`")
			sb.WriteString(strings.TrimPrefix(tok.Value, "="))
			sb.WriteString("`")
		case lexer.TokenDate, lexer.TokenDue, lexer.TokenTime:
			sb.WriteString("`")
			sb.WriteString(tok.Value)
			sb.WriteString("`")
		case lexer.TokenBlin:
			sb.WriteString("`")
			sb.WriteString(tok.Value)
			sb.WriteString("`")
		case lexer.TokenText:
			sb.WriteString(tok.Value)
		}
	}
	return sb.String()
}

func fixedPreview(preview string) string {
	lines := strings.Split(ansi.Wrap(preview, cardContentWidth, " "), "\n")
	if len(lines) > previewHeight {
		lines = lines[:previewHeight]
		lines[previewHeight-1] = ansi.Truncate(lines[previewHeight-1], cardContentWidth-3, "") + "..."
	}
	return strings.Join(lines, "\n")
}

func formatTimeTracking(entries map[string]TimeTotal) string {
	if len(entries) == 0 {
		return ""
	}
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%s %.2gh", id, entries[id].Hours))
	}
	return strings.Join(parts, ", ")
}

func displayMetadata(value string) string {
	return strings.TrimPrefix(value, "=")
}

func (m *model) loadAll() {
	m.allNotes = []Note{}
	entries, err := os.ReadDir(m.folder)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		path := filepath.Join(m.folder, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		rendered := renderPreview(content)
		tokens := lexer.New(string(content)).Run()

		var projects []string
		var tags []string
		var file_refs []string
		var fileDate time.Time
		var dueDate time.Time
		timeTracked := make(map[string]TimeTotal)
		hasDate := false

		for _, tok := range tokens {
			switch tok.Type {
			case lexer.TokenProject:
				projects = append(projects, tok.Value)
			case lexer.TokenTag:
				tags = append(tags, tok.Value)
			case lexer.TokenTime:
				if date, id, hours, err := lexer.ParseTimeTrackingDate(tok.Value); err == nil {
					total := timeTracked[id]
					total.Hours += hours
					if date.After(total.Last) {
						total.Last = date
					}
					timeTracked[id] = total
				}
			case lexer.TokenDue:
				if due, err := lexer.ParseDueDate(tok.Value); err == nil {
					if dueDate.IsZero() || due.Before(dueDate) {
						dueDate = due
					}
				}
			case lexer.TokenDate:
				if date, err := lexer.ParseDate(tok.Value); err == nil {
					if !hasDate || date.After(fileDate) {
						fileDate = date
						hasDate = true
					}
				}
			case lexer.TokenBlin:
				if file_ref, err := lexer.ParseBlin(tok.Value); err == nil {
					file_refs = append(file_refs, file_ref)
				}
			}
		}

		if !hasDate {
			if info, err := entry.Info(); err == nil {
				fileDate = info.ModTime()
			}
		}

		note := Note{
			Filename:    entry.Name(),
			Preview:     rendered,
			Time:        fileDate,
			Due:         dueDate,
			TimeTracked: timeTracked,
			Projects:    projects,
			Tags:        tags,
			FileRefs:    file_refs,
		}
		m.allNotes = append(m.allNotes, note)
	}

	sort.Slice(m.allNotes, func(i, j int) bool {
		return m.allNotes[i].Time.After(m.allNotes[j].Time)
	})

	m.refreshProjects()
}

func (m *model) refreshProjects() {
	oldProj := ""
	if m.projectIdx >= 0 && m.projectIdx < len(m.projects) {
		oldProj = m.projects[m.projectIdx]
	}

	m.projects = []string{allProjects, dueNotes, timeTrackingNotes}
	projSet := make(map[string]bool)
	for _, n := range m.allNotes {
		for _, p := range n.Projects {
			projSet[p] = true
		}
	}
	var pList []string
	for p := range projSet {
		pList = append(pList, p)
	}
	sort.Strings(pList)
	m.projects = append(m.projects, pList...)

	m.projectIdx = 0
	for i, p := range m.projects {
		if p == oldProj {
			m.projectIdx = i
			break
		}
	}
	m.refreshTags()
}

func (m *model) refreshTags() {
	oldTag := m.selectedTag
	proj := m.projects[m.projectIdx]

	m.tags = []string{"All Tags"}
	tagSet := make(map[string]bool)

	for _, n := range m.allNotes {
		if proj == dueNotes && n.Due.IsZero() {
			continue
		}
		if proj == timeTrackingNotes && len(n.TimeTracked) == 0 {
			continue
		}
		if proj != allProjects && proj != dueNotes && proj != timeTrackingNotes && !slices.Contains(n.Projects, proj) {
			continue
		}
		for _, t := range n.Tags {
			tagSet[t] = true
		}
	}
	var tList []string
	for t := range tagSet {
		tList = append(tList, t)
	}
	sort.Strings(tList)
	m.tags = append(m.tags, tList...)

	m.tagIdx = 0
	m.selectedTag = "All Tags"
	for i, t := range m.tags {
		if t == oldTag {
			m.tagIdx = i
			m.selectedTag = t
			break
		}
	}

	m.refreshGrid()
}

func (m *model) refreshGrid() {
	proj := m.projects[m.projectIdx]
	tag := m.selectedTag

	m.filtered = []Note{}
	for _, n := range m.allNotes {
		if proj == dueNotes && n.Due.IsZero() {
			continue
		}
		if proj == timeTrackingNotes && len(n.TimeTracked) == 0 {
			continue
		}
		if proj != allProjects && proj != dueNotes && proj != timeTrackingNotes && !slices.Contains(n.Projects, proj) {
			continue
		}
		if tag != "All Tags" && !slices.Contains(n.Tags, tag) {
			continue
		}
		m.filtered = append(m.filtered, n)
	}
	if proj == dueNotes {
		sort.Slice(m.filtered, func(i, j int) bool {
			return m.filtered[i].Due.Before(m.filtered[j].Due)
		})
	}

	if m.gridCursor >= len(m.filtered) {
		m.gridCursor = len(m.filtered) - 1
	}
	if m.gridCursor < 0 {
		m.gridCursor = 0
	}
	m.updateViewportContent()
}

func (m *model) viewMarkdown(filename string) {
	path := filepath.Join(m.folder, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		m.viewport.SetContent("Error reading file.")
		return
	}

	emphasized := injectEmphasis(content)
	out, err := glamour.Render(emphasized, "dark")
	if err != nil {
		m.viewport.SetContent("Error rendering markdown.")
		return
	}
	m.viewport.SetContent(out)
	m.viewport.GotoTop()
}

func (m *model) renderSidebar() string {
	w := sidebarWidth

	projColor := lipgloss.Color("203")
	if m.focus != 0 {
		projColor = lipgloss.Color("240")
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(projColor).Width(w).Align(lipgloss.Center)

	arrows := "< %s >"
	if m.focus == 0 {
		arrows = "◀ %s ▶"
	}
	header := headerStyle.Render(fmt.Sprintf(arrows, displayMetadata(m.projects[m.projectIdx])))

	var body []string
	for i, t := range m.tags {
		prefix := "  "
		if m.focus == 0 && i == m.tagIdx {
			prefix = "▶ "
		}

		style := lipgloss.NewStyle().Width(w)
		if i == m.tagIdx {
			if m.focus == 0 {
				style = style.Foreground(lipgloss.Color("214")).Bold(true)
			} else {
				style = style.Foreground(lipgloss.Color("245")) // dimmer selection
			}
		} else {
			style = style.Foreground(lipgloss.Color("240"))
		}
		body = append(body, style.Render(prefix+displayMetadata(t)))
	}

	sidebarStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("237")).
		PaddingRight(1).
		MarginRight(1).
		Width(w)

	content := header + "\n\n" + strings.Join(body, "\n")
	return sidebarStyle.Render(content)
}

func (m *model) generateGrid() string {
	if len(m.filtered) == 0 {
		return lipgloss.NewStyle().PaddingLeft(2).Render("No notes found for this filter.")
	}

	cols := max(m.viewport.Width/(cardWidth+2), 1)

	var rows []string
	var currentRow []string

	for i, file := range m.filtered {
		date := file.Time
		isDueView := m.projects[m.projectIdx] == dueNotes
		if isDueView {
			date = file.Due
		}
		dateStr := date.Format("2006-01-02")
		if date.IsZero() {
			dateStr = ""
		} else if isDueView {
			dateStr = "Due " + dateStr
		}

		dateLine := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(dateStr)

		titleWidth := max(cardContentWidth-lipgloss.Width(dateLine), 5)

		fname := file.Filename
		if len(fname) > titleWidth-4 {
			fname = fname[:titleWidth-5] + "..."
		}

		titleLine := lipgloss.NewStyle().
			Width(titleWidth).
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Render(fname)

		header := lipgloss.JoinHorizontal(lipgloss.Top, titleLine, dateLine)

		divider := lipgloss.NewStyle().
			Foreground(lipgloss.Color("237")).
			Render(strings.Repeat("─", cardContentWidth))

		previewContent := file.Preview
		if tracking := formatTimeTracking(file.TimeTracked); tracking != "" {
			summary := lipgloss.NewStyle().Foreground(lipgloss.Color("150")).Render("Time: " + tracking)
			previewContent = summary + "\n" + previewContent
		}

		// References to other files in blue
		if references := file.FileRefs; len(references) > 0 {
			refs := lipgloss.NewStyle().Foreground(lipgloss.Color("31")).Render("Mentions: " + strings.Join(references, ", "))
			previewContent = refs + "\n" + previewContent
		}

		// Wrap and truncate before rendering so ANSI-styled content cannot expand
		// a card past its fixed height.
		preview := lipgloss.NewStyle().
			Width(cardContentWidth).
			Height(previewHeight).
			MaxHeight(previewHeight).
			Render(fixedPreview(previewContent))
		content := lipgloss.JoinVertical(lipgloss.Left, header, divider, preview)

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Width(cardWidth).
			Padding(0, 1).
			MarginRight(1).
			MarginBottom(0)

		if i == m.gridCursor {
			if m.focus == 1 {
				style = style.BorderForeground(lipgloss.Color("203"))
			} else {
				style = style.BorderForeground(lipgloss.Color("240"))
			}
		} else {
			style = style.BorderForeground(lipgloss.Color("239"))
		}

		currentRow = append(currentRow, style.Render(content))

		if len(currentRow) == cols || i == len(m.filtered)-1 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			currentRow = nil
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *model) syncViewport() {
	if m.state == stateFiles {
		cols := max(m.viewport.Width/(cardWidth+2), 1)
		cursorRow := m.gridCursor / cols

		top := cursorRow * cardHeight
		bottom := top + cardHeight

		if top < m.viewport.YOffset {
			m.viewport.SetYOffset(top)
		} else if bottom > m.viewport.YOffset+m.viewport.Height {
			m.viewport.SetYOffset(bottom - m.viewport.Height)
		}
	}
}

func (m *model) updateViewportContent() {
	if !m.ready {
		return
	}
	if m.state == stateFiles {
		m.viewport.SetContent(m.generateGrid())
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.state == stateCreate || m.state == stateEdit {
		var cmds []tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.state = stateFiles
				m.updateViewportContent()
				return m, nil
			case "tab", "shift+tab":
				if m.createFocus == 0 {
					m.createFocus = 1
					m.nameInput.Blur()
					m.contentInput.Focus()
				} else {
					m.createFocus = 0
					m.contentInput.Blur()
					m.nameInput.Focus()
				}
				return m, nil
			case "enter":
				if m.createFocus == 0 {
					m.createFocus = 1
					m.nameInput.Blur()
					m.contentInput.Focus()
					return m, nil
				}
			case "ctrl+s":
				name := strings.TrimSpace(m.nameInput.Value())
				if name != "" {
					name = strings.ReplaceAll(name, " ", "-")
					if !strings.HasSuffix(name, ".md") {
						name += ".md"
					}

					if m.state == stateEdit && m.originalEditFilename != "" && m.originalEditFilename != name {
						os.Remove(filepath.Join(m.folder, m.originalEditFilename))
					}

					path := filepath.Join(m.folder, name)
					os.WriteFile(path, []byte(m.contentInput.Value()), 0644)

					m.loadAll() // reload all notes and projects
					m.state = stateFiles
					m.updateViewportContent()
				}
				return m, nil
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.contentInput.SetWidth(msg.Width - 4)
			m.contentInput.SetHeight(msg.Height - 12)
		}

		m.nameInput, cmd = m.nameInput.Update(msg)
		cmds = append(cmds, cmd)
		m.contentInput, cmd = m.contentInput.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}

	if m.state == stateTimeTracking {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.state = stateFiles
				m.viewport.Width = m.width - sidebarWidth - 2
				m.updateViewportContent()
				return m, nil
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.buildTimeTrackingTable()
			return m, nil
		}

		m.timeTable, cmd = m.timeTable.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case editorFinishedMsg:
		m.loadAll()
		if len(m.filtered) > 0 {
			m.viewMarkdown(m.filtered[m.gridCursor].Filename)
		}
		return m, tea.ClearScreen
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "c", "C":
			if m.state == stateFiles {
				m.state = stateCreate
				m.nameInput.SetValue("")
				m.contentInput.SetValue("")
				m.createFocus = 0
				m.nameInput.Focus()
				m.contentInput.Blur()
			}
		case "d":
			if m.state == stateFiles {
				for i, project := range m.projects {
					if project == dueNotes {
						m.projectIdx = i
						m.refreshTags()
						break
					}
				}
			}
		case "t":
			if m.state == stateFiles {
				m.state = stateTimeTracking
				m.buildTimeTrackingTable()
			}
		case "tab", "shift+tab":
			if m.state == stateFiles {
				m.focus = 1 - m.focus
				m.updateViewportContent()
			}
		case "esc":
			if m.state == stateMarkdown {
				m.state = stateFiles
				m.viewport.Width = m.width - sidebarWidth - 2 // Restore sidebar split layout
				m.updateViewportContent()
			}
		case "left", "h":
			if m.state == stateFiles {
				if m.focus == 0 {
					if m.projectIdx > 0 {
						m.projectIdx--
						m.refreshTags()
					}
				} else {
					if m.gridCursor > 0 {
						m.gridCursor--
						m.syncViewport()
						m.updateViewportContent()
					}
				}
			}
		case "right", "l":
			if m.state == stateFiles {
				if m.focus == 0 {
					if m.projectIdx < len(m.projects)-1 {
						m.projectIdx++
						m.refreshTags()
					}
				} else {
					if m.gridCursor < len(m.filtered)-1 {
						m.gridCursor++
						m.syncViewport()
						m.updateViewportContent()
					}
				}
			}
		case "up", "k":
			switch m.state {
			case stateFiles:
				if m.focus == 0 {
					if m.tagIdx > 0 {
						m.tagIdx--
						m.selectedTag = m.tags[m.tagIdx]
						m.refreshGrid()
					}
				} else {
					cols := max(m.viewport.Width/40, 1)
					if m.gridCursor >= cols {
						m.gridCursor -= cols
						m.syncViewport()
						m.updateViewportContent()
					}
				}
			case stateMarkdown:
				m.viewport, cmd = m.viewport.Update(msg)
			}
		case "down", "j":
			switch m.state {
			case stateFiles:
				if m.focus == 0 {
					if m.tagIdx < len(m.tags)-1 {
						m.tagIdx++
						m.selectedTag = m.tags[m.tagIdx]
						m.refreshGrid()
					}
				} else {
					cols := max(m.viewport.Width/40, 1)
					if m.gridCursor+cols < len(m.filtered) {
						m.gridCursor += cols
						m.syncViewport()
						m.updateViewportContent()
					}
				}
			case stateMarkdown:
				m.viewport, cmd = m.viewport.Update(msg)
			}
		case "e":
			if m.state == stateMarkdown {
				m.state = stateEdit
				fname := m.filtered[m.gridCursor].Filename
				m.originalEditFilename = fname

				displayname := strings.TrimSuffix(fname, ".md")
				m.nameInput.SetValue(displayname)

				path := filepath.Join(m.folder, fname)
				contentBytes, _ := os.ReadFile(path)
				m.contentInput.SetValue(string(contentBytes))

				m.createFocus = 1
				m.nameInput.Blur()
				m.contentInput.Focus()
				m.contentInput.CursorEnd()
			}
		case "E":
			if m.state == stateMarkdown {
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vim"
				}
				cmd := exec.Command(editor, filepath.Join(m.folder, m.filtered[m.gridCursor].Filename))
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
					return editorFinishedMsg{err}
				})
			}
		case "enter":
			if m.state == stateFiles {
				if m.focus == 0 {
					if m.projects[m.projectIdx] == timeTrackingNotes {
						m.state = stateTimeTracking
						m.buildTimeTrackingTable()
						return m, nil
					}
					// Hop focus to grid on enter
					m.focus = 1
					m.updateViewportContent()
				} else {
					if len(m.filtered) > 0 {
						m.state = stateMarkdown
						m.viewport.Width = m.width // Full width for Markdown read
						m.viewMarkdown(m.filtered[m.gridCursor].Filename)
					}
				}
			}
		default:
			if m.state == stateMarkdown {
				m.viewport, cmd = m.viewport.Update(msg)
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// The view contains a one-line header, one blank separator, and one-line
		// help footer. Giving the rest to the viewport keeps the footer last.
		headerHeight := 3

		vpWidth := m.width
		if m.state == stateFiles {
			vpWidth = m.width - sidebarWidth - 2
		}

		if !m.ready {
			m.viewport = viewport.New(vpWidth, msg.Height-headerHeight)
			m.ready = true
			m.updateViewportContent()
		} else {
			m.viewport.Width = vpWidth
			m.viewport.Height = msg.Height - headerHeight
			m.updateViewportContent()
		}
		m.contentInput.SetWidth(msg.Width - 4)
		m.contentInput.SetHeight(msg.Height - 12)
	}

	return m, cmd
}

func (m *model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	header := ""

	switch m.state {
	case stateFiles:
		header = headerStyle.Render(fmt.Sprintf(
			"Workspace  %s / %s  %d notes",
			displayMetadata(m.projects[m.projectIdx]),
			displayMetadata(m.selectedTag),
			len(m.filtered),
		))
	case stateMarkdown:
		if len(m.filtered) > 0 {
			header = headerStyle.Render("Viewing: " + m.filtered[m.gridCursor].Filename)
		}
	case stateCreate:
		header = headerStyle.Render("Create New Note")
	case stateEdit:
		header = headerStyle.Render("Edit Note")
	case stateTimeTracking:
		header = headerStyle.Render(fmt.Sprintf("Time Tracking  %d IDs", len(m.timeTable.Rows())))
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	help := ""

	if m.state == stateCreate || m.state == stateEdit {
		form := fmt.Sprintf(
			"\n%s\n%s\n\n%s\n%s\n\n%s",
			lipgloss.NewStyle().Bold(true).Render("Name:"),
			m.nameInput.View(),
			lipgloss.NewStyle().Bold(true).Render("Content:"),
			m.contentInput.View(),
			helpStyle.Render("Keys: [ctrl+s] save | [esc] cancel | [tab] switch focus | [ctrl+c] quit"),
		)
		return fmt.Sprintf("%s\n%s", header, form)
	}

	if m.state == stateMarkdown {
		help = helpStyle.Render("Keys: [esc] back to workspace | [e]dit internally | [E]xternal editor | [q]uit | [↑/↓] scroll")
		return fmt.Sprintf("%s\n\n%s\n%s", header, m.viewport.View(), help)
	}

	if m.state == stateTimeTracking {
		help = helpStyle.Render("Keys: [↑/↓] navigate | [esc] back to workspace | [q]uit")
		return fmt.Sprintf("%s\n\n%s\n%s", header, m.timeTable.View(), help)
	}

	// State Files View
	sidebar := m.renderSidebar()
	grid := m.viewport.View()
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, grid)

	help = helpStyle.Render("Keys: [tab] switch pane | [h/l] cycle projects | [j/k] cycle tags/notes | [d]ue | [t]ime | [c]reate | [enter] select | [q]uit")
	return fmt.Sprintf("%s\n\n%s\n%s", header, mainArea, help)
}
