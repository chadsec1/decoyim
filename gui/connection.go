package gui

import (
	"math/rand"
	"time"

	"github.com/coyim/coyim/config"
	"github.com/coyim/coyim/i18n"
	"github.com/coyim/coyim/xmpp/errors"
)

func (u *gtkUI) connectAccount(account *account) {
	p := account.session.GetConfig().Password
	if p == "" {
		u.askForPasswordAndConnect(account, false)
	} else {
		go func() {
			_ = u.connectWithPassword(account, p)
		}()
	}
}

func (u *gtkUI) torIsNotRunning() {
	u.installTor()
}

func (u *gtkUI) connectionFailureMoreInfoConnectionLost() {
	u.notify(i18n.Local("Connection lost"), i18n.Local("We lost connection to the server for unknown reasons.\n\nWe will try to reconnect."))
}

func (u *gtkUI) connectionFailureMoreInfoTCPBindingFailed() {
	u.notify(i18n.Local("Connection failure"), i18n.Local("We couldn't connect to the server because we couldn't determine a server address for the given domain.\n\nWe will try to reconnect."))
}

func (u *gtkUI) connectionFailureMoreInfoConnectionFailedGeneric() {
	u.notify(i18n.Local("Connection failure"), i18n.Local("We couldn't connect to the server for some reason - verify that the server address is correct and that you are actually connected to the internet.\n\nWe will try to reconnect."))
}

func (u *gtkUI) connectionFailureMoreInfoConnectionFailed(ee error) func() {
	return func() {
		u.notify(i18n.Local("Connection failure"),
			i18n.Localf("We couldn't connect to the server - verify that the server address is correct and that you are actually connected to the internet.\n\nThis is the error we got: %s\n\nWe will try to reconnect.", ee.Error()))
	}
}

func (u *gtkUI) connectWithPassword(account *account, password string) error {
	if !account.session.IsDisconnected() {
		return nil
	}

	removeNotification := u.showConnectAccountNotification(account)
	defer removeNotification()

	err := account.session.Connect(password, u.verifierFor(account))
	switch err {
	case config.ErrTorNotRunning:
		u.notifyTorIsNotRunning(account, u.torIsNotRunning)
	case errors.ErrTCPBindingFailed:
		u.notifyConnectionFailure(account, u.connectionFailureMoreInfoTCPBindingFailed)
	case errors.ErrAuthenticationFailed:
		account.cachedPassword = ""
		u.askForPasswordAndConnect(account, false)
	case errors.ErrGoogleAuthenticationFailed:
		account.cachedPassword = ""
		u.askForPasswordAndConnect(account, true)
	case errors.ErrConnectionFailed:
		u.notifyConnectionFailure(account, u.connectionFailureMoreInfoConnectionFailedGeneric)
	default:
		ff, ok := err.(*errors.ErrFailedToConnect)
		if ok {
			u.notifyConnectionFailure(account, u.connectionFailureMoreInfoConnectionFailed(ff))
		}
	}

	return err
}

func (u *gtkUI) askForPasswordAndConnect(account *account, addGoogleWarning bool) {
	if !account.IsAskingForPassword() {
		accountName := account.Account()
		if account.cachedPassword != "" {
			_ = u.connectWithPassword(account, account.cachedPassword)
			return
		}

		doInUIThread(func() {
			account.AskForPassword()
			u.askForPassword(accountName, addGoogleWarning,
				func() {
					account.session.SetWantToBeOnline(false)
					account.AskedForPassword()
				},
				func(password string) error {
					account.cachedPassword = password
					account.AskedForPassword()
					return u.connectWithPassword(account, password)
				},
				func(password string) {
					account.session.GetConfig().Password = password
					u.SaveConfig()
				},
			)
		})
	}
}

func (u *gtkUI) connectWithRandomDelay(a *account) {
	sleepDelay := time.Duration(rand.Int31n(7643)) * time.Millisecond
	time.Sleep(sleepDelay)
	a.Connect()
}

func (u *gtkUI) connectAllAutomatics(all bool) {
	allAccounts := u.am.getAllAccounts()
	acc := make([]*account, 0, len(allAccounts))

	for _, a := range allAccounts {
		if (all || a.session.GetConfig().ConnectAutomatically) && a.session.IsDisconnected() {
			acc = append(acc, a)
		}
	}

	for _, a := range acc {
		go func(ca *account) {
			u.connectWithRandomDelay(ca)
		}(a)
	}
}

func (u *gtkUI) disconnectAll() {
	for _, a := range u.am.getAllConnectedAccounts() {
		go func(ca *account) {
			ca.disconnect()
		}(a)
	}
}
