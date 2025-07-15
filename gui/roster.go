package gui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/chadsec1/decoyim/config"
	"github.com/chadsec1/decoyim/i18n"
	rosters "github.com/chadsec1/decoyim/roster"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gdki"
	"github.com/coyim/gotk3adapter/glibi"
	"github.com/coyim/gotk3adapter/gtki"
)

type roster struct {
	widget gtki.ScrolledWindow `gtk-widget:"roster"`
	view   gtki.TreeView       `gtk-widget:"roster-tree"`
	model  gtki.TreeStore

	fields *rosterFields

	isCollapsed map[string]bool
	toCollapse  []gtki.TreePath

	ui *gtkUI
}

type rosterFields struct {
	jid             *stringStoreField
	displayName     *stringStoreField
	accountID       *stringStoreField
	statusColor     *stringStoreField
	backgroundColor *stringStoreField
	weight          *intStoreField
	tooltip         *stringStoreField
	statusIcon      *pixbufStoreField
	rowType         *stringStoreField
	cssClass        *stringStoreField
}

const (
	indexJid               = 0
	indexDisplayName       = 1
	indexAccountID         = 2
	indexStatusColor       = 3
	indexBackgroundColor   = 4
	indexWeight            = 5
	indexTooltip           = 6
	indexStatusIcon        = 7
	indexRowType           = 8
	indexCSSClass          = 9
	indexParentJid         = 0
	indexParentDisplayName = 1
)

func createTreeModelAndAccessors() (gtki.TreeStore, *rosterFields) {
	model, err := g.gtk.TreeStoreNew(
		// jid
		glibi.TYPE_STRING,
		// display name
		glibi.TYPE_STRING,
		// account id
		glibi.TYPE_STRING,
		// status color
		glibi.TYPE_STRING,
		// background color
		glibi.TYPE_STRING,
		// weight of font
		glibi.TYPE_INT,
		// tooltip
		glibi.TYPE_STRING,
		// status icon
		pixbufType(),
		// row type
		glibi.TYPE_STRING,
		// css class
		glibi.TYPE_STRING,
	)
	if err != nil {
		panic(err)
	}

	fields := &rosterFields{}

	fields.jid = newStringStoreField(model, indexJid)
	fields.displayName = newStringStoreField(model, indexDisplayName)
	fields.accountID = newStringStoreField(model, indexAccountID)
	fields.statusColor = newStringStoreField(model, indexStatusColor)
	fields.backgroundColor = newStringStoreField(model, indexBackgroundColor)
	fields.weight = newIntStoreField(model, indexWeight)
	fields.tooltip = newStringStoreField(model, indexTooltip)
	fields.statusIcon = newPixbufStoreField(model, indexStatusIcon)
	fields.rowType = newStringStoreField(model, indexRowType)
	fields.cssClass = newStringStoreField(model, indexCSSClass)

	return model, fields
}

func (r *roster) init(u *gtkUI) {
	builder := newBuilder("Roster")

	r.isCollapsed = make(map[string]bool)
	r.ui = u

	builder.ConnectSignals(map[string]interface{}{
		"on_activate_buddy": r.onActivateRosterRow,
		"on_button_press":   r.onButtonPress,
	})

	panicOnDevError(builder.bindObjects(r))

	r.view.SetEnableSearch(true)
	r.view.SetSearchEqualSubstringMatch()

	r.model, r.fields = createTreeModelAndAccessors()

	r.view.SetModel(r.model)
}

func (r *roster) getAccountAndJidFromEvent(bt gdki.EventButton) (j jid.WithoutResource, account *account, rowType string, ok bool) {
	x := bt.X()
	y := bt.Y()
	path, _, _, _, found := r.view.GetPathAtPos(int(x), int(y))
	if !found {
		return nil, nil, "", false
	}
	iter, err := r.model.GetIter(path)
	if err != nil {
		return nil, nil, "", false
	}
	j = jid.NR(r.fields.jid.get(iter))
	accountID := r.fields.accountID.get(iter)
	rowType = r.fields.rowType.get(iter)
	account, ok = r.ui.accountManager.getAccountByID(accountID)
	return j, account, rowType, ok
}

func sortedGroupNames(groups map[string]bool) []string {
	sortedNames := make([]string, 0, len(groups))
	for k := range groups {
		sortedNames = append(sortedNames, k)
	}

	sort.Strings(sortedNames)

	return sortedNames
}

