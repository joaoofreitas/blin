package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaoofreitas/blin/internal/blin-lang"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	folder := flag.String("folder", cwd, "Folder to parse")
	needHelp := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *needHelp {
		flag.Usage()
		return
	}

	fmt.Println("Parsing folder:", *folder)

	entries, err := os.ReadDir(*folder)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", *folder, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		path := filepath.Join(*folder, entry.Name())
		fmt.Printf("\n--- Parsing File: %s ---\n", entry.Name())

		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", entry.Name(), err)
			continue
		}

		tokens := lexer.New(string(content)).Run()

		fmt.Printf("Found %d tokens:\n", len(tokens))
		for _, tok := range tokens {
			fmt.Printf("Line %d, Col %d | %-15v | %q\n", tok.Line, tok.Column, tok.Type, tok.Value)
		}
	}
}
