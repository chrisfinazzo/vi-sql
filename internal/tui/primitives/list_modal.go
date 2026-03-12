package primitives

import (
	"github.com/gdamore/tcell/v2"
	"github.com/kopecmaciej/tview"
)

// ListModal is a simple list primitive displayed as a centered modal.
type ListModal struct {
	*tview.Box

	list *tview.List
}

func NewListModal() *ListModal {
	return &ListModal{
		Box:  tview.NewBox(),
		list: tview.NewList(),
	}
}

// Draw draws this primitive onto the screen.
func (lm *ListModal) Draw(screen tcell.Screen) {
	screenWidth, screenHeight := screen.Size()

	width, height := screenWidth/2, screenHeight/2

	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2

	lm.SetRect(x, y, width, height)
	lm.Box.DrawForSubclass(screen, lm)

	x, y, width, height = x+1, y+1, width-2, height-2
	lm.list.SetRect(x, y, width, height)
	lm.list.Draw(screen)
}

// GetText returns text of the currently selected item.
func (lm *ListModal) GetText() string {
	selected := lm.list.GetCurrentItem()
	mainText, _ := lm.list.GetItemText(selected)
	return mainText
}

// SetBorderPadding sets the border padding of the list inside the modal.
func (lm *ListModal) SetBorderPadding(top, right, bottom, left int) *ListModal {
	lm.list.SetBorderPadding(top, right, bottom, left)
	return lm
}

// AddItem adds an item to the list.
func (lm *ListModal) AddItem(text string, secondaryText string, shortcut rune, selected func()) *ListModal {
	lm.list.AddItem(text, secondaryText, shortcut, selected)
	return lm
}

// RemoveItem removes an item from the list by index.
func (lm *ListModal) RemoveItem(index int) *ListModal {
	lm.list.RemoveItem(index)
	return lm
}

// GetCurrentItem returns the index of the currently selected item.
func (lm *ListModal) GetCurrentItem() int {
	return lm.list.GetCurrentItem()
}

// Clear removes all items from the list.
func (lm *ListModal) Clear() *ListModal {
	lm.list.Clear()
	return lm
}

// ShowSecondaryText controls whether secondary text is shown.
func (lm *ListModal) ShowSecondaryText(show bool) *ListModal {
	lm.list.ShowSecondaryText(show)
	return lm
}

// SetMainTextStyle sets the style for main text entries.
func (lm *ListModal) SetMainTextStyle(style tcell.Style) *ListModal {
	lm.list.SetMainTextStyle(style)
	return lm
}

// SetSecondaryTextStyle sets the style for secondary text entries.
func (lm *ListModal) SetSecondaryTextStyle(style tcell.Style) *ListModal {
	lm.list.SetSecondaryTextStyle(style)
	return lm
}

// SetSelectedStyle sets the style for the selected item.
func (lm *ListModal) SetSelectedStyle(style tcell.Style) *ListModal {
	lm.list.SetSelectedStyle(style)
	return lm
}

// InputHandler returns the handler for this primitive.
func (lm *ListModal) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return lm.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		lm.list.InputHandler()(event, setFocus)
	})
}
