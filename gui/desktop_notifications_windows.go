package gui

import (
	"os/exec"
	"syscall"

	"github.com/chadsec1/decoyim/ui"
)

const notificationFeaturesSupported = notificationStyles

type desktopNotifications struct {
	notificationStyle   string
	notificationUrgent  bool
	notificationExpires bool
}

func newDesktopNotifications() *desktopNotifications {
	return createDesktopNotifications()
}

func (dn *desktopNotifications) show(jid, from, message string) error {
	from = ui.EscapeAllHTMLTags(string(ui.StripSomeHTML([]byte(from))))
	summary, body := dn.format(from, message, false)

	notification := Notification{
		Title:   "CoyIM",
		Message: summary + body,
		Icon:    coyimIcon.getPath(),
	}
	return notification.Popup()
}

// Notification contains information for popping up a notification on Windows
type Notification struct {
	Title   string
	Message string
	Icon    string
}

// Popup will actually pop up the notification
func (n *Notification) Popup() error {
	cmd := exec.Command("toast.exe", "-t", n.Title, "-m", n.Message, "-p", n.Icon)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
