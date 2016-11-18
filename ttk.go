// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ttk

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

const (
	// see http://en.wikipedia.org/wiki/ANSI_escape_code#Colors
	ANSIFg = 30
	ANSIBg = 40

	AttrNA        = -1
	AttrReset     = 0
	AttrBold      = 1
	AttrUnderline = 3
	AttrReverse   = 7

	ColorBlack   = 0
	ColorRed     = 1
	ColorGreen   = 2
	ColorYellow  = 3
	ColorBlue    = 4
	ColorMagenta = 5
	ColorCyan    = 6
	ColorWhite   = 7
)

var (
	ErrNotEscSequence    = errors.New("not an escape sequence")
	ErrInvalidColor      = errors.New("invalid parameters for sequence")
	ErrInvalidAttribute  = errors.New("invalid attribute")
	ErrInvalidForeground = errors.New("invalid foreground")
	ErrInvalidBackground = errors.New("invalid background")
)

// Color creates an ANSI compatible escape sequence that encodes colors and
// attributes.
func Color(at, fg, bg int) (string, error) {
	var a, f, b string

	// can't be all NA
	if at == AttrNA && fg == AttrNA && bg == AttrNA {
		return "", ErrInvalidColor
	}

	switch at {
	case AttrNA:
		break
	case AttrBold, AttrUnderline, AttrReverse, AttrReset:
		a = fmt.Sprintf("%v;", at)
	default:
		return "", ErrInvalidAttribute
	}

	switch {
	case fg == AttrNA:
		break
	case fg >= ColorBlack && fg <= ColorWhite:
		f = fmt.Sprintf("%v;", fg+ANSIFg)
	default:
		return "", ErrInvalidForeground
	}

	switch {
	case bg == AttrNA:
		break
	case bg >= ColorBlack && bg <= ColorWhite:
		b = fmt.Sprintf("%v;", bg+ANSIBg)
	default:
		return "", ErrInvalidBackground
	}

	es := fmt.Sprintf("\x1b[%v%v%v", a, f, b)

	// replace last ; with m
	es = es[:len(es)-1] + "m"

	return es, nil
}

// DecodeColor decodes an ANSI color escape sequence and ignores trailing
// characters.  It returns an Attributs type that can be used directly in
// termbox (note that the termbox colors are off by one).  The skip contains
// the location of the next character that was not consumed by the escape
// sequence.
func DecodeColor(esc string) (*Attributes, int, error) {
	var a Attributes

	if len(esc) < 2 || !strings.HasPrefix(esc, "\x1b[") {
		return nil, 0, ErrNotEscSequence
	}

	// find trailing m
	i := strings.Index(esc[2:], "m")
	if i == -1 {
		return nil, 0, ErrNotEscSequence
	}

	foundM := false
	parameters := strings.Split(esc[2:i+2+1], ";")
	for _, v := range parameters {
		if strings.HasSuffix(v, "m") {
			v = v[:len(v)-1]
			foundM = true
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, 0, err
		}
		switch {
		case n == AttrReset:
			// return defaults
			a.Fg = fg
			a.Bg = bg
		case n == AttrBold:
			a.Fg |= termbox.AttrBold
		case n == AttrUnderline:
			a.Fg |= termbox.AttrUnderline
		case n == AttrReverse:
			a.Fg |= termbox.AttrReverse
		case n >= ColorBlack+ANSIFg && n <= ColorWhite+ANSIFg:
			// note that termbox colors are off by one
			a.Fg |= termbox.Attribute(n - ANSIFg + 1)
		case n >= ColorBlack+ANSIBg && n <= ColorWhite+ANSIBg:
			// note that termbox colors are off by one
			a.Bg |= termbox.Attribute(n - ANSIBg + 1)
		default:
			return nil, 0, ErrNotEscSequence
		}
	}

	if foundM == false {
		return nil, 0, ErrNotEscSequence
	}

	skip := strings.Index(esc, "m")
	if skip == -1 {
		// can't happen
		return nil, 0, ErrNotEscSequence
	}
	skip += 1 // character past m

	return &a, skip, nil
}

// EscapedLen returns total length of all escape sequences in a given string.
func EscapedLen(s string) int {
	if len(s) == 0 {
		return 0
	}

	total := 0
	for i, rw := 0, 0; i < len(s); i += rw {
		v, width := utf8.DecodeRuneInString(s[i:])
		if v == '\x1b' {
			_, skip, err := DecodeColor(s[i:])
			if err == nil {
				rw = skip
				total += skip
				continue

			}
		}
		rw = width
	}

	return total
}

