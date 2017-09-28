// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ttk

import (
	"strings"

	"github.com/nsf/termbox-go"
)

// WidgetEdit uniquely identifies the edit widget.
const (
	WidgetEdit = "edit"
)

var (
	_ Widgeter = (*Edit)(nil) // ensure interface is satisfied
)

// init registers the Edit Widget.
func init() {
	registeredWidgets[WidgetEdit] = NewEdit
}

// Edit is a text entry widget.  It prints the contents of target onto the
// window.  Note: all spaces are trimmed before and after the target string.
type Edit struct {
	Widget
	trueX      int     // actual x coordinate
	trueY      int     // actual y coordinate
	trueW      int     // actual width
	target     *string // result value of action
	display    []rune  // target as runes
	at         int     // start of displayed text
	width      int     // prefered widget width
	cx         int     // current cursor x position
	cy         int     // current cursor y position
	prevX      int     // previous window max x
	prevY      int     // previous window max y
	visibility Visibility
	attr       Attributes
}

func (e *Edit) Visibility(op Visibility) Visibility {
	switch op {
	case VisibilityGet:
		return e.visibility
	case VisibilityShow:
		e.visibility = op
		e.Render()
	case VisibilityHide:
		e.visibility = op
		e.clear()
	}

	return e.visibility
}

func (e *Edit) clear() {
	e.w.printf(e.trueX, e.trueY, defaultAttributes(), strings.Repeat(" ", e.trueW))
}

// Render implements the Render interface.  This is called from queue context
// so be careful to not use blocking calls.
func (e *Edit) Render() {
	if e.visibility == VisibilityHide {
		e.clear()
		return
	}

	filler := ""
	l := e.display[e.at:]
	if len(l) > e.trueW {
		l = e.display[e.at : e.at+e.trueW]
	} else {
		// just erase right hand side
		filler = strings.Repeat(" ", e.trueW-len(l))
	}
	e.w.printf(e.trueX, e.trueY, e.attr, "%v%v", string(l), filler)
}

func insert(slice []rune, index int, value rune) []rune {
	// Grow the slice by one element.
	slice = append(slice, value)
	// Use copy to move the upper part of the slice out of the way and open a hole.
	copy(slice[index+1:], slice[index:])
	// Store the new value.
	slice[index] = value
	// Return the result.
	return slice
}

// KeyHandler implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (e *Edit) KeyHandler(ev termbox.Event) bool {
	var inString int

	switch ev.Key {
	case termbox.KeyCtrlA, termbox.KeyHome:
		e.cx = e.trueX
		e.at = 0
		setCursor(e.cx, e.cy)
		e.Render()
		return true
	case termbox.KeyCtrlE, termbox.KeyEnd:
		if len(e.display) < e.trueW-1 {
			// no need to call display
			e.cx = e.trueX + len(e.display) - e.at
			setCursor(e.cx, e.cy)
			return true
		}
		e.cx = e.trueX + e.trueW - 1
		e.at = len(e.display) - e.trueW + 1
		setCursor(e.cx, e.cy)
		e.Render()
		return true
	case termbox.KeyCtrlU:
		e.cx = e.trueX
		e.at = 0
		e.display = []rune("")
		setCursor(e.cx, e.cy)
		e.Render()
		return true
	case termbox.KeyArrowRight:
		// check to see if we have content on the right hand side
		if e.cx-e.trueX == len(e.display[e.at:]) {
			return true
		}
		e.cx++
		if e.cx > e.trueW+e.trueX-1 {
			e.cx = e.trueW + e.trueX - 1

			// check for end of string before moving at
			if len(e.display[e.at:]) == 0 {
				return true
			}
			e.at++
			e.Render()
			return true
		}
		setCursor(e.cx, e.cy)
		return true
	case termbox.KeyArrowLeft:
		e.cx--
		if e.cx < e.trueX {
			e.cx = e.trueX
			e.at--
			if e.at < 0 {
				e.at = 0
			}
			e.Render()
		}
		setCursor(e.cx, e.cy)
		return true
	case termbox.KeyDelete:
		inString = e.cx - e.trueX + e.at
		if len(e.display) == inString {
			return true
		}
		// remove from slice
		e.display = append(e.display[:inString],
			e.display[inString+1:]...)
		e.Render()
		return true
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		inString = e.cx - e.trueX + e.at
		if inString <= 0 {
			return true
		}
		e.display = append(e.display[:inString-1],
			e.display[inString:]...)

		// cursor left magic
		if e.cx == e.trueX+1 {
			if e.at > e.trueW-1 {
				e.cx = e.trueW - 1
			} else {
				e.cx = e.at + e.trueX
			}
			if e.at >= e.cx {
				e.at -= e.cx
			}
		} else {
			e.cx--
		}
		setCursor(e.cx, e.cy)
		e.Render()
		return true
	case termbox.KeySpace:
		// use space
		ev.Ch = ' '
	case termbox.KeyEnter:
		*e.target = string(e.display)
		// return false and let the application decide if it wants
		// to consume the action
		return false
	}

	// normal runes are displayed and stored
	if ev.Ch != 0 && ev.Mod != 0 && ev.Key == 0 {
		// forward special
		return false
	} else if ev.Ch == 0 {
		return false
	}

	inString = e.cx - e.trueX + e.at
	e.display = insert(e.display, inString, ev.Ch)
	if e.cx < e.trueW+e.trueX-1 {
		e.cx++
		setCursor(e.cx, e.cy)
	} else {
		e.at++
	}

	e.Render()
	return true
}

