package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	lexer "github.com/joaoofreitas/blin/internal/blin-lang"
)

type viewState int

const (
	stateFiles viewState = iota
	stateTags
	stateProjects
	stateMarkdown
)

type FileItem struct {
	Filename string
	Preview  string
	Time     time.Time
}

type model struct {
	state     viewState
	files     []FileItem
	items     []string
	cursor    int
	viewport  viewport.Model
	ready     bool
	folder    string
	filterMsg string
	width     int
	height    int
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	m := model{
		state:  stateFiles,
		folder: cwd,
	}
	m.loadFiles("", "")

	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func getPreview(content []byte) string {
	str := strings.TrimSpace(string(content))
	if len(str) > 250 {
		str = str[:247] + "..."
	}
	lines := strings.Split(str, "\n")
	if len(lines) > 7 {
		str = strings.Join(lines[:7], "\n") + "\n..."
	}
	return str
}

func renderMarkdown(md string, width int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return strings.TrimSpace(out)
}

func getFileDate(entry os.DirEntry, tokens []lexer.Token) time.Time {
	info, err := entry.Info()
	modTime := time.Time{}
	if err == nil {
		modTime = info.ModTime()
	}

	var maxDate time.Time
	hasDate := false

	for _, tok := range tokens {
		if tok.Type == lexer.TokenDate {
			if t, err := lexer.ParseDate(tok.Value); err == nil {
				if !hasDate || t.After(maxDate) {
					maxDate = t
					hasDate = true
				}
			}
		}
	}

	if hasDate {
		return maxDate
	}
	return modTime
}

func (m *model) loadFiles(tagFilter, projectFilter string) {
	m.files = []FileItem{}
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

		raw := getPreview(content)
		rendered := renderMarkdown(raw, 36) // 40 card width - 4 padding/borders
		tokens := lexer.New(string(content)).Run()
		fileDate := getFileDate(entry, tokens)

		if tagFilter == "" && projectFilter == "" {
			m.files = append(m.files, FileItem{
				Filename: entry.Name(),
				Preview:  rendered,
				Time:     fileDate,
			})
			continue
		}

		match := false
		for _, tok := range tokens {
			if tagFilter != "" && tok.Type == lexer.TokenTag && tok.Value == tagFilter {
				match = true
				break
			}
			if projectFilter != "" && tok.Type == lexer.TokenProject && tok.Value == projectFilter {
				match = true
				break
			}
		}
		if match {
			m.files = append(m.files, FileItem{
				Filename: entry.Name(),
				Preview:  rendered,
				Time:     fileDate,
			})
		}
	}

	sort.Slice(m.files, func(i, j int) bool {
		return m.files[i].Time.After(m.files[j].Time)
	})

	if tagFilter != "" {
		m.filterMsg = "Filtered by tag: " + tagFilter
	} else if projectFilter != "" {
		m.filterMsg = "Filtered by project: " + projectFilter
	} else {
		m.filterMsg = ""
	}
	m.cursor = 0
	m.updateViewportContent()
}

func (m *model) loadTags() {
	m.items = []string{}
	tags := make(map[string]bool)
	entries, _ := os.ReadDir(m.folder)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(m.folder, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		tokens := lexer.New(string(content)).Run()
		for _, tok := range tokens {
			if tok.Type == lexer.TokenTag {
				tags[tok.Value] = true
			}
		}
	}
	for tag := range tags {
		m.items = append(m.items, tag)
	}
	m.filterMsg = "Select a tag to filter files"
	m.cursor = 0
	m.updateViewportContent()
}

func (m *model) loadProjects() {
	m.items = []string{}
	projects := make(map[string]bool)
	entries, _ := os.ReadDir(m.folder)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(m.folder, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		tokens := lexer.New(string(content)).Run()
		for _, tok := range tokens {
			if tok.Type == lexer.TokenProject {
				projects[tok.Value] = true
			}
		}
	}
	for proj := range projects {
		m.items = append(m.items, proj)
	}
	m.filterMsg = "Select a project to filter files"
	m.cursor = 0
	m.updateViewportContent()
}

func (m *model) viewMarkdown(filename string) {
	path := filepath.Join(m.folder, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		m.viewport.SetContent("Error reading file.")
		return
	}

	out, err := glamour.Render(string(content), "dark")
	if err != nil {
		m.viewport.SetContent("Error rendering markdown.")
		return
	}
	m.viewport.SetContent(out)
	m.viewport.GotoTop()
}

