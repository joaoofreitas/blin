package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	lexer "github.com/joaoofreitas/blin/internal/blin-lang"
)

type Note struct {
	Name        string
	Time        time.Time
	Due         time.Time
	TimeTracked map[string]TimeTotal
	Content     []byte
	Tags        []string
	Projects    []string
}

type TimeTotal struct {
	Hours float64
	Last  time.Time
}

const (
	resetColor   = "\033[0m"
	tagColor     = "\033[38;5;203m"
	projectColor = "\033[38;5;43m"
	dateColor    = "\033[38;5;150m"
	headerColor  = "\033[1;38;5;255;48;2;34;139;34m"
	sectionColor = "\033[1;38;5;203m"
	timeColor    = "\033[38;5;214m"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	folder := flag.String("folder", cwd, "Folder containing notes")
	list := flag.Bool("ls", false, "Print selected notes as raw Markdown")
	listTags := flag.Bool("ls-tags", false, "List tags")
	listProjects := flag.Bool("ls-projects", false, "List projects")
	due := flag.Bool("due", false, "List notes with due dates, soonest first")
	timeTracked := flag.Bool("time-tracked", false, "List notes with tracked time")
	filterTag := flag.String("filter-tag", "", "Filter by tag")
	filterProject := flag.String("filter-project", "", "Filter by project")
	view := flag.String("view", "", "Render a note by filename")
	create := flag.String("create", "", "Create a note (extension is optional)")
	content := flag.String("content", "", "Content for -create")
	page := flag.Int("page", 1, "Page number when using -per-page")
	perPage := flag.Int("per-page", 0, "Notes per page; 0 disables pagination")

	flag.Parse()

	switch {
	case *create != "":
		if err := createNote(*folder, *create, *content); err != nil {
			fatal(err)
		}
	case *view != "":
		if err := viewNote(*folder, *view); err != nil {
			fatal(err)
		}
	case *listTags:
		notes, err := loadNotes(*folder)
		if err != nil {
			fatal(err)
		}
		listValues(tagsForNotes(filterNotes(notes, *filterTag, *filterProject)))
	case *listProjects:
		notes, err := loadNotes(*folder)
		if err != nil {
			fatal(err)
		}
		listValues(projectsForNotes(filterNotes(notes, *filterTag, *filterProject)))
	case *due:
		notes, err := loadNotes(*folder)
		if err != nil {
			fatal(err)
		}
		printDueNotes(dueNotes(filterNotes(notes, *filterTag, *filterProject)), *page, *perPage)
	case *timeTracked:
		notes, err := loadNotes(*folder)
		if err != nil {
			fatal(err)
		}
		printTimeTrackingTotals(timeTrackedNotes(filterNotes(notes, *filterTag, *filterProject)))
	case *list:
		notes, err := loadNotes(*folder)
		if err != nil {
			fatal(err)
		}
		printRawNotes(filterNotes(notes, *filterTag, *filterProject), *page, *perPage)
	default:
		notes, err := loadNotes(*folder)
		if err != nil {
			fatal(err)
		}
		printNotes(filterNotes(notes, *filterTag, *filterProject), *page, *perPage)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "blin:", err)
	os.Exit(1)
}

func loadNotes(folder string) ([]Note, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", folder, err)
	}

	var notes []Note
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		path := filepath.Join(folder, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		note := Note{Name: entry.Name(), Content: content}
		note.TimeTracked = make(map[string]TimeTotal)
		tokens := lexer.New(string(content)).Run()
		for _, tok := range tokens {
			switch tok.Type {
			case lexer.TokenTag:
				if !contains(note.Tags, tok.Value) {
					note.Tags = append(note.Tags, tok.Value)
				}
			case lexer.TokenProject:
				if !contains(note.Projects, tok.Value) {
					note.Projects = append(note.Projects, tok.Value)
				}
			case lexer.TokenTime:
				date, id, hours, err := lexer.ParseTimeTrackingDate(tok.Value)
				if err != nil {
					return nil, fmt.Errorf("parse time tracking in %s: %w", path, err)
				}
				total := note.TimeTracked[id]
				total.Hours += hours
				if date.After(total.Last) {
					total.Last = date
				}
				note.TimeTracked[id] = total
			case lexer.TokenDue:
				dueDate, err := lexer.ParseDueDate(tok.Value)
				if err != nil {
					return nil, fmt.Errorf("parse due date in %s: %w", path, err)
				}
				if note.Due.IsZero() || dueDate.Before(note.Due) {
					note.Due = dueDate
				}
			case lexer.TokenDate:
				if date, err := lexer.ParseDate(tok.Value); err == nil && date.After(note.Time) {
					note.Time = date
				}
			}
		}

		if note.Time.IsZero() {
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", path, err)
			}
			note.Time = info.ModTime()
		}
		notes = append(notes, note)
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Time.After(notes[j].Time)
	})
	return notes, nil
}

