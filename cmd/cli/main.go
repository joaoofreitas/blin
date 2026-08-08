package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	lexer "github.com/joaoofreitas/blin/internal/blin-lang"
)

type FileEntry struct {
	Name string
	Time time.Time
}

func getFileDate(path string, entry os.DirEntry) time.Time {
	info, err := entry.Info()
	modTime := time.Time{}
	if err == nil {
		modTime = info.ModTime()
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return modTime
	}

	tokens := lexer.New(string(content)).Run()
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

func getSortedFiles(folder string) []FileEntry {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil
	}

	var files []FileEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(folder, entry.Name())
		date := getFileDate(path, entry)
		files = append(files, FileEntry{Name: entry.Name(), Time: date})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Time.After(files[j].Time) // most recent first
	})

	return files
}

func main() {
	needHelp := flag.Bool("help", false, "Show help")
	list := flag.Bool("ls", false, "List markdown files in the current directory")
	listTags := flag.Bool("ls-tags", false, "List all tags found in the markdown files")
	listProjects := flag.Bool("ls-projects", false, "List all projects found in the markdown files")
	filterTag := flag.String("filter-tag", "", "Filter files by tag")
	filterProject := flag.String("filter-project", "", "Filter files by project")

	flag.Parse()
	if *needHelp {
		flag.Usage()
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	if *list {
		listMarkdownFiles(cwd)
	} else if *listTags {
		listTagsInMarkdownFiles(cwd)
	} else if *listProjects {
		listProjectsInMarkdownFiles(cwd)
	} else if *filterTag != "" {
		filterFilesByTag(cwd, *filterTag)
	} else if *filterProject != "" {
		filterFilesByProject(cwd, *filterProject)
	} else {
		flag.Usage()
	}
}

func listMarkdownFiles(folder string) {
	files := getSortedFiles(folder)
	for _, file := range files {
		fmt.Println(file.Name)
	}
}

func listTagsInMarkdownFiles(folder string) {
	tags := make(map[string]bool)
	files := getSortedFiles(folder)

	for _, file := range files {
		path := filepath.Join(folder, file.Name)
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
		fmt.Println(tag)
	}
}

func listProjectsInMarkdownFiles(folder string) {
	projects := make(map[string]bool)
	files := getSortedFiles(folder)

	for _, file := range files {
		path := filepath.Join(folder, file.Name)
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
		fmt.Println(proj)
	}
}

func filterFilesByTag(folder, tag string) {
	files := getSortedFiles(folder)

	for _, file := range files {
		path := filepath.Join(folder, file.Name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		tokens := lexer.New(string(content)).Run()
		hasTag := false
		for _, tok := range tokens {
			if tok.Type == lexer.TokenTag && tok.Value == tag {
				hasTag = true
				break
			}
		}

		if hasTag {
			fmt.Println(file.Name)
		}
	}
}

func filterFilesByProject(folder, project string) {
	files := getSortedFiles(folder)

	for _, file := range files {
		path := filepath.Join(folder, file.Name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		tokens := lexer.New(string(content)).Run()
		hasProject := false
		for _, tok := range tokens {
			if tok.Type == lexer.TokenProject && tok.Value == project {
				hasProject = true
				break
			}
		}

		if hasProject {
			fmt.Println(file.Name)
		}
	}
}
