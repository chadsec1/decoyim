package gui

import (
	"sort"
	"sync"
	"time"

	"github.com/coyim/coyim/config"
	"github.com/coyim/coyim/i18n"
	"github.com/coyim/coyim/session/events"
	smock "github.com/coyim/coyim/session/mock"
	"github.com/coyim/gotk3adapter/glib_mock"
	"github.com/coyim/gotk3adapter/glibi"
	"github.com/coyim/gotk3adapter/gtk_mock"
	"github.com/coyim/gotk3adapter/gtki"
	. "gopkg.in/check.v1"
)

type AccountSuite struct{}

var _ = Suite(&AccountSuite{})

type namedSessionMock struct {
	smock.SessionMock
	name string
}

func (v *namedSessionMock) GetConfig() *config.Account {
	return &config.Account{Account: v.name}
}

func (*AccountSuite) Test_account_sorting(c *C) {
	one := &account{session: &namedSessionMock{name: "bca"}}
	two := &account{session: &namedSessionMock{name: "abc"}}
	three := &account{session: &namedSessionMock{name: "cba"}}

	accounts := []*account{one, two, three}

	sort.Sort(byAccountNameAlphabetic(accounts))

	c.Assert(accounts, DeepEquals, []*account{two, one, three})
}

type accountInfoBarMock struct {
	gtk_mock.MockInfoBar

	hideCalled, destroyCalled int
}

func (v *accountInfoBarMock) Hide() {
	v.hideCalled++
}

func (v *accountInfoBarMock) Destroy() {
	v.destroyCalled++
}

func (*AccountSuite) Test_account_removeCurrentNotification_doesNothingIfItIsNil(c *C) {
	ac := &account{currentNotification: nil}
	ac.removeCurrentNotification()

	c.Assert(ac.currentNotification, IsNil)
}

func (*AccountSuite) Test_account_removeCurrentNotification_removesNotificationIfExists(c *C) {
	one := &accountInfoBarMock{}
	ac := &account{currentNotification: one}
	ac.removeCurrentNotification()

	c.Assert(ac.currentNotification, IsNil)
	c.Assert(one.hideCalled, Equals, 1)
	c.Assert(one.destroyCalled, Equals, 1)
}

func (*AccountSuite) Test_account_removeCurrentNotificationIf_doesNothingIfItIsntTheSameNotification(c *C) {
	one := &accountInfoBarMock{}
	two := &accountInfoBarMock{}
	ac := &account{currentNotification: one}
	ac.removeCurrentNotificationIf(two)

	c.Assert(ac.currentNotification, Equals, one)
	c.Assert(one.hideCalled, Equals, 0)
	c.Assert(one.destroyCalled, Equals, 0)
}

func (*AccountSuite) Test_account_removeCurrentNotificationIf_removesTheNotificationIfItMatches(c *C) {
	one := &accountInfoBarMock{}
	ac := &account{currentNotification: one}
	ac.removeCurrentNotificationIf(one)

	c.Assert(ac.currentNotification, IsNil)
	c.Assert(one.hideCalled, Equals, 1)
	c.Assert(one.destroyCalled, Equals, 1)
}

func (*AccountSuite) Test_account_IsAskingForPassword(c *C) {
	c.Assert((&account{askingForPassword: true}).IsAskingForPassword(), Equals, true)
	c.Assert((&account{askingForPassword: false}).IsAskingForPassword(), Equals, false)
}

func (*AccountSuite) Test_account_AskForPassword(c *C) {
	a := &account{}
	a.AskForPassword()
	c.Assert(a.askingForPassword, Equals, true)
}

func (*AccountSuite) Test_account_AskedForPassword(c *C) {
	a := &account{askingForPassword: true}
	a.AskedForPassword()
	c.Assert(a.askingForPassword, Equals, false)
}

type accountDirectGlibIdleAddMock struct {
	glib_mock.Mock
}

func (v *accountDirectGlibIdleAddMock) IdleAdd(v1 interface{}) glibi.SourceHandle {
	ffx := v1.(func())
	ffx()
	return glibi.SourceHandle(0)
}

type accountMockGtk struct {
	gtk_mock.Mock
}

func (*accountMockGtk) MenuNew() (gtki.Menu, error) {
	return &accountMockMenu{}, nil
}

func (*accountMockGtk) MenuItemNewWithMnemonic(mnem string) (gtki.MenuItem, error) {
	return &accountMockMenuItem{mnemonic: mnem, sensitive: true}, nil
}