// Unescape returns the unescaped string.
func Unescape(s string) string {
	if len(s) == 0 {
		return ""
	}

	var ret string
	for i, rw := 0, 0; i < len(s); i += rw {
		v, width := utf8.DecodeRuneInString(s[i:])
		if v == '\x1b' {
			_, skip, err := DecodeColor(s[i:])
			if err == nil {
				rw = skip
				continue

			}
		}
		ret += string(v)
		rw = width
	}

	return ret
}

// Cell contains a single screen cell.
// This structure exists in order to mark cells that require rendering.
// This is required in order to only render deltas, this matters over slow
// links.
type Cell struct {
	termbox.Cell      // anon since we are only adding the dirty bit
	dirty        bool // like your mom
}

// Attributes represents attributes which are defined as text color, bold,
// blink etc.
type Attributes struct {
	Fg termbox.Attribute // foreground
	Bg termbox.Attribute // background
}

var (
	// ErrAlreadyInitialized is used on reentrant calls of Init.
	ErrAlreadyInitialized = errors.New("terminal already initialized")

	// terminal
	maxX       int        // max x
	maxY       int        // max y
	termRaw    bool       // true in raw managed window mode
	keyHandler bool       // true if key handler has been launched
	rawMtx     sync.Mutex // required for switching terminal modes

	// all render and termbox access must go through this channel
	work chan func() // render work queue

	// windows
	lastWindowID int             // last used window id
	focus        *Window         // currently focused window
	prevFocus    *Window         // previously focused window
	windows      map[int]*Window // all managed windows
	keyC         chan Key        // key handler channel

	// lookerupper between Windower an *Window
	windower2window map[Windower]*Window

	// defaults
	bg termbox.Attribute // background color
	fg termbox.Attribute // foreground color
)

// init sets up all global variables and prepares ttk for use.
func init() {
	work = make(chan func(), 32)
	keyC = make(chan Key, 1024)
	windows = make(map[int]*Window)
	windower2window = make(map[Windower]*Window)

	// setup render queue
	// we do this song and dance in order to be able to deal with slow
	// connections where rendering could take a long time
	execute := make(chan bool, 1)
	fa := make([]func(), 0, 20)
	mtx := sync.Mutex{}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		wg.Done()
		for {
			select {
			case _, ok := <-execute:
				if !ok {
					return
				}
				for {
					// get work off queue
					mtx.Lock()
					if len(fa) == 0 {
						mtx.Unlock()
						break
					}
					f := fa[0]
					fa[0] = nil // just in case to prevent leak
					fa = fa[1:]
					mtx.Unlock()

					// actually do work
					f()
				}
			}
		}
	}()

	go func() {
		wg.Done()
		for {
			select {
			case f, ok := <-work:
				if !ok {
					return
				}
				// queue work
				mtx.Lock()
				fa = append(fa, f)
				mtx.Unlock()

				// tell executer there is work
				select {
				case execute <- true:
				default:
				}
			}
		}
	}()
	wg.Wait()
}

// initKeyHandler starts the internal key handler.
// Must be called with mutex held and as a go routine.
func initKeyHandler() {
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			e := ev
			Queue(func() {
				var (
					widget Widgeter
					window Windower
				)
				if focus != nil {
					var used bool
					used, window, widget = focus.keyHandler(e)
					if used {
						flush()
						return
					}
				}

				// forward to global application handler
				keyC <- Key{
					Mod:    e.Mod,
					Key:    e.Key,
					Ch:     e.Ch,
					Window: window,
					Widget: widget,
				}
				// XXX this is a terrible workaround!!
				// the app is racing this channel
				// we need to somehow block here before doing
				// anything else
				//time.Sleep(25 * time.Millisecond)
			})

		case termbox.EventResize:
			Queue(func() {
				resizeAndRender(focus)
			})
		case termbox.EventMouse:
		case termbox.EventError:
			return
		}
	}
}

// Init switches the terminal to raw mode and commences managed window mode.
// This function shall be called prior to any ttk calls.
func Init() error {
	rawMtx.Lock()
	defer rawMtx.Unlock()

	if termRaw == true {
		return ErrAlreadyInitialized
	}

	// switch mode
	err := termbox.Init()
	if err != nil {
		return err
	}

	bg = termbox.ColorDefault
	fg = termbox.ColorDefault
	termbox.HideCursor()
	termbox.SetInputMode(termbox.InputAlt) // this may need to become variable
	_ = termbox.Clear(bg, bg)
	maxX, maxY = termbox.Size()
	_ = termbox.Flush()

	// see if we need to launch the key handler
	if keyHandler == false {
		go initKeyHandler()
		keyHandler = true
	}

	termRaw = true // we are now in raw mode

	return nil
}