func (r *roster) getGroupNamesFor(a *account) []string {
	groups := map[string]bool{}
	contacts := r.ui.accountManager.getAllContacts()[a]
	for name := range contacts.GetGroupNames() {
		if groups[name] {
			continue
		}

		groups[name] = true
	}

	return sortedGroupNames(groups)
}

func (r *roster) updatePeer(acc *account, jid jid.WithoutResource, nickname string, groups []string, updateRequireEncryption, requireEncryption bool) error {
	peer, ok := r.ui.getPeer(acc, jid)
	if !ok {
		return fmt.Errorf("Could not find peer %s", jid)
	}

	// This updates what is displayed in the roster
	peer.Nickname = nickname
	peer.SetGroups(groups)

	// NOTE: This requires the account to be connected in order to rename peers,
	// which should not be the case. This is one example of why gui.account should
	// own the account config -  and not the session.
	conf := acc.session.GetConfig()
	conf.SavePeerDetails(jid.String(), nickname, groups)
	if updateRequireEncryption {
		conf.UpdateEncryptionRequired(jid.String(), requireEncryption)
	}

	r.ui.SaveConfig()
	doInUIThread(r.redraw)

	return nil
}

func toArray(groupList gtki.ListStore) []string {
	groups := []string{}

	iter, ok := groupList.GetIterFirst()
	for ok {
		gValue, _ := groupList.GetValue(iter, 0)
		if group, err := gValue.GetString(); err == nil {
			groups = append(groups, group)
		}

		ok = groupList.IterNext(iter)
	}

	return groups
}

func (r *roster) setSensitive(menuItem gtki.MenuItem, account *account, peer jid.WithoutResource) {
	p, ok := r.ui.getPeer(account, peer)
	if !ok {
		return
	}

	menuItem.SetSensitive(p.HasResources())
}

func (r *roster) createAccountPeerPopup(jid jid.WithoutResource, account *account, bt gdki.EventButton) {
	builder := newBuilder("ContactPopupMenu")
	mn := builder.getObj("contactMenu").(gtki.Menu)

	resourcesMenuItem := builder.getObj("resourcesMenuItem").(gtki.MenuItem)
	r.appendResourcesAsMenuItems(jid, account, resourcesMenuItem)

	sendFileMenuItem := builder.getObj("sendFileMenuItem").(gtki.MenuItem)
	r.setSensitive(sendFileMenuItem, account, jid)

	sendDirMenuItem := builder.getObj("sendDirectoryMenuItem").(gtki.MenuItem)
	r.setSensitive(sendDirMenuItem, account, jid)

	builder.ConnectSignals(map[string]interface{}{
		"on_remove_contact": func() {
			account.session.RemoveContact(jid.String())
			r.ui.removePeer(account, jid)
			r.redraw()
		},
		"on_edit_contact": func() {
			doInUIThread(func() { r.openEditContactDialog(jid, account) })
		},
		"on_allow_contact_to_see_status": func() {
			_ = account.session.ApprovePresenceSubscription(jid, "" /* generate id */)
		},
		"on_forbid_contact_to_see_status": func() {
			_ = account.session.DenyPresenceSubscription(jid, "" /* generate id */)
		},
		"on_ask_contact_to_see_status": func() {
			_ = account.session.RequestPresenceSubscription(jid, "")
		},
		"on_dump_info": func() {
			r.ui.accountManager.debugPeersFor(account)
		},
		"on_send_file_to_contact": func() {
			account.sendFileTo(jid, r.ui, nil)
		},
		"on_send_directory_to_contact": func() {
			account.sendDirectoryTo(jid, r.ui, nil)
		},
	})

	mn.ShowAll()
	mn.PopupAtPointer(bt)
}

func (r *roster) appendResourcesAsMenuItems(jid jid.WithoutResource, account *account, menuItem gtki.MenuItem) {
	peer, ok := r.ui.getPeer(account, jid)
	if !ok {
		return
	}

	hasResources := peer.HasResources()
	menuItem.SetSensitive(hasResources)

	if !hasResources {
		return
	}

	innerMenu, _ := g.gtk.MenuNew()
	for _, resource := range peer.Resources() {
		item, _ := g.gtk.CheckMenuItemNewWithMnemonic(resource.String())
		rs := resource
		_ = item.Connect("activate",
			func() {
				doInUIThread(func() {
					r.ui.openTargetedConversationView(account, jid.WithResource(rs), true)
				})
			})
		innerMenu.Append(item)
	}

	menuItem.SetSubmenu(innerMenu)
}