func filterNotes(notes []Note, tag, project string) []Note {
	var filtered []Note
	for _, note := range notes {
		if tag != "" && !contains(note.Tags, tag) {
			continue
		}
		if project != "" && !contains(note.Projects, project) {
			continue
		}
		filtered = append(filtered, note)
	}
	return filtered
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func tagsForNotes(notes []Note) []string {
	values := make(map[string]bool)
	for _, note := range notes {
		for _, tag := range note.Tags {
			values[tag] = true
		}
	}
	return sortedValues(values)
}

func projectsForNotes(notes []Note) []string {
	values := make(map[string]bool)
	for _, note := range notes {
		for _, project := range note.Projects {
			values[project] = true
		}
	}
	return sortedValues(values)
}

func sortedValues(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func listValues(values []string) {
	for _, value := range values {
		fmt.Println(value)
	}
}

func printNotes(notes []Note, page, perPage int) {
	printPagedNotes(notes, false, page, perPage)
}

func printRawNotes(notes []Note, page, perPage int) {
	printPagedNotes(notes, false, page, perPage)
}

func printDueNotes(notes []Note, page, perPage int) {
	printPagedNotes(notes, true, page, perPage)
}

func printPagedNotes(notes []Note, showDue bool, page, perPage int) {
	pageNotes, currentPage, totalPages, err := paginateNotes(notes, page, perPage)
	if err != nil {
		fatal(err)
	}
	printNoteContents(pageNotes, showDue)
	if perPage > 0 {
		fmt.Printf("\n%sPage %d of %d%s\n", sectionColor, currentPage, totalPages, resetColor)
	}
}

func paginateNotes(notes []Note, page, perPage int) ([]Note, int, int, error) {
	if perPage == 0 {
		return notes, 0, 0, nil
	}
	if perPage < 1 {
		return nil, 0, 0, errors.New("per-page must be greater than zero")
	}
	if page < 1 {
		return nil, 0, 0, errors.New("page must be greater than zero")
	}

	totalPages := (len(notes) + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		return nil, 0, 0, fmt.Errorf("page %d does not exist; only %d page(s) available", page, totalPages)
	}

	start := (page - 1) * perPage
	if start >= len(notes) {
		return nil, page, totalPages, nil
	}
	end := start + perPage
	if end > len(notes) {
		end = len(notes)
	}
	return notes[start:end], page, totalPages, nil
}

func printNoteContents(notes []Note, showDue bool) {
	for index, note := range notes {
		if index > 0 {
			fmt.Println()
		}
		date := note.Time
		label := "DATE"
		if showDue {
			date = note.Due
			label = "DUE"
		}
		tracking := formatTimeTracking(note.TimeTracked)
		fmt.Printf("%s %s  %s %s%s\n", headerColor, note.Name, label, date.Format("2006-01-02"), resetColor)
		if tracking != "" {
			fmt.Printf("%sTIME%s %s\n", timeColor, resetColor, tracking)
		}
		fmt.Print(colorizeMarkdown(note.Content))
		if len(note.Content) == 0 || note.Content[len(note.Content)-1] != '\n' {
			fmt.Println()
		}
	}
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

func dueNotes(notes []Note) []Note {
	var filtered []Note
	for _, note := range notes {
		if !note.Due.IsZero() {
			filtered = append(filtered, note)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Due.Before(filtered[j].Due)
	})
	return filtered
}

func timeTrackedNotes(notes []Note) []Note {
	var filtered []Note
	for _, note := range notes {
		if len(note.TimeTracked) > 0 {
			filtered = append(filtered, note)
		}
	}
	return filtered
}

func printTimeTrackingTotals(notes []Note) {
	totals := make(map[string]TimeTotal)
	for _, note := range notes {
		for id, total := range note.TimeTracked {
			aggregate := totals[id]
			aggregate.Hours += total.Hours
			if total.Last.After(aggregate.Last) {
				aggregate.Last = total.Last
			}
			totals[id] = aggregate
		}
	}

	fmt.Printf("%sID                    TOTAL    LAST TRACKED%s\n", sectionColor, resetColor)
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
	for _, id := range ids {
		total := totals[id]
		fmt.Printf("%-20s  %5.2f h  %s\n", id, total.Hours, total.Last.Format("2006-01-02"))
	}
}

func viewNote(folder, name string) error {
	path, err := notePath(folder, name)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	fmt.Print(colorizeMarkdown(content))
	return nil
}

func colorizeMarkdown(content []byte) string {
	tokens := lexer.New(string(content)).Run()
	var builder strings.Builder

	for _, token := range tokens {
		switch token.Type {
		case lexer.TokenTag:
			builder.WriteString(tagColor)
			builder.WriteString(strings.TrimPrefix(token.Value, "="))
			builder.WriteString(resetColor)
		case lexer.TokenProject:
			builder.WriteString(projectColor)
			builder.WriteString(strings.TrimPrefix(token.Value, "="))
			builder.WriteString(resetColor)
		case lexer.TokenDate, lexer.TokenDue, lexer.TokenTime:
			builder.WriteString(dateColor)
			builder.WriteString(token.Value)
			builder.WriteString(resetColor)
		case lexer.TokenText:
			builder.WriteString(token.Value)
		}
	}
	return builder.String()
}

func createNote(folder, name, content string) error {
	path, err := notePath(folder, name)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return openEditor(path)
}

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	command := exec.Command(editor, path)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w", editor, err)
	}
	return nil
}

func notePath(folder, name string) (string, error) {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".md")
	name = strings.Join(strings.Fields(name), "-")
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", errors.New("note name must be a filename")
	}
	return filepath.Join(folder, name+".md"), nil
}
