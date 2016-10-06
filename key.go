// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ttk

import "github.com/nsf/termbox-go"

// Key contains a key stroke and possible modifiers.
type Key struct {
	Mod    termbox.Modifier // key modifier
	Key    termbox.Key      // special key
	Ch     rune             // normal key
	Window Windower         // window that contains widget
	Widget Widgeter         // widget that emmitted key
}
