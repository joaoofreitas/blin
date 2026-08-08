package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lexer "github.com/joaoofreitas/blin/internal/blin-lang"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	needHelp := flag.Bool("help", false, "Show help")
	folder := flag.String("folder", cwd, "Folder to parse")
	list := flag.Bool("ls", false, "List markdown files in the current directory")
	listTags := flag.Bool("ls-tags", false, "List all tags found in the markdown files")
	listProjects := flag.Bool("ls-projects", false, "List all projects found in the markdown files")
	filterTag := flag.String("filter-tag", "", "Filter files by tag")
	filterProject := flag.String("filter-project", "", "Filter files by project")

	flag.Parse()
	if *needHelp {
		flag.Usage()
	}

	if *list {
		listMarkdownFiles(*folder)
	} else if *listTags {
		listTagsInMarkdownFiles(*folder)
	} else if *listProjects {
		listProjectsInMarkdownFiles(*folder)
	} else if *filterTag != "" {
		filterFilesByTag(*folder, *filterTag)
	} else if *filterProject != "" {
		filterFilesByProject(*folder, *filterProject)
	} else {
		flag.Usage()
	}
}

func listMarkdownFiles(folder string) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", folder, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		fmt.Println(entry.Name())
	}
}

func listTagsInMarkdownFiles(folder string) {
	tags := make(map[string]bool)
	entries, err := os.ReadDir(folder)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", folder, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		path := filepath.Join(folder, entry.Name())
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
	entries, err := os.ReadDir(folder)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", folder, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		path := filepath.Join(folder, entry.Name())
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
	entries, err := os.ReadDir(folder)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", folder, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		path := filepath.Join(folder, entry.Name())
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
			fmt.Println(entry.Name())
		}
	}
}

func filterFilesByProject(folder, project string) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", folder, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		path := filepath.Join(folder, entry.Name())
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
			fmt.Println(entry.Name())
		}
	}
}
