// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ttk

import (
	"errors"

	"github.com/nsf/termbox-go"
)

// Widget is the base structure of all widgets.
type Widget struct {
	w *Window
	x int
	y int
}

var (
	// ErrWidgetNotRegistered is generated when a NewWidget call is made
	// and the widget was not registered.  Widgets must be registered so
	// that applications may define their own and fully participate in ttk
	// rules.
	ErrWidgetNotRegistered = errors.New("widget not registered")

	// registeredWidgets contains the registered widgets types.
	registeredWidgets = make(map[string]func(*Window, int, int) (Widgeter,
		error))
)

type Visibility int

const (
	VisibilityGet Visibility = iota
	VisibilityHide
	VisibilityShow
)

// Widgeter is the generic Widget interface.  All widgets shall conform to it.
// Since the Widgeter functions are called from queue context the Widget must
// take care to not call blocking queue context calls.
type Widgeter interface {
	CanFocus() bool                   // return true if widget can focus
	Focus()                           // Focus on widget
	Render()                          // Render the widget
	Resize()                          // Resize the widget
	KeyHandler(termbox.Event) bool    // handle key strokes
	Visibility(Visibility) Visibility // show/hide widget
}

// MakeWidget creates a generic Widget structure.
func MakeWidget(w *Window, x, y int) Widget {
	return Widget{
		w: w,
		x: x,
		y: y,
	}
}
