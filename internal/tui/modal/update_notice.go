package modal

import (
	"fmt"

	"github.com/kopecmaciej/tview"
	"github.com/kopecmaciej/vi-sql/internal/tui/core"
)

const updateNoticeId = "UpdateNotice"

// ShowUpdateNotice displays a modal informing the user that a new version is available.
func ShowUpdateNotice(pages *core.Pages, latestVersion string, onDismiss func()) {
	text := fmt.Sprintf(
		"Version [yellow]%s[-] is available.\n\nVisit [blue]github.com/kopecmaciej/vi-sql[-] to download.",
		latestVersion,
	)

	m := tview.NewModal()
	m.SetTitle(" Update Available ")
	m.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	m.SetText(text)
	m.AddButtons([]string{"Dismiss"})
	m.SetDoneFunc(func(_ int, _ string) {
		pages.RemovePage(updateNoticeId)
		if onDismiss != nil {
			onDismiss()
		}
	})

	pages.AddPage(updateNoticeId, m, true, true)
}
