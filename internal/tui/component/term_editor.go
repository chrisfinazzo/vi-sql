package component

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kopecmaciej/vi-sql/internal/tui/core"
)

// TermEditor opens a temp .sql file in the user's preferred editor and returns
// the contents when the editor exits. It can be embedded in any component that
// needs external editor support.
type TermEditor struct {
	*core.BaseElement
}

func NewTermEditor() *TermEditor {
	return &TermEditor{
		BaseElement: core.NewBaseElement(),
	}
}

// Open suspends the TUI, opens the editor with optional initial SQL, and
// returns the trimmed file contents after the editor exits.
func (e *TermEditor) Open(initial string) (string, error) {
	tmpFile, err := os.CreateTemp("", "vi-sql-*.sql")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if initial != "" {
		if _, err := tmpFile.WriteString(initial); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			return "", fmt.Errorf("failed to write initial SQL: %w", err)
		}
	}
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	editorCmd := e.resolveEditor()
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("editor command is empty")
	}

	bin, err := exec.LookPath(parts[0])
	if err != nil {
		return "", fmt.Errorf("editor %q not found in PATH: %w", parts[0], err)
	}
	args := append(parts[1:], tmpPath)

	var result string
	var editorErr error

	e.App.Suspend(func() {
		cmd := exec.Command(bin, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if editorErr = cmd.Run(); editorErr != nil {
			return
		}
		data, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			editorErr = readErr
			return
		}
		result = strings.TrimSpace(string(data))
	})

	return result, editorErr
}

// resolveEditor returns the editor command in priority order:
// config > $EDITOR > vi.
func (e *TermEditor) resolveEditor() string {
	if cmd, err := e.App.GetConfig().GetEditorCmd(); err == nil && cmd != "" {
		return cmd
	}
	if env := os.Getenv("EDITOR"); env != "" {
		return env
	}
	return "vi"
}
