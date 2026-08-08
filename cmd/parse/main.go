package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Get current folder
	curr_folder, err := os.Getwd()
	if err != nil {
		curr_folder = "."
	}

	folder := flag.String("folder", curr_folder, "Folder to parse")
	need_help := flag.Bool("help", false, "Show help")
	flag.Parse()

	// Print usage if help flag is set
	if *need_help {
		flag.Usage()
		return
	}

	fmt.Println("Parsing folder:", *folder)
}
