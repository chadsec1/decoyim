package gui

import (
	"sync"

	"github.com/coyim/coyim/config"
	"github.com/coyim/coyim/i18n"
	"github.com/coyim/gotk3adapter/glibi"
	"github.com/coyim/gotk3adapter/gtki"
)

var (
	// TODO: shouldn't this be specific to the account ID in question?
	accountChangedSignal glibi.Signal
)

var accountsLock sync.Mutex

func (u *gtkUI) buildStaticAccountsMenu(submenu gtki.Menu) {
	connectAutomaticallyItem, _ := g.gtk.CheckMenuItemNewWithMnemonic(i18n.Local("Connect On _Startup"))
	u.config.config().WhenLoaded(func(a *config.ApplicationConfig) {
		connectAutomaticallyItem.SetActive(a.ConnectAutomatically)
	})

	_ = connectAutomaticallyItem.Connect("activate", func() {
		u.setConnectAllAutomatically(connectAutomaticallyItem.GetActive())
	})
	submenu.Append(connectAutomaticallyItem)

	connectAllMenu, _ := g.gtk.MenuItemNewWithMnemonic(i18n.Local("_Connect All"))
	_ = connectAllMenu.Connect("activate", func() { u.connectAllAutomatics(true) })
	submenu.Append(connectAllMenu)

	disconnectAllMenu, _ := g.gtk.MenuItemNewWithMnemonic(i18n.Local("_Disconnect All"))
	_ = disconnectAllMenu.Connect("activate", u.disconnectAll)
	submenu.Append(disconnectAllMenu)

	sep2, _ := g.gtk.SeparatorMenuItemNew()
	submenu.Append(sep2)

	registerAccMenu, _ := g.gtk.MenuItemNewWithMnemonic(i18n.Local("_New Account"))
	_ = registerAccMenu.Connect("activate", u.showServerSelectionWindow)
	submenu.Append(registerAccMenu)

	addAccMenu, _ := g.gtk.MenuItemNewWithMnemonic(i18n.Local("_Add Account"))
	_ = addAccMenu.Connect("activate", u.showAddAccountWindow)
	submenu.Append(addAccMenu)

	importMenu, _ := g.gtk.MenuItemNewWithMnemonic(i18n.Local("_Import Account"))
	_ = importMenu.Connect("activate", u.runImporter)
	submenu.Append(importMenu)

	connectAutomaticallyItem.SetSensitive(false)
	connectAllMenu.SetSensitive(false)
	disconnectAllMenu.SetSensitive(false)
	registerAccMenu.SetSensitive(false)
	addAccMenu.SetSensitive(false)
	importMenu.SetSensitive(false)

	u.config.whenHaveConfig(func() {
		doInUIThread(func() {
			connectAutomaticallyItem.SetSensitive(true)
			connectAllMenu.SetSensitive(true)
			disconnectAllMenu.SetSensitive(true)
			registerAccMenu.SetSensitive(true)
			addAccMenu.SetSensitive(true)
			importMenu.SetSensitive(true)
		})
	})
}

func (u *gtkUI) buildAccountsMenu() {
	accountsLock.Lock()
	defer accountsLock.Unlock()

	submenu, _ := g.gtk.MenuNew()

	allAccounts := u.am.getAllAccounts()
	for _, account := range allAccounts {
		account.appendMenuTo(u, submenu)
	}

	if len(allAccounts) > 0 {
		sep, _ := g.gtk.SeparatorMenuItemNew()
		submenu.Append(sep)
	}

	u.buildStaticAccountsMenu(submenu)

	submenu.ShowAll()

	u.mainUI.accountsMenuItem.SetSubmenu(submenu)
}