func (m *model) generateGrid() string {
	if len(m.files) == 0 {
		return normalStyle.Render("No items found.")
	}

	cardWidth := 40
	cols := m.width / (cardWidth + 2)
	if cols < 1 {
		cols = 1
	}

	var rows []string
	var currentRow []string

	for i, file := range m.files {
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render(file.Filename)
		
		// Use a max height for the content, clipping if necessary
		content := lipgloss.JoinVertical(lipgloss.Left, title, "", file.Preview)

		style := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Width(cardWidth).
			Height(14). // Fixed height for mosaic sticky notes
			Padding(0, 1).
			MarginRight(1).
			MarginBottom(1)

		if i == m.cursor {
			style = style.BorderForeground(lipgloss.Color("205"))
		} else {
			style = style.BorderForeground(lipgloss.Color("240"))
		}

		currentRow = append(currentRow, style.Render(content))

		if len(currentRow) == cols || i == len(m.files)-1 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			currentRow = nil
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *model) generateList() string {
	if len(m.items) == 0 {
		return normalStyle.Render("No items found.")
	}
	s := ""
	for i, item := range m.items {
		if m.cursor == i {
			s += selectedStyle.Render("> " + item) + "\n"
		} else {
			s += normalStyle.Render(item) + "\n"
		}
	}
	return s
}

func (m *model) syncViewport() {
	if m.state == stateFiles {
		cardWidth := 40
		cols := m.width / (cardWidth + 2)
		if cols < 1 {
			cols = 1
		}
		cardHeight := 16 // 14 + 2 margins
		cursorRow := m.cursor / cols
		
		top := cursorRow * cardHeight
		bottom := top + cardHeight
		
		if top < m.viewport.YOffset {
			m.viewport.SetYOffset(top)
		} else if bottom > m.viewport.YOffset + m.viewport.Height {
			m.viewport.SetYOffset(bottom - m.viewport.Height)
		}
	} else if m.state == stateTags || m.state == stateProjects {
		if m.cursor < m.viewport.YOffset {
			m.viewport.SetYOffset(m.cursor)
		} else if m.cursor >= m.viewport.YOffset + m.viewport.Height {
			m.viewport.SetYOffset(m.cursor - m.viewport.Height + 1)
		}
	}
}

func (m *model) updateViewportContent() {
	if !m.ready {
		return
	}
	switch m.state {
	case stateFiles:
		m.viewport.SetContent(m.generateGrid())
	case stateTags, stateProjects:
		m.viewport.SetContent(m.generateList())
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.state == stateMarkdown {
				m.state = stateFiles
				m.updateViewportContent()
			} else if m.state != stateFiles {
				m.state = stateFiles
				m.loadFiles("", "")
			} else if m.filterMsg != "" {
				m.loadFiles("", "")
			}
		case "f":
			if m.state != stateMarkdown {
				m.state = stateFiles
				m.loadFiles("", "")
			}
		case "t":
			if m.state != stateMarkdown {
				m.state = stateTags
				m.loadTags()
			}
		case "p":
			if m.state != stateMarkdown {
				m.state = stateProjects
				m.loadProjects()
			}
		case "left", "h":
			if m.state == stateFiles {
				if m.cursor > 0 {
					m.cursor--
					m.syncViewport()
					m.updateViewportContent()
				}
			}
		case "right", "l":
			if m.state == stateFiles {
				if m.cursor < len(m.files)-1 {
					m.cursor++
					m.syncViewport()
					m.updateViewportContent()
				}
			}
		case "up", "k":
			if m.state == stateFiles {
				cols := m.width / 42
				if cols < 1 { cols = 1 }
				if m.cursor >= cols {
					m.cursor -= cols
					m.syncViewport()
					m.updateViewportContent()
				}
			} else if m.state != stateMarkdown {
				if m.cursor > 0 {
					m.cursor--
					m.syncViewport()
					m.updateViewportContent()
				}
			} else {
				m.viewport, cmd = m.viewport.Update(msg)
			}
		case "down", "j":
			if m.state == stateFiles {
				cols := m.width / 42
				if cols < 1 { cols = 1 }
				if m.cursor + cols < len(m.files) {
					m.cursor += cols
					m.syncViewport()
					m.updateViewportContent()
				}
			} else if m.state != stateMarkdown {
				if m.cursor < len(m.items)-1 {
					m.cursor++
					m.syncViewport()
					m.updateViewportContent()
				}
			} else {
				m.viewport, cmd = m.viewport.Update(msg)
			}
		case "enter":
			switch m.state {
			case stateFiles:
				if len(m.files) > 0 {
					m.state = stateMarkdown
					m.viewMarkdown(m.files[m.cursor].Filename)
				}
			case stateTags:
				if len(m.items) > 0 {
					tag := m.items[m.cursor]
					m.state = stateFiles
					m.loadFiles(tag, "")
				}
			case stateProjects:
				if len(m.items) > 0 {
					proj := m.items[m.cursor]
					m.state = stateFiles
					m.loadFiles("", proj)
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
		headerHeight := 4
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight)
			m.ready = true
			m.updateViewportContent()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight
			m.updateViewportContent()
		}
	}

	return m, cmd
}

var (
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).PaddingLeft(2)
	normalStyle   = lipgloss.NewStyle().PaddingLeft(4)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func (m *model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	header := ""
	switch m.state {
	case stateFiles:
		header = titleStyle.Render("📂 Files")
		if m.filterMsg != "" {
			header += " - " + m.filterMsg
		}
	case stateTags:
		header = titleStyle.Render("🏷️  Tags")
		if m.filterMsg != "" {
			header += " - " + m.filterMsg
		}
	case stateProjects:
		header = titleStyle.Render("🚀 Projects")
		if m.filterMsg != "" {
			header += " - " + m.filterMsg
		}
	case stateMarkdown:
		if len(m.files) > 0 {
			header = titleStyle.Render("📄 Viewing: " + m.files[m.cursor].Filename)
		} else {
			header = titleStyle.Render("📄 Viewing")
		}
	}

	help := helpStyle.Render("Keys: [f]iles [t]ags [p]rojects | [enter] select | [esc] back | [q]uit | [h/j/k/l] or arrows navigate")
	if m.state == stateMarkdown {
		help = helpStyle.Render("Keys: [esc] back to list | [q]uit | [↑/↓] scroll")
	}

	return fmt.Sprintf("%s\n\n%s\n%s", header, m.viewport.View(), help)
}
