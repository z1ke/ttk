// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ttk

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

// WidgetList uniquely identifies the list widget.
const (
	WidgetList = "list"
)

var (
	_ Widgeter = (*List)(nil) // ensure interface is satisfied
)

// init registers the List Widget.
func init() {
	registeredWidgets[WidgetList] = NewList
}

// List is a text only widget.  It prints the contents of text onto the
// window.
type List struct {
	Widget
	width      int
	height     int
	trueW      int
	trueH      int
	trueX      int
	trueY      int
	at         int  // top line being displayed
	paging     bool // paging in progress?
	attr       Attributes
	content    []string
	visibility Visibility
}

// clip renders the list by clipping all lines at widget width.
func (l *List) clip() {
	at := len(l.content) - l.trueH
	if at < 0 {
		at = 0
	}
	if at > len(l.content)-1 {
		return
	}

	line := []rune(l.content[at])
	for i := 0; i < l.trueH; i++ {
		spacing := l.trueW - len(line) + EscapedLen(l.content[at])
		if spacing < 0 {
			// line wrapped
			spacing = 0
			line = line[:l.trueW] // clip
		}
		filler := strings.Repeat(" ", spacing)
		l.w.printf(0, l.trueY+i, l.attr, "%v%v", string(line), filler)

		at++
		if at > len(l.content)-1 {
			return
		}
		line = []rune(l.content[at])
	}
}

func (l *List) Visibility(op Visibility) Visibility {
	switch op {
	case VisibilityGet:
		return l.visibility
	case VisibilityShow:
		l.visibility = op
		l.Render()
	case VisibilityHide:
		l.visibility = op
		l.clear()
	}

	return l.visibility
}

func (l *List) clear() {
	s := strings.Repeat(" ", l.trueW)
	for i := 0; i < l.trueH; i++ {
		l.w.printf(0, i+l.trueY, defaultAttributes(), s)
	}
}

// Render implements the Render interface.  This is called from queue context
// so be careful to not use blocking calls.
func (l *List) Render() {
	if len(l.content) == 0 || l.visibility == VisibilityHide {
		return
	}

	l.Display(Current)
}

// KeyHandler implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (l *List) KeyHandler(ev termbox.Event) bool {
	return false // not handled
}

// CanFocus implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (l *List) CanFocus() bool {
	return false // can not be focused
}

// Focus implements the interface.  This is called from queue context
// so be careful to not use blocking calls.
func (l *List) Focus() {
	// do nothing
}

// NewList is the List initializer.  This call implements the NewWidget
// convention by taking a *Window and and an anchor point to render the widget.
func NewList(w *Window, x, y int) (Widgeter, error) {
	return &List{
		Widget: MakeWidget(w, x, y),
	}, nil
}

// SetAttributes sets the Attributes.  This will not be displayed immediately.
// SetAttributes shall be called from queue context.
func (l *List) SetAttributes(a Attributes) {
	l.attr = a
}

func (l *List) Resize() {
	l.trueX = l.x
	l.trueY = l.y
	l.trueW = l.width
	l.trueH = l.height

	// width < 1 means x - width
	if l.width < 1 {
		l.trueW = l.w.x + l.width
	}

	// height < 1 means y - height
	if l.height < 1 {
		l.trueH = l.w.y + l.height - 1
	}
}

// AddList is a convenience function to add a new list to a window.  It wraps
// the AddWidget call.  AddList must be called from queue.
func (w *Window) AddList(x, y, width, height int) *List {
	// we can ignore error for builtins
	l, _ := w.AddWidget(WidgetList, x, y)
	list := l.(*List)
	list.width = width
	list.height = height
	list.Resize()
	list.SetAttributes(defaultAttributes())

	list.content = make([]string, 0, 1000)
	return list
}

// Append adds a line of text to the list.  Append must be called from queue.
func (l *List) Append(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	l.content = append(l.content, s)

	// adjust at if we are not in a paging operation
	if l.paging {
		return
	}
	l.at = len(l.content) - l.trueH
	if l.at < 0 {
		l.at = 0
	}
}

// Location are hints for the Display function.
type Location int

const (
	Top     Location = iota // Display top line of list
	Bottom                  // Display bottom line of list
	Up                      // Render page up on list
	Down                    // Render page down on list
	Current                 // Render current location on list
)

// Display renders the widget.  This is called from queue context so be careful
// to not use blocking calls.
func (l *List) Display(where Location) {
	if len(l.content) == 0 || l.visibility == VisibilityHide {
		return
	}

	c := l.content

	// XXX this isn't 100% correct since it doesn't handle wrapping lines.
	switch where {
	case Current:
	case Top:
		if len(c) > l.trueH {
			l.at = 0
			l.paging = true
		}
	case Bottom:
		l.at = len(c) - l.trueH
		if l.at < 0 {
			l.at = 0
		}
		l.paging = false

	case Up:
		l.at = l.at - l.trueH + 1
		if l.at < 0 {
			l.at = 0
		}
		l.paging = true

	case Down:
		y := l.at + l.trueH - 1
		if y+l.trueH > len(c) {
			l.Display(Bottom)
			return
		}
		l.at = y
		l.paging = true

	default:
		return
	}

	c = c[l.at : l.at+l.trueH]

	// create a buffer with all lines neatly clipped
	buffer := make([][]rune, 0, l.trueH*2)
	for _, s := range c {
		printWidth := 0
		start := 0
		var lastColor, leftover string
		var cc string // color continuation on next line
		for i := 0; i < len(s); {
			r, width := utf8.DecodeRuneInString(s[i:])
			if r == '\x1b' {
				_, skip, err := DecodeColor(s[i:])
				if err == nil {
					lastColor = s[i : i+skip]
					i += skip
					leftover = s[start:i]
					continue
				}
			}
			i += width
			printWidth++
			if printWidth > l.trueW-1 {
				// clip, so reset start and printWidth
				buffer = append(buffer,
					[]rune(lastColor+s[start:i]))
				start = i
				printWidth = 0
				cc = lastColor
				if start == len(s) {
					// we ended exactly with a color on
					// term boundary, clear out leftover
					// that was set in lastColor check
					leftover = ""
					break
				}
				continue
			} else if i < len(s) {
				// we do this unecessary song and dance to only
				// assign leftover once
				continue
			}
			leftover = s[start:i]
			break // will always break but do it anyway for clarity
		}
		if leftover != "" {
			// done clipping, next line
			filler := strings.Repeat(" ", l.trueW-printWidth)
			buffer = append(buffer, []rune(cc+leftover+filler))
		}
	}

	// now clip buffer to widget l.trueH; we only want to show bottom
	// l.trueH lines
	if len(buffer) > l.trueH {
		buffer = buffer[len(buffer)-l.trueH:]
	}
	for i, v := range buffer {
		l.w.printf(0, l.trueY+i, l.attr, "%v", string(v))
	}
}

// IsPaging indicates if the widget is displaying bottom line.  If the bottom
// line is being displayed it means that the appended text is rendered.
func (l *List) IsPaging() bool {
	return l.paging
}
