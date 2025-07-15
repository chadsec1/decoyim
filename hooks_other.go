// +build !darwin

package main

import (
	"github.com/chadsec1/decoyim/gui"
)

var hooks = noHooks
var extraGraphics interface{} = nil

func noHooks() gui.OSHooks {
	return &gui.NoHooks{}
}
