package component

import (
	"fmt"
	"math/rand"

	"github.com/kopecmaciej/vi-sql/internal/config"
	"github.com/kopecmaciej/vi-sql/internal/manager"
	"github.com/kopecmaciej/vi-sql/internal/tui/core"
)

const HintsId = "Hints"

type Hints struct {
	*core.BaseElement
	*core.TextView
}

func NewHints() *Hints {
	h := &Hints{
		BaseElement: core.NewBaseElement(),
		TextView:    core.NewTextView(),
	}
	h.SetIdentifier(HintsId)
	h.SetAfterInitFunc(h.init)
	return h
}

func (h *Hints) init() error {
	h.TextView.SetDynamicColors(true)
	h.TextView.SetBorderPadding(0, 0, 1, 1)
	h.setStyle()
	h.handleEvents()
	return nil
}

func (h *Hints) setStyle() {
	h.SetStyle(h.App.GetStyles())
}

func (h *Hints) handleEvents() {
	go h.HandleEvents(HintsId, func(event manager.EventMsg) {
		switch event.Message.Type {
		case manager.StyleChanged:
			h.setStyle()
		}
	})
}

func (h *Hints) Render() {
	idx := rand.Intn(len(appHints))
	text := appHints[idx](h.App.GetKeys())
	h.TextView.SetTextColor(h.App.GetStyles().Global.SecondaryTextColor.Color())
	if h.App.GetConfig().Styles.BetterSymbols {
		h.TextView.SetText("💡 " + text)
	} else {
		h.TextView.SetText(" [::d]Hint:[-:-:-] " + text)
	}
}

var appHints = []func(k *config.KeyBindings) string{
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("Press [::b]%s[-:-:-] to open the actions panel — the fastest way to reach any major operation.", k.Main.OpenActions.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("Use [::b]%s[-:-:-] to filter the current view without leaving the keyboard.", k.Common.Filter.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("Press [::b]%s[-:-:-] on a selected row to peek at its full content in a side panel.", k.Data.PeekRow.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("[::b]%s[-:-:-] opens a new query tab so you can run SQL while keeping the current table visible.", k.Main.NewTab.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("Press [::b]%s[-:-:-] to hide the schema tree and give more space to the data view.", k.Main.HideSchema.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("Use [::b]%s[-:-:-] to sort by the column under the cursor without opening a sort dialog.", k.Data.SortByColumn.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("[::b]%s[-:-:-] copies the current row to the clipboard as a ready-to-paste INSERT statement.", k.Data.CopyRow.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("Press [::b]%s[-:-:-] on any key in this page to rebind it to your own preferred combination.", k.Common.Edit.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("Use [::b]%s[-:-:-] / [::b]%s[-:-:-] to page through large result sets.", k.Data.PreviousPage.String(), k.Data.NextPage.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("[::b]%s[-:-:-] opens the SQL query editor history so you can re-run or edit past queries.", k.SQLQueryEditor.OpenHistory.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("Press [::b]%s[-:-:-] to explain the current query and inspect its execution plan.", k.Data.ExplainQuery.String())
	},
	func(k *config.KeyBindings) string {
		return fmt.Sprintf("[::b]%s[-:-:-] switches to a different style theme — try it to find one that suits your terminal.", k.Global.ChangeStyle.String())
	},
}
