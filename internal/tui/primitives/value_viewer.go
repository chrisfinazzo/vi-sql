package primitives

import (
	"github.com/gdamore/tcell/v2"
	"github.com/kopecmaciej/tview"
)

// ValueViewer is a full-screen scrollable text viewer for large field values
// (e.g. pretty-printed JSON or XML) opened from the Peeker.
type ValueViewer struct {
	*tview.Box

	textView *tview.TextView
	done     func()
}

func NewValueViewer() *ValueViewer {
	tv := tview.NewTextView()
	tv.SetScrollable(true)
	tv.SetWrap(true)
	tv.SetBorderPadding(0, 0, 1, 1)

	return &ValueViewer{
		Box:      tview.NewBox(),
		textView: tv,
	}
}

// SetContent sets the viewer title and text, and resets scroll to the top.
func (v *ValueViewer) SetContent(title, content string) *ValueViewer {
	v.SetTitle(" " + title + " ")
	v.SetTitleAlign(tview.AlignLeft)
	v.textView.SetText(content)
	v.textView.ScrollToBeginning()
	return v
}

func (v *ValueViewer) SetDoneFunc(done func()) *ValueViewer {
	v.done = done
	return v
}

func (v *ValueViewer) SetBackgroundColor(color tcell.Color) *ValueViewer {
	v.Box.SetBackgroundColor(color)
	v.textView.SetBackgroundColor(color)
	return v
}

func (v *ValueViewer) SetTextColor(color tcell.Color) *ValueViewer {
	v.textView.SetTextColor(color)
	return v
}

// Focus delegates focus to the inner TextView so scrolling works.
func (v *ValueViewer) Focus(delegate func(p tview.Primitive)) {
	delegate(v.textView)
}

func (v *ValueViewer) HasFocus() bool {
	return v.textView.HasFocus()
}

func (v *ValueViewer) Draw(screen tcell.Screen) {
	v.Box.DrawForSubclass(screen, v)
	x, y, w, h := v.GetInnerRect()
	v.textView.SetRect(x, y, w, h)
	v.textView.Draw(screen)
}

func (v *ValueViewer) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return v.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		row, col := v.textView.GetScrollOffset()

		switch event.Key() {
		case tcell.KeyEscape:
			if v.done != nil {
				v.done()
			}
			return
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				if v.done != nil {
					v.done()
				}
				return
			case 'j':
				v.textView.ScrollTo(row+1, col)
				return
			case 'k':
				if row > 0 {
					v.textView.ScrollTo(row-1, col)
				}
				return
			case 'g':
				v.textView.ScrollToBeginning()
				return
			case 'G':
				v.textView.ScrollToEnd()
				return
			}
		}

		// Delegate arrows, PgUp, PgDn to the TextView's own handler.
		if h := v.textView.InputHandler(); h != nil {
			h(event, setFocus)
		}
	})
}
