package modal

import (
	"github.com/gdamore/tcell/v2"
	"github.com/kopecmaciej/tview"
	"github.com/kopecmaciej/vi-sql/internal/tui/core"
	"github.com/rs/zerolog/log"
)

const (
	ErrorModalId = "Error"
)

func NewError(message string, err error) *tview.Modal {
	taggedMessage := "[White::b] " + message + " [::]"

	if err != nil {
		errMsg := err.Error()
		if errMsg != "" {
			if len(errMsg) > 240 {
				errMsg = errMsg[:240] + " ..."
			}
			taggedMessage += "\n" + errMsg
		}
	}

	errModal := tview.NewModal()
	errModal.SetTitle(" Error ")
	errModal.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	errModal.SetTextColor(tcell.ColorRed)
	errModal.SetText(taggedMessage)
	errModal.AddButtons([]string{"Ok"})

	return errModal
}

// ShowError shows a modal with an error message and logs it.
func ShowError(page *core.Pages, message string, err error) {
	log.Error().Err(err).Msg(message)
	errModal := NewError(message, err)

	errModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonLabel == "Ok" {
			page.RemovePage(ErrorModalId)
		}
	})
	page.AddPage(ErrorModalId, errModal, true, true)
}

// ShowErrorAndSetFocus shows an error modal, logs it, and restores focus on dismiss.
func ShowErrorAndSetFocus(page *core.Pages, message string, err error, setFocus func()) {
	log.Error().Err(err).Msg(message)
	errModal := NewError(message, err)
	errModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonLabel == "Ok" {
			page.RemovePage(ErrorModalId)
			setFocus()
		}
	})
	page.AddPage(ErrorModalId, errModal, true, true)
}
