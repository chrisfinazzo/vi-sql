package primitives

import (
	"github.com/gdamore/tcell/v2"
	"github.com/kopecmaciej/tview"
)

// FormModal is a centered modal window that contains a tview.Form.
type FormModal struct {
	*tview.Box
	Form   *tview.Form
	cancel func()
}

// NewFormModal returns a new FormModal.
func NewFormModal() *FormModal {
	m := &FormModal{
		Box: tview.NewBox(),
	}
	m.Form = tview.NewForm().
		SetButtonsAlign(tview.AlignCenter).
		SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.Form.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(0, 0, 0, 0)

	return m
}

// SetCancelFunc sets a handler called when the user presses Escape.
func (m *FormModal) SetCancelFunc(handler func()) *FormModal {
	m.cancel = handler
	return m
}

// Draw draws the modal centered on the screen.
func (m *FormModal) Draw(screen tcell.Screen) {
	screenWidth, screenHeight := screen.Size()

	width := screenWidth / 2
	height := screenHeight / 2

	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2

	m.SetRect(x, y, width, height)
	m.Box.DrawForSubclass(screen, m)

	m.Form.SetRect(x+1, y+1, width-2, height-2)
	m.Form.Draw(screen)
}

// Focus delegates focus to the inner form.
func (m *FormModal) Focus(delegate func(p tview.Primitive)) {
	delegate(m.Form)
}

// HasFocus reports whether the inner form has focus.
func (m *FormModal) HasFocus() bool {
	return m.Form.HasFocus()
}

// InputHandler handles keyboard input, routing Escape to the cancel func.
func (m *FormModal) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return m.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event.Key() == tcell.KeyEscape {
			if m.cancel != nil {
				m.cancel()
			}
			return
		}
		if handler := m.Form.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}

// MouseHandler routes mouse events to the inner form.
func (m *FormModal) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return m.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		if !m.InRect(event.Position()) {
			return false, nil
		}
		consumed, capture = m.Form.MouseHandler()(action, event, setFocus)
		if consumed {
			setFocus(m)
		}
		return
	})
}
