// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/companyzero/ttk"
	"github.com/gdamore/tcell/termbox"
)

var (
	_ ttk.Windower = (*mainWindow)(nil) // ensure interface is satisfied
)

type mainWindow struct {
	l    *ttk.Label
	e    *ttk.Edit
	e2   *ttk.Edit
	e3   *ttk.Edit
	e4   *ttk.Edit
	t    *ttk.Label
	s    *ttk.Label
	list *ttk.List
}

// called from queue
func (mw *mainWindow) Init(w *ttk.Window) {
	mw.l = w.AddLabel(2, 2, "hello world")
	mw.l.SetAttributes(ttk.Attributes{
		Fg: termbox.ColorYellow,
		Bg: termbox.ColorBlue,
	})

	// edit box
	var s string = "12345"
	mw.e = w.AddEdit(2, 4, -2, &s)

	var s2 string
	mw.e2 = w.AddEdit(4, 5, -4, &s2)

	var s3 string
	mw.e3 = w.AddEdit(3, 6, -8, &s3)

	var s4 string
	mw.e4 = w.AddEdit(0, 8, 0, &s4)

	// list box
	mw.list = w.AddList(10, 10, 0, -2)
	mw.list.Append("this is a list box with some content")

	// title
	mw.t = w.AddStatus(0, ttk.JustifyCenter, "title %v", 12)
	mw.t.SetAttributes(ttk.Attributes{
		Fg: termbox.ColorBlack,
		Bg: termbox.ColorGreen,
	})

	// status
	mw.s = w.AddStatus(-1, ttk.JustifyRight, "status: %v", "OMG")
	mw.s.SetAttributes(ttk.Attributes{
		Fg: termbox.ColorBlack,
		Bg: termbox.ColorYellow,
	})
	ttk.Flush()
}

func (mw *mainWindow) Render(w *ttk.Window) {
}

// called from queue
func (mw *mainWindow) KeyHandler(w *ttk.Window, k ttk.Key) {
}

type secondWindow struct {
	l *ttk.Label
	e *ttk.Edit
}

// called from queue
func (sw *secondWindow) Init(w *ttk.Window) {
	sw.l = w.AddLabel(2, 2, "hello world from #2")
	sw.l.SetAttributes(ttk.Attributes{
		Fg: termbox.ColorRed,
		Bg: termbox.ColorCyan,
	})

	// edit box
	var s string = "abc"
	sw.e = w.AddEdit(2, 14, -2, &s)
	ttk.Flush()
}

func (sw *secondWindow) Render(w *ttk.Window) {
}

// called from queue
func (mw *secondWindow) KeyHandler(w *ttk.Window, k ttk.Key) {
}

func _main() error {
	err := ttk.Init()
	if err != nil {
		return err
	}
	defer ttk.Deinit()

	ww := &secondWindow{}
	sw := ttk.NewWindow(ww)

	w := &mainWindow{}
	mw := ttk.NewWindow(w)

	ttk.Focus(mw)

	for {
		key := <-ttk.KeyChannel()
		switch key.Key {
		case termbox.KeyF1:
			ttk.Focus(mw)
		case termbox.KeyF2:
			ttk.Focus(sw)
		case termbox.KeyCtrlQ:
			return nil
		case termbox.KeyEnter:
			// XXX check if mw is focused
			mw.FocusNext()
		default:
			ttk.ForwardKey(key)
		}
	}
}

func main() {
	err := _main()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return
	}
}
