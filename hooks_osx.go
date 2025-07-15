// +build darwin

package main

import (
	"github.com/chadsec1/decoyim/gui"
	"github.com/coyim/gotk3osx"
)

var hooks = gui.CreateOSX
var extraGraphics interface{} = gotk3osx.Real