// CanFocus implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (e *Edit) CanFocus() bool {
	return true // can focus
}

// Focus implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (e *Edit) Focus() {
	if e.cx == -1 || e.cy == -1 {
		// || is deliberate to handle "just in case"
		e.cx = e.trueX
		e.cy = e.trueY
		e.at = 0
	}
	setCursor(e.cx, e.cy)
}

// NewEdit is the Edit initializer.  This call implements the NewWidget
// convention by taking a *Window and and an anchor point to render the widget.
func NewEdit(w *Window, x, y int) (Widgeter, error) {
	return &Edit{
		Widget: MakeWidget(w, x, y),
	}, nil
}

// SetAttributes sets the Attributes.  This will not be displayed immediately.
// SetAttributes shall be called from queue context.
func (e *Edit) SetAttributes(a Attributes) {
	e.attr = a
}

// GetText returns the edit text.
// GetText shall be called from queue context.
func (e *Edit) GetText() string {
	return string(e.display)
}

// SetText sets the edit text.  if end is set to true the cursor and text will
// be set to the end of the string.  This will not be displayed immediately.
// SetText shall be called from queue context.
func (e *Edit) SetText(s *string, end bool) {
	e.target = s
	e.display = []rune(*s)
	e.at = 0

	// send synthesized key to position cursor and text
	ev := termbox.Event{}
	if end {
		ev.Key = termbox.KeyCtrlE
	} else {
		ev.Key = termbox.KeyCtrlA
	}
	e.KeyHandler(ev)
}

func (e *Edit) Resize() {
	inString := e.cx - e.trueX + e.at
	e.trueX = e.x
	e.trueY = e.y
	e.trueW = e.width

	// y<0 is relative to bottom line
	if e.y < 0 {
		e.trueY = e.w.y + e.y + 1
	}

	// e.width <1 means -width from right hand side
	if e.width < 1 {
		e.trueW = e.w.x - e.x + e.width
	}

	// reset cursor and at
	if e.w.y != e.prevY {
		e.cy = e.trueY
		e.prevY = e.w.y
	}
	if e.w.x != e.prevX {
		switch {
		case len(e.display) == inString:
			// end of text
			if len(e.display) < e.trueW-1 {
				e.cx = e.trueX + len(e.display)
				e.at = 0
			} else {
				e.cx = e.trueX + e.trueW - 1
				e.at = len(e.display) - e.trueW + 1
			}
		case inString <= 0:
			// begin of text
			e.at = 0
			e.cx = e.trueX
		default:
			// middle of text
			if e.prevX <= e.w.x {
				// do nothing since x grew
			} else {
				// shift location of at based on shrinkage
				if e.cx >= e.w.x {
					e.cx -= e.prevX - e.w.x
					e.at += e.prevX - e.w.x
				}
			}
		}
		e.prevX = e.w.x
	}
}

// AddEdit is a convenience function to add a new edit to a window.  Capacity
// and width determine the maxima of the returned value.  It wraps the
// AddWidget call.  AddEdit must be called from queue.
func (w *Window) AddEdit(x, y, width int, target *string) *Edit {
	// we can ignore error for builtins
	e, _ := w.AddWidget(WidgetEdit, x, y)
	edit := e.(*Edit)
	edit.width = width

	// save current sizes to detect actual window resizes
	edit.prevX = w.x
	edit.prevY = w.y

	edit.Resize()

	// cursor
	edit.cx = -1
	edit.cy = -1

	// set target string
	edit.SetText(target, true)

	// flip attributes
	a := defaultAttributes()
	a2 := Attributes{
		Fg: a.Bg,
		Bg: a.Fg,
	}
	edit.SetAttributes(a2)

	return edit
}
