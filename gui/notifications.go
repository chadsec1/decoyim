package gui

import (
	"time"

	"github.com/coyim/coyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gtki"
)

const mergeNotificationsThreshold = 7

func (u *gtkUI) lastActionTimeFor(f string) time.Time {
	return u.actionTimes[f]
}

func (u *gtkUI) registerLastActionTimeFor(f string, t time.Time) {
	u.actionTimes[f] = t
}

func (u *gtkUI) maybeNotify(timestamp time.Time, account *account, peer jid.WithoutResource, message string) {
	if u.deNotify == nil {
		return
	}

	dname := u.am.displayNameFor(account, peer)

	if timestamp.Before(u.lastActionTimeFor(peer.String()).Add(time.Duration(mergeNotificationsThreshold) * time.Second)) {
		u.hasLog.log.Debug("Decided to not show notification, since the time is not ready")
		return
	}

	u.registerLastActionTimeFor(peer.String(), timestamp)

	err := u.deNotify.show(peer.String(), dname, message)
	if err != nil {
		u.hasLog.log.WithError(err).Warn("Error when showing notification")
	}
}

func (u *gtkUI) showConnectAccountNotification(account *account) func() {
	var notification gtki.InfoBar

	doInUIThread(func() {
		notification = account.buildConnectionNotification()
		account.setCurrentNotification(notification, u.mainUI.notificationArea)
	})

	return func() {
		doInUIThread(func() {
			account.removeCurrentNotificationIf(notification)
		})
	}
}

func (u *gtkUI) notifyTorIsNotRunning(account *account, moreInfo func()) {
	doInUIThread(func() {
		notification := account.buildTorNotRunningNotification(moreInfo)
		account.setCurrentNotification(notification, u.mainUI.notificationArea)
	})
}

func (u *gtkUI) notifyConnectionFailure(account *account, moreInfo func()) {
	doInUIThread(func() {
		notification := account.buildConnectionFailureNotification(moreInfo)
		account.setCurrentNotification(notification, u.mainUI.notificationArea)
	})
}

func (u *gtkUI) notify(title, message string) {
	builder := newBuilder("SimpleNotification")
	obj := builder.getObj("dialog")
	dlg := obj.(gtki.MessageDialog)

	_ = dlg.SetProperty("title", title)
	_ = dlg.SetProperty("text", message)
	dlg.SetTransientFor(u.mainUI.window)

	doInUIThread(func() {
		dlg.Run()
		dlg.Destroy()
	})
}
