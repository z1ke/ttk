// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ttk

import (
	"fmt"
	"unicode/utf8"

	"github.com/gdamore/tcell/termbox"
)

// Window contains a window context.
type Window struct {
	id           int        // window id
	x            int        // max x
	y            int        // max y
	mgr          Windower   // key handler + renderer
	backingStore []Cell     // output buffer
	widgets      []Widgeter // window widgets
	focus        int        // currently focused widget
}

// Windower interface.  Each window has a Windower interface associated with
// it.  It provides all functions to deal with rendering and user interaction.
type Windower interface {
	Init(w *Window)
	Render(w *Window)
	KeyHandler(*Window, Key)
}

// AddWidget is the generic function to add a widget to a window.  This
// function should only be called by widgets.  Application code, by convention,
// should call the non-generic type asserted call (i.e. AddLabel).
// AddWidget shall be called from queue context.
func (w *Window) AddWidget(id string, x, y int) (Widgeter, error) {
	rw, found := registeredWidgets[id]
	if !found {
		return nil, ErrWidgetNotRegistered
	}
	widget, err := rw(w, x, y)
	if err != nil {
		return nil, err
	}
	w.widgets = append(w.widgets, widget)
	return widget, err
}

// printf prints into the backend buffer.
// This will not show immediately.
// printf shall be called from queue context.
func (w *Window) printf(x, y int, a Attributes, format string,
	args ...interface{}) {
	out := fmt.Sprintf(format, args...)
	xx := 0
	c := Cell{}
	c.Fg = a.Fg
	c.Bg = a.Bg
	mx := w.x - x
	var rw int
	for i := 0; i < len(out); i += rw {
		if x+xx+1 > mx {
			break
		}

		v, width := utf8.DecodeRuneInString(out[i:])
		if v == '\x1b' {
			// see if we understand this escape seqeunce
			cc, skip, err := DecodeColor(out[i:])
			if err == nil {
				c.Fg = cc.Fg
				c.Bg = cc.Bg
				rw = skip
				continue

			}
		}

		rw = width
		c.Ch = v
		w.setCell(x+xx, y, c)
		xx++
	}
}

// setCell sets the content of the window cell at the x and y coordinate.
// setCell shall be called from queue context.
func (w *Window) setCell(x, y int, c Cell) {
	c.dirty = true
	w.backingStore[x+(y*w.x)] = c
}

// getCell returns the content of the window cell at the x and y coordinate.
// getCell shall be called from queue context.
func (w *Window) getCell(x, y int) *Cell {
	c := &w.backingStore[x+(y*w.x)]
	return c
}

// resize sets new x and y maxima.
// resize shall be called from queue context.
func (w *Window) resize(x, y int) {
	w.x = x
	w.y = y
	w.backingStore = make([]Cell, x*y)

	// iterate over widgets
	for _, widget := range w.widgets {
		widget.Resize()
	}
}

// render calls the user provided Render and than renders the widgets in the
// window.
func (w *Window) render() {
	w.mgr.Render(w)

	// iterate over widgets
	for _, widget := range w.widgets {
		widget.Render()
	}

	// focus on a widget
	w.focusWidget()
}

// focusWidget focuses on the current widget.  If focus is -1 it'll focus on
// the first available widget.
// focusWidget shall be called from queue context.
func (w *Window) focusWidget() {
	setCursor(-1, -1) // hide
	if w.focus < 0 {
		for i, widget := range w.widgets {
			if widget.CanFocus() {
				w.focus = i
				widget.Focus()
				return
			}
		}
		// nothing to do
		return
	}

	// make sure we are in bounds
	if w.focus > len(w.widgets) {
		// this really should not happen
		return
	}

	w.widgets[w.focus].Focus()
}

// focusNext focuses on the next available widget.
// focusNext shall be called from queue context.
func (w *Window) focusNext() {
	if w.focus < 0 {
		w.focusWidget()
		return
	}

	// find next widget
	for i, widget := range w.widgets[w.focus+1:] {
		if !widget.CanFocus() {
			continue
		}

		setCursor(-1, -1) // hide
		w.focus = i + w.focus + 1
		widget.Focus()
		return
	}

	// if we get here there was nothing to focus on so focus on first widget
	w.focus = -1
	w.focusWidget()
}

// FocusNext focuses on the next available widget.
func (w *Window) FocusNext() {
	Queue(func() {
		w.focusNext()
		flush()
	})
}

// focusPrevious focuses on the previous available widget.
// focusPrevious shall be called from queue context.
func (w *Window) focusPrevious() {
	// it is ok to be negative since that'll focus on the first widget
	w.focus--
	if w.focus < 0 {
		w.focusWidget()
		return
	}

	// find previous widget
	for i := w.focus; i > 0; i-- {
		widget := w.widgets[i]
		if !widget.CanFocus() {
			continue
		}
		setCursor(-1, -1) // hide
		w.focus = i
		widget.Focus()
		return
	}

	// if we get here we need to focus on last focusable widget
	for i := len(w.widgets) - 1; i > w.focus; i-- {
		widget := w.widgets[i]
		if !widget.CanFocus() {
			continue
		}
		setCursor(-1, -1) // hide
		w.focus = i
		widget.Focus()
		return
	}

	// if we get here it means we found nothing usable and give up
}

// FocusPrevious focuses on the next available widget.
func (w *Window) FocusPrevious() {
	Queue(func() {
		w.focusPrevious()
		flush()
	})
}

// keyHandler routes event to proper widget.  This is called from queue context
// so be careful to not use blocking calls.
func (w *Window) keyHandler(ev termbox.Event) (bool, Windower, Widgeter) {
	if w.focus < 0 || w.focus > len(w.widgets) {
		return false, w.mgr, nil // not used
	}
	return w.widgets[w.focus].KeyHandler(ev), w.mgr, w.widgets[w.focus]
}