func (r *roster) createAccountPopup(account *account, bt gdki.EventButton) {
	mn := account.createSubmenu(r.ui)
	if *config.DebugFlag {
		mn.Append(account.createSeparatorItem())
		mn.Append(account.createDumpInfoItem(r))
		mn.Append(account.createXMLConsoleItem(r.ui.window))
	}
	mn.ShowAll()
	mn.PopupAtPointer(bt)
}

func (r *roster) onButtonPress(view gtki.TreeView, ev gdki.Event) bool {
	bt := g.gdk.EventButtonFrom(ev)
	if bt.Button() == 0x03 {
		jid, account, rowType, ok := r.getAccountAndJidFromEvent(bt)
		if ok {
			switch rowType {
			case "peer":
				r.createAccountPeerPopup(jid.NoResource(), account, bt)
			case "account":
				r.createAccountPopup(account, bt)
			}
		}
	}

	return false
}

func collapseTransform(s string) string {
	res := sha256.Sum256([]byte(s))
	return hex.EncodeToString(res[:])
}

func (r *roster) restoreCollapseStatus() {
	pieces := strings.Split(r.ui.settings.GetCollapsed(), ":")
	for _, p := range pieces {
		if p != "" {
			r.isCollapsed[p] = true
		}
	}
}

func (r *roster) saveCollapseStatus() {
	var vals []string
	for e, v := range r.isCollapsed {
		if v {
			vals = append(vals, e)
		}
	}
	r.ui.settings.SetCollapsed(strings.Join(vals, ":"))
}

func (r *roster) activateAccountRow(jid string) {
	ix := collapseTransform(jid)
	r.isCollapsed[ix] = !r.isCollapsed[ix]
	r.saveCollapseStatus()
	r.redraw()
}

func (r *roster) onActivateRosterRow(v gtki.TreeView, path gtki.TreePath) {
	iter, err := r.model.GetIter(path)
	if err != nil {
		return
	}

	peer := r.fields.jid.get(iter)
	rowType := r.fields.rowType.get(iter)

	switch rowType {
	case "peer":
		selection, err := v.GetSelection()
		if err != nil {
			return
		}

		defer selection.UnselectPath(path)
		accountID := r.fields.accountID.get(iter)
		account, ok := r.ui.accountManager.getAccountByID(accountID)
		if !ok {
			return
		}
		r.ui.openConversationView(account, jid.NR(peer), true)
	case "account":
		r.activateAccountRow(peer)
	case "group":
		// We ignore this, since a double click on the group doesn't really have any effect
	default:
		panic(fmt.Sprintf("unknown roster row type: %s", rowType))
	}
}

func (r *roster) update(account *account, entries *rosters.List) {
	r.ui.accountManager.lock.Lock()
	defer r.ui.accountManager.lock.Unlock()

	r.ui.accountManager.setContacts(account, entries)
}

func willDisplayTheirStatusToUs(p *rosters.Peer) bool {
	return p.Subscription == "to" ||
		p.Subscription == "both"
}

func isWaitingForResponse(p *rosters.Peer) bool {
	return p.Subscription == "none" ||
		p.Subscription == "" ||
		p.Subscription == "from" ||
		p.PendingSubscribeID != "" ||
		p.Asked
}

func isNominallyVisible(accountName string, p *rosters.Peer, showWaiting bool) bool {
	return accountName != p.Jid.String() && (willDisplayTheirStatusToUs(p) || (showWaiting && isWaitingForResponse(p)))
}

func shouldDisplay(accountName string, p *rosters.Peer, showOffline, showWaiting bool) bool {
	return isNominallyVisible(accountName, p, showWaiting) && ((showOffline || p.IsOnline()) || (showWaiting && isWaitingForResponse(p)))
}

func isOnline(p *rosters.Peer) bool {
	return p.PendingSubscribeID == "" && p.IsOnline()
}

func decideStatusFor(p *rosters.Peer) string {
	if p.PendingSubscribeID != "" || p.Asked {
		return "unknown"
	}

	if !p.IsOnline() {
		return "offline"
	}

	switch p.MainStatus() {
	case "dnd":
		return "busy"
	case "xa":
		return "extended-away"
	case "away":
		return "away"
	}

	return "available"
}

