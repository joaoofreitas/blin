package notes

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Path builds a safe .md path under folder from a note name.
func Path(folder, name string) (string, error) {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".md")
	name = strings.Join(strings.Fields(name), "-")
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", errors.New("note name must be a filename")
	}
	return filepath.Join(folder, name+".md"), nil
}

// Write creates or overwrites a note file and returns its path.
func Write(folder, name, content string) (string, error) {
	path, err := Path(folder, name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// OpenEditor opens path in $EDITOR (defaults to vim).
func OpenEditor(path string) error {
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