// Deinit switches the terminal back to cooked mode and it terminates managed
// window mode.  Init must be called again if a switch is required again.
// Deinit shall be called on application exit; failing to do so may leave the
// terminal corrupted.  If that does happen typing "reset" on the shell usually
// fixes this problem.
func Deinit() {
	wait := make(chan interface{})
	Queue(func() {
		termbox.Close()
		focus = nil
		prevFocus = nil
		windows = make(map[int]*Window) // toss all windows

		rawMtx.Lock()
		termRaw = false
		rawMtx.Unlock()

		wait <- true
	})
	<-wait
}

// Queue sends work to the queue and returns almost immediately.
func Queue(f func()) {
	work <- f
}

// KeyChannel returns the the Key channel that can be used in the application
// to handle keystrokes.
func KeyChannel() chan Key {
	// no need to lock since it never changes
	return keyC
}

// NewWindow creates a new window type.
func NewWindow(manager Windower) *Window {
	wc := make(chan *Window)
	Queue(func() {
		w := &Window{
			id:           lastWindowID,
			mgr:          manager,
			x:            maxX,
			y:            maxY,
			focus:        -1, // no widget focused
			backingStore: make([]Cell, maxX*maxY),
			widgets:      make([]Widgeter, 0, 16),
		}
		lastWindowID++
		windows[w.id] = w
		windower2window[manager] = w
		manager.Init(w)
		wc <- w
	})
	return <-wc
}

// ForwardKey must be called from the application to route key strokes to
// windows.  The life cycle of keystrokes is as follows: widgets -> global
// application context -> window.  Care must be taken in the application to not
// rely on keystrokes that widgets may use.
func ForwardKey(k Key) {
	if k.Window == nil {
		return
	}
	k.Window.KeyHandler(windower2window[k.Window], k)
}

// defaultAttributes returns the default attributes.
// defaultAttributes shall be called from queue context.
func defaultAttributes() Attributes {
	return Attributes{
		Fg: fg,
		Bg: bg,
	}
}

// DefaultAttributes returns the default attributes.
// This is a blocking call.
func DefaultAttributes() Attributes {
	c := make(chan Attributes)
	Queue(func() {
		c <- defaultAttributes()
	})
	return <-c
}

// flush copies focused window backing store onto the physical screen.
// flush shall be called from queue context.
func flush() {
	if focus == nil {
		return
	}
	for y := 0; y < focus.y; y++ {
		for x := 0; x < focus.x; x++ {
			c := focus.getCell(x, y)
			if !c.dirty {
				// skip unchanged cells
				continue
			}
			c.dirty = false

			// this shall be the only spot where
			// termbox.SetCell is called!
			termbox.SetCell(x, y, c.Ch, c.Fg, c.Bg)
		}
	}
	_ = termbox.Flush()
}

// Flush copies focused window backing store onto the physical screen.
func Flush() {
	Queue(func() {
		flush()
	})
}

// setCursor sets the cursor at the specified location.  This will not show
// immediately.  setCursor shall be called from queue context.
func setCursor(x, y int) {
	termbox.SetCursor(x, y)
}

// focus on provided window. This will implicitly focus on a window widget
// that can have focus.  Render and flush it onto the terminal.
// focus shall be called from queue context.
func focusWindow(w *Window) {
	if w == nil {
		return
	}
	_, found := windows[w.id]
	if !found {
		return
	}
	if focus == w {
		return
	}
	prevFocus = focus
	focus = w

	resizeAndRender(w)
}

// resizeAndRender resizes a window and renders it.
func resizeAndRender(w *Window) {
	// render window
	if w != nil {
		_ = termbox.Clear(bg, bg)
		maxX, maxY = termbox.Size()

		w.resize(maxX, maxY)
		w.render()

		// display all the things
		flush()
	}
}

// Focus on provided window. This will implicitly focus on a window widget
// that can have focus.  Render and flush it onto the terminal.
func Focus(w *Window) {
	Queue(func() {
		focusWindow(w)
	})
}

// FocusPrevious focus on previous focused window. This will implicitly focus
// on a window widget that can have focus.  Render and flush it onto the
// terminal.
func FocusPrevious() {
	Queue(func() {
		focusWindow(prevFocus)
	})
}

// Panic application but deinit first so that the terminal will not be corrupt.
func Panic(format string, args ...interface{}) {
	termbox.Close()
	msg := fmt.Sprintf(format, args...)
	panic(msg)
}

// Exit application but deinit first so that the terminal will not be corrupt.
func Exit(format string, args ...interface{}) {
	termbox.Close()
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