func decideColorFor(cs colorSet, p *rosters.Peer) cssColor {
	if !p.IsOnline() {
		return cs.rosterPeerOfflineForeground
	}
	return cs.rosterPeerOnlineForeground
}

func createGroupDisplayName(parentName string, counter *counter, isExpanded bool) string {
	name := parentName
	if !isExpanded {
		name = fmt.Sprintf("[%s]", name)
	}
	return fmt.Sprintf("%s (%d/%d)", name, counter.online, counter.total)
}

func createTooltipFor(item *rosters.Peer) string {
	pname := html.EscapeString(item.NameForPresentation())
	jid := html.EscapeString(item.Jid.String())
	if pname != jid {
		return fmt.Sprintf("%s (%s)", pname, jid)
	}
	return jid
}

func (r *roster) addItem(item *rosters.Peer, parentIter gtki.TreeIter, indent string) {
	cs := r.ui.currentColorSet()
	iter := r.model.Append(parentIter)
	potentialExtra := ""
	if isWaitingForResponse(item) {
		potentialExtra = i18n.Local(" (waiting for approval)")
	}

	r.fields.jid.set(iter, item.Jid.String())
	r.fields.displayName.set(iter, fmt.Sprintf("%s %s%s", indent, item.NameForPresentation(), potentialExtra))
	r.fields.accountID.set(iter, item.BelongsTo)
	r.fields.statusColor.set(iter, decideColorFor(cs, item).toCSS())
	r.fields.backgroundColor.set(iter, cs.rosterPeerBackground.toCSS())
	r.fields.tooltip.set(iter, createTooltipFor(item))
	r.fields.statusIcon.set(iter, statusIcons[decideStatusFor(item)].GetPixbuf())
	r.fields.rowType.set(iter, "peer")
}

func (r *roster) redrawMerged() {
	showOffline := !r.ui.config().Display.ShowOnlyOnline
	showWaiting := !r.ui.config().Display.ShowOnlyConfirmed

	r.ui.accountManager.lock.RLock()
	defer r.ui.accountManager.lock.RUnlock()

	r.toCollapse = nil

	grp := rosters.TopLevelGroup()
	for account, contacts := range r.ui.accountManager.getAllContacts() {
		contacts.AddTo(grp, account.session.GroupDelimiter())
	}

	accountCounter := &counter{}
	r.displayGroup(grp, nil, accountCounter, showOffline, showWaiting, "")

	r.view.ExpandAll()
	for _, path := range r.toCollapse {
		r.view.CollapseRow(path)
	}
}

type counter struct {
	total  int
	online int
}

func (c *counter) inc(total, online bool) {
	if total {
		c.total++
	}
	if online {
		c.online++
	}
}

func (r *roster) sortedPeers(ps []*rosters.Peer) []*rosters.Peer {
	if r.ui.config().Display.SortByStatus {
		sort.Sort(byStatus(ps))
	} else {
		sort.Sort(byNameForPresentation(ps))
	}
	return ps
}

type byNameForPresentation []*rosters.Peer

func (s byNameForPresentation) Len() int { return len(s) }
func (s byNameForPresentation) Less(i, j int) bool {
	return s[i].NameForPresentation() < s[j].NameForPresentation()
}
func (s byNameForPresentation) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func statusValueFor(s string) int {
	switch s {
	case "available":
		return 0
	case "away":
		return 1
	case "extended-away":
		return 2
	case "busy":
		return 3
	case "offline":
		return 4
	case "unknown":
		return 5
	}
	return -1
}

type byStatus []*rosters.Peer