func (*accountMockGtk) CheckMenuItemNewWithMnemonic(mnem string) (gtki.CheckMenuItem, error) {
	return &accountMockCheckMenuItem{mnemonic: mnem}, nil
}

func (*accountMockGtk) SeparatorMenuItemNew() (gtki.SeparatorMenuItem, error) {
	return &accountMockSeparatorMenuItem{}, nil
}

type accountMockMenu struct {
	gtk_mock.MockMenu

	menuItems []gtki.MenuItem
}

func (v *accountMockMenu) Append(v1 gtki.MenuItem) {
	v.menuItems = append(v.menuItems, v1)
}

func (v *accountMockMenu) GetMenuItemByName(name string) *accountMockMenuItem {
	switch name {
	case "Connect":
		return v.menuItems[0].(*accountMockMenuItem)
	case "Disconnect":
		return v.menuItems[1].(*accountMockMenuItem)
	case "Check Connection":
		return v.menuItems[2].(*accountMockMenuItem)
	case "Edit":
		return v.menuItems[5].(*accountMockMenuItem)
	case "Change Password":
		return v.menuItems[6].(*accountMockMenuItem)
	case "Remove":
		return v.menuItems[7].(*accountMockMenuItem)
	default:
		return nil
	}
}

func (v *accountMockMenu) GetCheckMenuItemByName(name string) *accountMockCheckMenuItem {
	switch name {
	case "Connect Automatically":
		return v.menuItems[9].(*accountMockCheckMenuItem)
	case "Always Encrypt Conversation":
		return v.menuItems[10].(*accountMockCheckMenuItem)
	default:
		return nil
	}
}

type accountMockMenuItem struct {
	gtk_mock.MockMenuItem

	mnemonic  string
	sensitive bool

	lock sync.Mutex

	onActivate interface{}
}

type accountMockCheckMenuItem struct {
	gtk_mock.MockCheckMenuItem

	mnemonic string
	active   bool

	onActivate interface{}
}

func (v *accountMockMenuItem) isSensitive() bool {
	v.lock.Lock()
	defer v.lock.Unlock()
	return v.sensitive
}

func (v *accountMockCheckMenuItem) SetActive(v1 bool) {
	v.active = v1
}

func (v *accountMockMenuItem) Connect(p string, v1 interface{}) glibi.SignalHandle {
	if p == "activate" {
		v.onActivate = v1
	}

	return glibi.SignalHandle(0)
}

func (v *accountMockMenuItem) SetSensitive(v1 bool) {
	v.lock.Lock()
	defer v.lock.Unlock()
	v.sensitive = v1
}

func (v *accountMockCheckMenuItem) Connect(p string, v1 interface{}) glibi.SignalHandle {
	if p == "activate" {
		v.onActivate = v1
	}

	return glibi.SignalHandle(0)
}

type accountMockSeparatorMenuItem struct {
	gtk_mock.MockSeparatorMenuItem
}

type accountMockGlib struct {
	glib_mock.Mock
}

func (*accountMockGlib) Local(vx string) string {
	return "[localized] " + vx
}

func (*AccountSuite) Test_account_createSubmenu_createsTheGeneralStructure(c *C) {
	i18n.InitLocalization(&accountMockGlib{})
	g = Graphics{gtk: &accountMockGtk{}, glib: &accountDirectGlibIdleAddMock{}}

	sess := &accountMockSession{config: &config.Account{}}
	a := &account{session: sess}
	menu := a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	c.Assert(menu, Not(IsNil))

	createdMenu := menu.(*accountMockMenu)

	c.Assert(createdMenu.menuItems, Not(IsNil))
	c.Assert(createdMenu.GetMenuItemByName("Check Connection").mnemonic, Equals, "[localized] _Check Connection")
	c.Assert(createdMenu.GetMenuItemByName("Connect").mnemonic, Equals, "[localized] _Connect")
	c.Assert(createdMenu.GetMenuItemByName("Disconnect").mnemonic, Equals, "[localized] _Disconnect")

	_, ok := createdMenu.menuItems[3].(*accountMockSeparatorMenuItem)
	c.Assert(ok, Equals, true)

	c.Assert(createdMenu.GetMenuItemByName("Edit").mnemonic, Equals, "[localized] _Edit...")
	c.Assert(createdMenu.GetMenuItemByName("Change Password").mnemonic, Equals, "[localized] _Change Password...")
	c.Assert(createdMenu.GetMenuItemByName("Remove").mnemonic, Equals, "[localized] _Remove")

	_, ok = createdMenu.menuItems[8].(*accountMockSeparatorMenuItem)
	c.Assert(ok, Equals, true)

	c.Assert(createdMenu.GetCheckMenuItemByName("Connect Automatically").mnemonic, Equals, "[localized] Connect _Automatically")
	c.Assert(createdMenu.GetCheckMenuItemByName("Always Encrypt Conversation").mnemonic, Equals, "[localized] Always Encrypt Conversation")
}

