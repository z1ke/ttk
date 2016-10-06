// Copyright (c) 2016 Company 0, LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ttk

import (
	"fmt"
	"testing"
)

func TestUnescape(t *testing.T) {
	redbold, _ := Color(AttrBold, ColorRed, AttrNA)
	blue, _ := Color(AttrNA, ColorBlue, AttrNA)
	greencyan, _ := Color(AttrNA, ColorGreen, ColorCyan)
	reset, _ := Color(AttrReset, AttrNA, AttrNA)

	redTest := fmt.Sprintf("lalala %vmoo%v test", redbold, reset)
	redU := Unescape(redTest)
	if redU != "lalala moo test" {
		t.Fatalf("red")
	}

	blueTest := fmt.Sprintf("%vlalala moo test%v", blue, reset)
	blueU := Unescape(blueTest)
	if blueU != "lalala moo test" {
		t.Fatalf("blue")
	}

	greencyanTest := fmt.Sprintf("%v%vlalala moo test%v", greencyan, reset, reset)
	greencyanU := Unescape(greencyanTest)
	if greencyanU != "lalala moo test" {
		t.Fatalf("greencyan")
	}
}
