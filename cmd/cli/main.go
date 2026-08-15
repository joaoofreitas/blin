package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	lexer "github.com/joaoofreitas/blin/internal/blin-lang"
	"github.com/joaoofreitas/blin/internal/notes"
)

const (
	resetColor   = "\033[0m"
	tagColor     = "\033[38;5;203m"
	projectColor = "\033[38;5;43m"
	dateColor    = "\033[38;5;150m"
	headerColor  = "\033[1;38;5;255;48;2;34;139;34m"
	sectionColor = "\033[1;38;5;203m"
	timeColor    = "\033[38;5;214m"
	refColor     = "\033[38;5;75m"
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
		path, err := notes.Write(*folder, *create, *content)
		if err != nil {
			fatal(err)
		}
		if err := notes.OpenEditor(path); err != nil {
			fatal(err)
		}
	case *view != "":
		if err := viewNote(*folder, *view); err != nil {
			fatal(err)
		}
	default:
		all, err := notes.Load(*folder)
		if err != nil {
			fatal(err)
		}
		filtered := notes.Filter(all, *filterTag, *filterProject)

		switch {
		case *listTags:
			printLines(notes.Tags(filtered))
		case *listProjects:
			printLines(notes.Projects(filtered))
		case *due:
			printPagedNotes(notes.WithDue(filtered), true, *page, *perPage)
		case *timeTracked:
			printTimeTrackingTotals(notes.WithTimeTracked(filtered))
		case *list:
			printPagedNotes(filtered, false, *page, *perPage)
		default:
			printPagedNotes(filtered, false, *page, *perPage)
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "blin:", err)
	os.Exit(1)
}

func printLines(values []string) {
	for _, value := range values {
		fmt.Println(value)
	}
}

func printPagedNotes(ns []notes.Note, showDue bool, page, perPage int) {
	pageNotes, currentPage, totalPages, err := notes.Page(ns, page, perPage)
	if err != nil {
		fatal(err)
	}
	printNoteContents(pageNotes, showDue)
	if perPage > 0 {
		fmt.Printf("\n%sPage %d of %d%s\n", sectionColor, currentPage, totalPages, resetColor)
	}
}

func printNoteContents(ns []notes.Note, showDue bool) {
	for index, note := range ns {
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

		if len(note.FileRefs) > 0 {
			fmt.Printf("%sMentions:%s %s\n", refColor, resetColor, strings.Join(note.FileRefs, ", "))
		}
		if tracking := notes.FormatTimeTracking(note.TimeTracked); tracking != "" {
			fmt.Printf("%sTime Track%s %s\n", timeColor, resetColor, tracking)
		}

		fmt.Print(colorizeMarkdown(note.Content))
		if len(note.Content) == 0 || note.Content[len(note.Content)-1] != '\n' {
			fmt.Println()
		}
	}
}

func printTimeTrackingTotals(ns []notes.Note) {
	fmt.Printf("%sNAME                    TOTAL    LAST TRACKED%s\n", sectionColor, resetColor)
	for _, row := range notes.AggregateTimeTotals(ns) {
		fmt.Printf("%-20s  %5.2f h  %s\n", row.ID, row.Total.Hours, row.Total.Last.Format("2006-01-02"))
	}
}

func viewNote(folder, name string) error {
	path, err := notes.Path(folder, name)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	fmt.Print(colorizeMarkdown(string(content)))
	return nil
}

func colorizeMarkdown(content string) string {
	var builder strings.Builder
	for _, token := range lexer.New(content).Run() {
		switch token.Type {
		case lexer.TokenTag:
			builder.WriteString(tagColor)
			builder.WriteString(notes.DisplayMetadata(token.Value))
			builder.WriteString(resetColor)
		case lexer.TokenProject:
			builder.WriteString(projectColor)
			builder.WriteString(notes.DisplayMetadata(token.Value))
			builder.WriteString(resetColor)
		case lexer.TokenDate, lexer.TokenDue, lexer.TokenTime:
			builder.WriteString(dateColor)
			builder.WriteString(token.Value)
			builder.WriteString(resetColor)
		case lexer.TokenBlin:
			builder.WriteString(refColor)
			builder.WriteString(strings.TrimPrefix(token.Value, "=blin:"))
			builder.WriteString(resetColor)
		case lexer.TokenText:
			builder.WriteString(token.Value)
		}
	}
	return builder.String()
}
