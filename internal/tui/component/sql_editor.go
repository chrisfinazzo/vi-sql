package component

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kopecmaciej/vi-sql/internal/tui/core"
)

// SQLEditor opens a temp .sql file in the user's preferred editor and returns
// the contents when the editor exits. It can be embedded in any component that
// needs external editor support.
type SQLEditor struct {
	*core.BaseElement
}

func NewSQLEditor() *SQLEditor {
	return &SQLEditor{
		BaseElement: core.NewBaseElement(),
	}
}

// Open suspends the TUI, opens the editor with optional initial SQL, and
// returns the trimmed file contents after the editor exits.
func (e *SQLEditor) Open(initial string) (string, error) {
	tmpFile, err := os.CreateTemp("", "vi-sql-*.sql")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if initial != "" {
		tmpFile.WriteString(initial)
	}
	tmpFile.Close()
	defer os.Remove(tmpPath)

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
func (e *SQLEditor) resolveEditor() string {
	if cmd, err := e.App.GetConfig().GetEditorCmd(); err == nil && cmd != "" {
		return cmd
	}
	if env := os.Getenv("EDITOR"); env != "" {
		return env
	}
	return "vi"
}
