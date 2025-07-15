package gui

import "github.com/chadsec1/decoyim/gui/settings"

type mainSettings struct {
	displaySettings  *displaySettings
	keyboardSettings *keyboardSettings
	settings         *settings.Settings
}