func (*AccountSuite) Test_account_createSubmenu_setsTheCheckboxesCorrectly(c *C) {
	i18n.InitLocalization(&accountMockGlib{})
	g = Graphics{gtk: &accountMockGtk{}, glib: &accountDirectGlibIdleAddMock{}}

	conf := &config.Account{ConnectAutomatically: true, AlwaysEncrypt: true}
	sess := &accountMockSession{config: conf}
	a := &account{session: sess}

	menu := a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	createdMenu := menu.(*accountMockMenu)
	c.Assert(createdMenu.GetCheckMenuItemByName("Connect Automatically").active, Equals, true)
	c.Assert(createdMenu.GetCheckMenuItemByName("Always Encrypt Conversation").active, Equals, true)

	conf.AlwaysEncrypt = false
	menu = a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	createdMenu = menu.(*accountMockMenu)
	c.Assert(createdMenu.GetCheckMenuItemByName("Connect Automatically").active, Equals, true)
	c.Assert(createdMenu.GetCheckMenuItemByName("Always Encrypt Conversation").active, Equals, false)

	conf.ConnectAutomatically = false
	menu = a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	createdMenu = menu.(*accountMockMenu)
	c.Assert(createdMenu.GetCheckMenuItemByName("Connect Automatically").active, Equals, false)
	c.Assert(createdMenu.GetCheckMenuItemByName("Always Encrypt Conversation").active, Equals, false)

	conf.AlwaysEncrypt = true
	menu = a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	createdMenu = menu.(*accountMockMenu)
	c.Assert(createdMenu.GetCheckMenuItemByName("Connect Automatically").active, Equals, false)
	c.Assert(createdMenu.GetCheckMenuItemByName("Always Encrypt Conversation").active, Equals, true)
}

func (*AccountSuite) Test_account_createSubmenu_setsActivationCorrectly(c *C) {
	i18n.InitLocalization(&accountMockGlib{})
	g = Graphics{gtk: &accountMockGtk{}, glib: &accountDirectGlibIdleAddMock{}}

	sess := &accountMockSession{config: &config.Account{}}
	a := &account{session: sess}

	menu := a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	createdMenu := menu.(*accountMockMenu)

	// We can't really check that these things are set to the correct functions, just that they are set
	// It might be possible to try invoking them and see that they do the right things, at some point
	// For now, too much bother.

	c.Assert(createdMenu.GetMenuItemByName("Check Connection").onActivate, Not(IsNil))
	c.Assert(createdMenu.GetMenuItemByName("Connect").onActivate, Not(IsNil))
	c.Assert(createdMenu.GetMenuItemByName("Disconnect").onActivate, Not(IsNil))

	c.Assert(createdMenu.GetMenuItemByName("Edit").onActivate, Not(IsNil))
	c.Assert(createdMenu.GetMenuItemByName("Change Password").onActivate, Not(IsNil))
	c.Assert(createdMenu.GetMenuItemByName("Remove").onActivate, Not(IsNil))

	c.Assert(createdMenu.GetCheckMenuItemByName("Connect Automatically").onActivate, Not(IsNil))
	c.Assert(createdMenu.GetCheckMenuItemByName("Always Encrypt Conversation").onActivate, Not(IsNil))
}

type accountMockSession struct {
	smock.SessionMock

	isDisconnected bool
	isConnected    bool
	config         *config.Account
	events         []chan<- interface{}

	lock sync.Mutex
}

func (v *accountMockSession) IsDisconnected() bool {
	v.lock.Lock()
	defer v.lock.Unlock()
	return v.isDisconnected
}

func (v *accountMockSession) setIsDisconnected(val bool) {
	v.lock.Lock()
	defer v.lock.Unlock()
	v.isDisconnected = val
}

func (v *accountMockSession) IsConnected() bool {
	return v.isConnected
}

