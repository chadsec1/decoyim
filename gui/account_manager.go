package gui

import (
	"fmt"
	"sync"

	"github.com/chadsec1/decoyim/config"
	"github.com/chadsec1/decoyim/decoylog"
	"github.com/chadsec1/decoyim/otrclient"
	rosters "github.com/chadsec1/decoyim/roster"
	"github.com/chadsec1/decoyim/session/access"
	sessions "github.com/chadsec1/decoyim/session/access"
	"github.com/chadsec1/decoyim/xmpp/interfaces"
	"github.com/chadsec1/decoyim/xmpp/jid"
)

type accountManager struct {
	accounts []*account
	events   chan interface{}
	contacts map[*account]*rosters.List

	connectedAccountsObserversIndex int
	connectedAccountsObservers      map[int]func()
	connectedAccountsObserversLock  sync.RWMutex

	accountEncryption otrclient.CommandManager

	log decoylog.Logger

	sessionFactory sessions.Factory
	dialerFactory  interfaces.DialerFactory

	lock sync.RWMutex
}

func (m *accountManager) init(c otrclient.CommandManager, log decoylog.Logger) {
	m.events = make(chan interface{}, 10)
	m.accounts = make([]*account, 0, 5)
	m.contacts = make(map[*account]*rosters.List)
	m.accountEncryption = c
	m.log = log
	m.connectedAccountsObservers = make(map[int]func())
}

func (m *accountManager) getAllAccounts() []*account {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return append([]*account(nil), m.accounts...)
}

// onChangeOfConnectedAccounts will register a callback - this function will
// be called when the list of all connected accounts change in some way
// It might be called more than once for a specific event
func (m *accountManager) onChangeOfConnectedAccounts(f func()) int {
	m.connectedAccountsObserversLock.Lock()
	defer m.connectedAccountsObserversLock.Unlock()

	token := m.connectedAccountsObserversIndex
	m.connectedAccountsObserversIndex++

	m.connectedAccountsObservers[token] = f

	return token
}

func (m *accountManager) notifyChangeOfConnectedAccounts() {
	m.connectedAccountsObserversLock.RLock()
	defer m.connectedAccountsObserversLock.RUnlock()

	for _, f := range m.connectedAccountsObservers {
		go f()
	}
}

func (m *accountManager) removeConnectedAccountsObserver(token int) {
	m.connectedAccountsObserversLock.Lock()
	defer m.connectedAccountsObserversLock.Unlock()

	delete(m.connectedAccountsObservers, token)
}

func (m *accountManager) getAllConnectedAccounts() []*account {
	m.lock.RLock()
	defer m.lock.RUnlock()

	accounts := make([]*account, 0, len(m.accounts))
	for _, acc := range m.accounts {
		if !acc.connected() {
			continue
		}

		accounts = append(accounts, acc)
	}

	return accounts
}

func (m *accountManager) disconnectAll() {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, acc := range m.accounts {
		acc.disconnect()
	}
}

func (m *accountManager) getAccountByID(ID string) (*account, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, acc := range m.accounts {
		if acc.ID() == ID {
			return acc, true
		}
	}

	return nil, false
}

func (m *accountManager) getAllContacts() map[*account]*rosters.List {
	return m.contacts
}

func (m *accountManager) getContacts(acc *account) *rosters.List {
	return m.contacts[acc]
}

func (m *accountManager) setContacts(account *account, contacts *rosters.List) {
	if account == nil {
		panic("Developer error: account should never be nil")
	}
	m.contacts[account] = contacts
}

func (u *gtkUI) addAccount(appConfig *config.ApplicationConfig, account *config.Account, sf access.Factory, df interfaces.DialerFactory) {
	defer u.notifyChangeOfConnectedAccounts()

	u.lock.Lock()
	defer u.lock.Unlock()

	acc := newAccount(appConfig, account, sf, df)
	go u.observeAccountEvents(acc)
	acc.log = u.accountManager.log.WithField("account", account.Account)
	acc.session.Subscribe(acc.events)
	acc.session.SetCommandManager(u)
	acc.session.SetConnector(acc)

	u.accounts = append(u.accounts, acc)
	u.setContacts(acc, rosters.New())
}

func (m *accountManager) removeAccount(conf *config.Account, k func()) {
	defer m.notifyChangeOfConnectedAccounts()

	toRemove, exists := m.getAccountByID(conf.ID())
	if !exists {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	accs := make([]*account, 0, len(m.accounts)-1)
	for _, acc := range m.accounts {
		if acc == toRemove {
			delete(m.contacts, acc)
			continue
		}

		accs = append(accs, acc)
	}

	m.accounts = accs

	k()
}

func (u *gtkUI) buildAccounts(appConfig *config.ApplicationConfig, sf access.Factory, df interfaces.DialerFactory) {
	defer u.notifyChangeOfConnectedAccounts()

	hasConfUpdates := false
	for _, accountConf := range appConfig.Accounts {
		if _, ok := u.getAccountByID(accountConf.ID()); ok {
			continue
		}

		hasUpdate, err := accountConf.EnsurePrivateKey()
		if err != nil {
			continue
		}

		hasConfUpdates = hasConfUpdates || hasUpdate
		u.addAccount(appConfig, accountConf, sf, df)
	}

	if hasConfUpdates {
		u.ExecuteCmd(otrclient.SaveApplicationConfigCmd{})
	}
}

func (u *gtkUI) addNewAccountsFromConfig(appConfig *config.ApplicationConfig, sf access.Factory, df interfaces.DialerFactory) {
	defer u.notifyChangeOfConnectedAccounts()

	for _, configAccount := range appConfig.Accounts {
		_, found := u.getAccountByID(configAccount.ID())
		if found {
			continue
		}

		u.addAccount(appConfig, configAccount, sf, df)
	}
}

func (m *accountManager) removePeer(account *account, peer jid.WithoutResource) {
	m.lock.Lock()
	defer m.lock.Unlock()

	l, ok := m.contacts[account]
	if !ok {
		return
	}

	l.Remove(peer)
}

func (m *accountManager) getPeer(account *account, peer jid.WithoutResource) (*rosters.Peer, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	l, ok := m.contacts[account]
	if !ok {
		account.log.Warn("Failure to look up account from account manager")
		return nil, false
	}

	return l.Get(peer)
}

func (m *accountManager) displayNameFor(account *account, peer jid.WithoutResource) string {
	p, ok := m.getPeer(account, peer)
	if !ok {
		return peer.String()
	}

	return p.NameForPresentation()
}

func (m *accountManager) debugPeersFor(account *account) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	rs, ok := m.getAllContacts()[account]
	if ok {
		rs.Iter(func(_ int, item *rosters.Peer) {
			fmt.Printf("->   %s\n", item.Dump())
		})
	}

	fmt.Printf(" ************************************** \n")
	fmt.Println()
}
