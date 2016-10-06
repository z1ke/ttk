// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ttk

import (
	"fmt"
	"strings"

	"github.com/nsf/termbox-go"
)

// WidgetLabel uniquely identifies the label widget.
const (
	WidgetLabel = "label"
)

var (
	_ Widgeter = (*Label)(nil) // ensure interface is satisfied
)

// init registers the Label Widget.
func init() {
	registeredWidgets[WidgetLabel] = NewLabel
}

// Label is a text only widget.  It prints the contents of text onto the
// window.
type Label struct {
	Widget
	trueX int
	trueY int
	text  string
	attr  Attributes

	// status label only
	status     bool // status means fill entire line
	justify    Justify
	visibility Visibility
}

func (l *Label) Visibility(op Visibility) Visibility {
	switch op {
	case VisibilityGet:
		return l.visibility
	case VisibilityShow:
		l.visibility = op
	case VisibilityHide:
		l.visibility = op
	}

	return l.visibility
}

func (l *Label) clear() {
	l.w.printf(l.trueX, l.trueY, defaultAttributes(), strings.Repeat(" ", l.w.x))
}

// Render implements the Render interface.  This is called from queue context
// so be careful to not use blocking calls.
func (l *Label) Render() {
	if l.visibility == VisibilityHide {
		l.clear()
		return
	}

	if l.status == false {
		l.w.printf(l.trueX, l.trueY, l.attr, "%v", l.text)
		return
	}

	text := l.text
	spacing := l.w.x - len([]rune(text)) + EscapedLen(text)
	if spacing < 0 {
		spacing = 0
	}

	left := ""
	right := ""
	switch l.justify {
	case JustifyRight:
		left = strings.Repeat(" ", spacing)
	case JustifyLeft:
		right = strings.Repeat(" ", spacing)
	case JustifyCenter:
		left = strings.Repeat(" ", spacing/2)
		right = strings.Repeat(" ", spacing/2+spacing%2)
	}
	l.w.printf(0, l.trueY, l.attr, "%v%v%v", left, text, right)
}

// KeyHandler implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (l *Label) KeyHandler(ev termbox.Event) bool {
	return false // not handled
}

// CanFocus implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (l *Label) CanFocus() bool {
	return false // can not be focused
}

// Focus implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (l *Label) Focus() {
	// do nothing
}

// NewLabel is the Label initializer.  This call implements the NewWidget
// convention by taking a *Window and and an anchor point to render the widget.
func NewLabel(w *Window, x, y int) (Widgeter, error) {
	return &Label{
		Widget: MakeWidget(w, x, y),
	}, nil
}

// SetAttributes sets the Attributes.  This will not be displayed immediately.
// SetAttributes shall be called from queue context.
func (l *Label) SetAttributes(a Attributes) {
	l.attr = a
}

// SetText sets the label caption.  This will not be displayed immediately.
// SetText shall be called from queue context.
func (l *Label) SetText(format string, args ...interface{}) {
	l.text = fmt.Sprintf(format, args...)
}

// AddLabel is a convenience function to add a new label to a window.  It wraps
// the AddWidget call.  AddLabel must be called from queue.
func (w *Window) AddLabel(x, y int, format string, args ...interface{}) *Label {
	// we can ignore error for builtins
	l, _ := w.AddWidget(WidgetLabel, x, y)
	label := l.(*Label)
	label.Resize()
	label.SetAttributes(defaultAttributes())
	label.SetText(format, args...)

	return label
}

// Justify is used to determine where text is printed on the Status widget
// (special Label)
type Justify int

// Justify rules for Status widget.
const (
	JustifyLeft Justify = iota
	JustifyRight
	JustifyCenter
)

func (l *Label) Resize() {
	l.trueX = l.x
	l.trueY = l.y

	// y<0 means lines from the bottom
	if l.y < 0 {
		l.trueY = l.w.y + l.y
	}
}

// AddStatus is an alternative Label initializer.  A Status is a label that has
// the property that it fills an entire line and is justified.  This call
// implements the NewWidget convention by taking a *Window and and an anchor
// point to render the widget.
func (w *Window) AddStatus(y int, j Justify, format string,
	args ...interface{}) *Label {
	l, _ := w.AddWidget(WidgetLabel, 0, y)
	label := l.(*Label)
	label.Resize()
	label.status = true
	label.justify = j

	// flip attributes
	a := defaultAttributes()
	a2 := Attributes{
		Fg: a.Bg,
		Bg: a.Fg,
	}
	label.SetAttributes(a2)

	// print
	label.SetText(format, args...)

	return label
}