func (v *accountMockSession) GetConfig() *config.Account {
	return v.config
}

func (v *accountMockSession) Subscribe(v1 chan<- interface{}) {
	v.events = append(v.events, v1)
}

func (*AccountSuite) Test_account_createSubmenu_setsConnectAndDisconnectSensitivity(c *C) {
	i18n.InitLocalization(&accountMockGlib{})
	g = Graphics{gtk: &accountMockGtk{}, glib: &accountDirectGlibIdleAddMock{}}

	sess := &accountMockSession{isDisconnected: true, config: &config.Account{}}
	a := &account{session: sess}

	menu := a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	createdMenu := menu.(*accountMockMenu)
	c.Assert(createdMenu.GetMenuItemByName("Check Connection").isSensitive(), Equals, false)
	c.Assert(createdMenu.GetMenuItemByName("Connect").isSensitive(), Equals, true)
	c.Assert(createdMenu.GetMenuItemByName("Disconnect").isSensitive(), Equals, false)

	sess.setIsDisconnected(false)
	sess.isConnected = true
	menu = a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	createdMenu = menu.(*accountMockMenu)
	c.Assert(createdMenu.GetMenuItemByName("Check Connection").isSensitive(), Equals, true)
	c.Assert(createdMenu.GetMenuItemByName("Connect").isSensitive(), Equals, false)
	c.Assert(createdMenu.GetMenuItemByName("Disconnect").isSensitive(), Equals, true)
}

func (*AccountSuite) Test_account_createSubmenu_willWatchForThingsToChangeTheConnectSensitivity(c *C) {
	i18n.InitLocalization(&accountMockGlib{})
	g = Graphics{gtk: &accountMockGtk{}, glib: &accountDirectGlibIdleAddMock{}}

	sess := &accountMockSession{isDisconnected: true, config: &config.Account{}}
	a := &account{session: sess}

	menu := a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	connectItem := menu.(*accountMockMenu).menuItems[0].(*accountMockMenuItem)

	c.Assert(connectItem.isSensitive(), Equals, true)

	sess.setIsDisconnected(false)
	for _, cc := range sess.events {
		cc <- events.Event{
			Type: events.Connecting,
		}
	}

	waitFor(c, func() bool { return !connectItem.isSensitive() })

	sess.setIsDisconnected(false)
	for _, cc := range sess.events {
		cc <- events.Event{
			Type: events.Connected,
		}
	}

	waitFor(c, func() bool { return !connectItem.isSensitive() })

	sess.setIsDisconnected(true)
	for _, cc := range sess.events {
		cc <- events.Event{
			Type: events.Disconnected,
		}
	}

	waitFor(c, func() bool { return connectItem.isSensitive() })
}

func waitFor(c *C, f func() bool) {
	cx := make(chan bool)

	go func() {
		for !f() {
			time.Sleep(time.Duration(20) * time.Millisecond)
		}
		cx <- true
	}()

	select {
	case <-time.After(5 * time.Second):
		c.Assert(f(), Equals, true)
	case <-cx:
		c.Assert(f(), Equals, true)
	}
}

func (*AccountSuite) Test_account_createSubmenu_willWatchForThingsToChangeTheDisconnectSensitivity(c *C) {
	i18n.InitLocalization(&accountMockGlib{})
	g = Graphics{gtk: &accountMockGtk{}, glib: &accountDirectGlibIdleAddMock{}}

	sess := &accountMockSession{isDisconnected: true, config: &config.Account{}}
	a := &account{session: sess}

	menu := a.createSubmenu(&gtkUI{am: &accountManager{accounts: []*account{a}}})
	disconnectItem := menu.(*accountMockMenu).menuItems[1].(*accountMockMenuItem)

	c.Assert(disconnectItem.isSensitive(), Equals, false)

	sess.setIsDisconnected(false)
	for _, cc := range sess.events {
		cc <- events.Event{
			Type: events.Connecting,
		}
	}

	waitFor(c, func() bool { return disconnectItem.isSensitive() })

	sess.setIsDisconnected(false)
	for _, cc := range sess.events {
		cc <- events.Event{
			Type: events.Connected,
		}
	}

	waitFor(c, func() bool { return disconnectItem.isSensitive() })

	sess.setIsDisconnected(true)
	for _, cc := range sess.events {
		cc <- events.Event{
			Type: events.Disconnected,
		}
	}

	waitFor(c, func() bool { return !disconnectItem.isSensitive() })
}
