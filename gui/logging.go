package gui

import "github.com/chadsec1/decoyim/coylog"

type withLog interface {
	Log() coylog.Logger
}

type hasLog struct {
	log coylog.Logger
}

func (h *hasLog) Log() coylog.Logger {
	return h.log
}

func (u *gtkUI) Log() coylog.Logger {
	return u.hasLog.Log()
}

func (m *accountManager) Log() coylog.Logger {
	return m.log
}

func (a *account) Log() coylog.Logger {
	return a.log
}

func (c *conversationPane) Log() coylog.Logger {
	return c.account.log
}