func (s byStatus) Len() int { return len(s) }
func (s byStatus) Less(i, j int) bool {
	return statusValueFor(decideStatusFor(s[i])) < statusValueFor(decideStatusFor(s[j]))
}
func (s byStatus) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (r *roster) displayGroup(g *rosters.Group, parentIter gtki.TreeIter, accountCounter *counter, showOffline, showWaiting bool, accountName string) {
	pi := parentIter
	groupCounter := &counter{}
	groupID := accountName + "//" + g.FullGroupName()

	isEmpty := true
	for _, item := range g.UnsortedPeers() {
		if shouldDisplay(accountName, item, showOffline, showWaiting) {
			isEmpty = false
		}
	}

	if g.GroupName != "" && (!isEmpty || r.showEmptyGroups()) {
		pi = r.model.Append(parentIter)
		r.fields.jid.set(pi, groupID)
		r.fields.rowType.set(pi, "group")
		r.fields.weight.set(pi, 500)
		r.fields.backgroundColor.set(pi, r.ui.currentColorSet().rosterGroupBackground.toCSS())
	}

	for _, item := range r.sortedPeers(g.UnsortedPeers()) {
		vs := isNominallyVisible(accountName, item, showWaiting)
		o := isOnline(item)
		accountCounter.inc(vs, vs && o)
		groupCounter.inc(vs, vs && o)

		if shouldDisplay(accountName, item, showOffline, showWaiting) {
			r.addItem(item, pi, "")
		}
	}

	for _, gr := range g.Groups() {
		r.displayGroup(gr, pi, accountCounter, showOffline, showWaiting, accountName)
	}

	if g.GroupName != "" && (!isEmpty || r.showEmptyGroups()) {
		parentPath, _ := r.model.GetPath(pi)
		shouldCollapse, ok := r.isCollapsed[collapseTransform(groupID)]
		isExpanded := true
		if ok && shouldCollapse {
			isExpanded = false
			r.toCollapse = append(r.toCollapse, parentPath)
		}

		r.fields.displayName.set(pi, createGroupDisplayName(g.FullGroupName(), groupCounter, isExpanded))
	}
}

func (r *roster) redrawSeparateAccount(account *account, contacts *rosters.List, showOffline, showWaiting bool) {
	cs := r.ui.currentColorSet()
	parentIter := r.model.Append(nil)

	accountCounter := &counter{}

	grp := contacts.Grouped(account.session.GroupDelimiter())
	parentName := account.Account()
	r.displayGroup(grp, parentIter, accountCounter, showOffline, showWaiting, parentName)

	r.fields.jid.set(parentIter, parentName)
	r.fields.accountID.set(parentIter, account.ID())
	r.fields.rowType.set(parentIter, "account")
	r.fields.weight.set(parentIter, 700)

	bgcolor := cs.rosterAccountOnlineBackground
	if account.session.IsDisconnected() {
		bgcolor = cs.rosterAccountOfflineBackground
	}
	r.fields.backgroundColor.set(parentIter, bgcolor.toCSS())

	parentPath, _ := r.model.GetPath(parentIter)
	shouldCollapse, ok := r.isCollapsed[collapseTransform(parentName)]
	isExpanded := true
	if ok && shouldCollapse {
		isExpanded = false
		r.toCollapse = append(r.toCollapse, parentPath)
	}
	var stat string
	if account.session.IsDisconnected() {
		stat = "offline"
	} else if account.session.IsConnected() {
		stat = "available"
	} else {
		stat = "connecting"
	}

	r.fields.statusIcon.set(parentIter, statusIcons[stat].GetPixbuf())
	r.fields.displayName.set(parentIter, createGroupDisplayName(parentName, accountCounter, isExpanded))
}

func (r *roster) sortedAccounts() []*account {
	var as []*account
	for account := range r.ui.accountManager.getAllContacts() {
		if account == nil {
			r.ui.hasLog.log.Warn("adding an account that is nil...")
		}
		as = append(as, account)
	}
	//TODO sort by nickname if available
	sort.Sort(byAccountNameAlphabetic(as))
	return as
}

func (r *roster) showEmptyGroups() bool {
	return r.ui.settings.GetShowEmptyGroups()
}

func (r *roster) redrawSeparate() {
	showOffline := !r.ui.config().Display.ShowOnlyOnline
	showWaiting := !r.ui.config().Display.ShowOnlyConfirmed

	r.ui.accountManager.lock.RLock()
	defer r.ui.accountManager.lock.RUnlock()

	r.toCollapse = nil

	for _, account := range r.sortedAccounts() {
		r.redrawSeparateAccount(account, r.ui.accountManager.getContacts(account), showOffline, showWaiting)
	}

	r.view.ExpandAll()
	for _, path := range r.toCollapse {
		r.view.CollapseRow(path)
	}
}

func (r *roster) redraw() {
	//TODO: this should be behind a mutex
	r.model.Clear()

	if r.ui.shouldViewAccounts() {
		r.redrawSeparate()
	} else {
		r.redrawMerged()
	}
}
