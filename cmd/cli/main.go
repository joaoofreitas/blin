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
	Name     string
	Time     time.Time
	Due      time.Time
	Content  []byte
	Tags     []string
	Projects []string
}

const (
	resetColor   = "\033[0m"
	tagColor     = "\033[38;5;205m"
	projectColor = "\033[38;5;43m"
	dateColor    = "\033[38;5;150m"
	headerColor  = "\033[1;38;5;255;48;5;24m"
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
	filterTag := flag.String("filter-tag", "", "Filter by tag")
	filterProject := flag.String("filter-project", "", "Filter by project")
	view := flag.String("view", "", "Render a note by filename")
	create := flag.String("create", "", "Create a note (extension is optional)")
	content := flag.String("content", "", "Content for -create")

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
		printDueNotes(dueNotes(filterNotes(notes, *filterTag, *filterProject)))
	case *list:
		notes, err := loadNotes(*folder)
		if err != nil {
			fatal(err)
		}
		printRawNotes(filterNotes(notes, *filterTag, *filterProject))
	default:
		notes, err := loadNotes(*folder)
		if err != nil {
			fatal(err)
		}
		printNotes(filterNotes(notes, *filterTag, *filterProject))
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
			case lexer.TokenDate:
				if dueDate, err := lexer.ParseDueDate(tok.Value); err == nil {
					if note.Due.IsZero() || dueDate.Before(note.Due) {
						note.Due = dueDate
					}
				} else if date, err := lexer.ParseDate(tok.Value); err == nil && date.After(note.Time) {
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

func printNotes(notes []Note) {
	printNoteContents(notes, false)
}

func printRawNotes(notes []Note) {
	printNoteContents(notes, false)
}

func printDueNotes(notes []Note) {
	printNoteContents(notes, true)
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
		fmt.Printf("%s %s  %s %s%s\n", headerColor, note.Name, label, date.Format("2006-01-02"), resetColor)
		fmt.Print(colorizeMarkdown(note.Content))
		if len(note.Content) == 0 || note.Content[len(note.Content)-1] != '\n' {
			fmt.Println()
		}
	}
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
			builder.WriteString(token.Value)
			builder.WriteString(resetColor)
		case lexer.TokenProject:
			builder.WriteString(projectColor)
			builder.WriteString(token.Value)
			builder.WriteString(resetColor)
		case lexer.TokenDate:
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
