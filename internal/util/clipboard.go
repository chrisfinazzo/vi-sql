package util

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/rs/zerolog/log"
)

func Copy(text string) {
	if err := clipboard.WriteAll(text); err != nil {
		log.Error().Err(err).Msg("Error writing to clipboard")
	}
}

func Paste() string {
	text, err := clipboard.ReadAll()
	if err != nil {
		log.Error().Err(err).Msg("Error reading from clipboard")
		return ""
	}
	return strings.TrimSpace(text)
}

// GetClipboard returns copy/paste funcs for use with tview's SetClipboard API.
func GetClipboard() (func(string), func() string) {
	return Copy, Paste
}
