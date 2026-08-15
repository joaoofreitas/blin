package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	lexer "github.com/joaoofreitas/blin/internal/blin-lang"
)

// Load reads and parses all markdown notes in folder, newest first.
func Load(folder string) ([]Note, error) {
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

		note, err := Parse(entry.Name(), string(content))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
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

// Parse extracts blin metadata from a note's markdown content.
func Parse(name, content string) (Note, error) {
	note := Note{
		Name:        name,
		Content:     content,
		TimeTracked: make(map[string]TimeTotal),
	}

	for _, tok := range lexer.New(content).Run() {
		switch tok.Type {
		case lexer.TokenTag:
			if !slices.Contains(note.Tags, tok.Value) {
				note.Tags = append(note.Tags, tok.Value)
			}
		case lexer.TokenProject:
			if !slices.Contains(note.Projects, tok.Value) {
				note.Projects = append(note.Projects, tok.Value)
			}
		case lexer.TokenTime:
			date, id, hours, err := lexer.ParseTimeTrackingDate(tok.Value)
			if err != nil {
				return Note{}, fmt.Errorf("parse time tracking: %w", err)
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
				return Note{}, fmt.Errorf("parse due date: %w", err)
			}
			if note.Due.IsZero() || dueDate.Before(note.Due) {
				note.Due = dueDate
			}
		case lexer.TokenBlin:
			fileRef, err := lexer.ParseBlin(tok.Value)
			if err != nil {
				return Note{}, fmt.Errorf("parse blin reference: %w", err)
			}
			if !slices.Contains(note.FileRefs, fileRef) {
				note.FileRefs = append(note.FileRefs, fileRef)
			}
		case lexer.TokenDate:
			if date, err := lexer.ParseDate(tok.Value); err == nil && date.After(note.Time) {
				note.Time = date
			}
		}
	}

	return note, nil
}
