package gui

import "github.com/chadsec1/decoyim/decoylog"

type withLog interface {
	Log() decoylog.Logger
}

type hasLog struct {
	log decoylog.Logger
}

func (h *hasLog) Log() decoylog.Logger {
	return h.log
}

func (u *gtkUI) Log() decoylog.Logger {
	return u.hasLog.Log()
}

func (m *accountManager) Log() decoylog.Logger {
	return m.log
}

func (a *account) Log() decoylog.Logger {
	return a.log
}

func (c *conversationPane) Log() decoylog.Logger {
	return c.account.log
}
