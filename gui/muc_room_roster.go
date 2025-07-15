package gui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/coyim/gotk3adapter/gdki"
	log "github.com/sirupsen/logrus"

	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/i18n"
	coyroster "github.com/chadsec1/decoyim/roster"
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/chadsec1/decoyim/session/muc/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/glibi"
	"github.com/coyim/gotk3adapter/gtki"
)

const (
	roomViewRosterGroupCollapseIconName = "pan-down-symbolic"
	roomViewRosterGroupExpandIconName   = "pan-end-symbolic"
)

const (
	roomViewRosterFontWeightNormal = 400
	roomViewRosterFontWeightBold   = 700
)

const (
	roomViewRosterImageIndex int = iota
	roomViewRosterNicknameIndex
	roomViewRosterAffiliationIndex
	roomViewRosterInfoIndex
	roomViewRosterFontWeightIndex
	roomViewRosterForegroundIndex
	roomViewRosterBackgroundIndex
	roomViewRosterOccupantRoleForegroundIndex
	roomViewRosterOccupantImageVisibilityIndex
	roomViewRosterOccupantAffiliationVisibilityIndex
	roomViewRosterTextDisplayIndex
	roomViewRosterExpanderIconIndex
	roomViewRosterExpanderVisibilityIndex
)

type roomViewRoster struct {
	u          *gtkUI
	roomView   *roomView
	rosterInfo *roomViewRosterInfo

	roster  *muc.RoomRoster
	account *account
	roomID  jid.Bare

	view        gtki.Box      `gtk-widget:"roster-view"`
	rosterPanel gtki.Box      `gtk-widget:"roster-main-panel"`
	tree        gtki.TreeView `gtk-widget:"roster-tree-view"`

	model gtki.TreeStore

	log coylog.Logger
}

func (v *roomView) newRoomViewRoster() *roomViewRoster {
	r := &roomViewRoster{
		u:        v.u,
		roomView: v,
		roster:   v.room.Roster(),
		account:  v.account,
		roomID:   v.roomID(),
		log:      v.log,
	}

	r.initBuilder()
	r.initDefaults()
	r.initSubscribers()

	return r
}

func (r *roomViewRoster) initBuilder() {
	builder := newBuilder("MUCRoomRoster")
	builder.ConnectSignals(map[string]interface{}{
		"on_occupant_tree_view_row_activated": r.onOccupantRowActivated,
	})

	panicOnDevError(builder.bindObjects(r))
}

func (r *roomViewRoster) initDefaults() {
	r.rosterInfo = r.newRoomViewRosterInfo()

	r.model, _ = g.gtk.TreeStoreNew(
		// status icon or opened/closed image
		pixbufType(),
		// display nickname
		glibi.TYPE_STRING,
		// affiliation
		glibi.TYPE_STRING,
		// info tooltip
		glibi.TYPE_STRING,
		// font weight
		glibi.TYPE_INT,
		// foreground color
		glibi.TYPE_STRING,
		// background color
		glibi.TYPE_STRING,
		// occupant role foreground color
		glibi.TYPE_STRING,
		// occupant image visibility
		glibi.TYPE_BOOLEAN,
		// occupant affiliation visibility
		glibi.TYPE_BOOLEAN,
		// text display
		glibi.TYPE_STRING,
		// expander icon name
		glibi.TYPE_STRING,
		// expander icon visibility
		glibi.TYPE_BOOLEAN,
	)

	r.tree.SetModel(r.model)
	r.draw()
}

func (r *roomViewRoster) initSubscribers() {
	r.roomView.subscribe("roster", func(ev roomViewEvent) {
		switch ev.(type) {
		case selfOccupantJoinedEvent,
			occupantJoinedEvent,
			occupantUpdatedEvent,
			occupantLeftEvent,
			selfOccupantRemovedEvent,
			occupantRemovedEvent,
			selfOccupantRoleUpdatedEvent,
			occupantRoleUpdatedEvent,
			selfOccupantAffiliationUpdatedEvent,
			occupantAffiliationUpdatedEvent,
			selfOccupantReconnectedEvent,
			selfOccupantRemovedOnAffiliationChangeEvent,
			occupantRemovedOnAffiliationChangeEvent:
			r.onUpdateRoster()
		case roomDisableEvent:
			r.roomDisableEvent()
		case roomEnableEvent:
			r.roomEnableEvent()
		}
	})
}

// onOccupantSelected MUST be called from the UI thread
func (r *roomViewRoster) onOccupantSelected(_ gtki.TreeView, path gtki.TreePath) {
	nickname, err := r.getNicknameFromTreeModel(path)
	if err != nil {
		r.log.Debug("Occupant nickname not found in the roster model")
		return
	}

	o, ok := r.occupantFromRoster(nickname)
	if !ok {
		r.log.WithField("nickname", nickname).Debug("The occupant was not found in the roster")
		return
	}

	r.showOccupantInfo(o)
}

// occupantFromRoster MUST NOT be called from the UI thread
func (r *roomViewRoster) occupantFromRoster(nickname string) (*muc.Occupant, bool) {
	return r.roster.GetOccupant(nickname)
}

// onGroupActivated MUST be called from the UI thread
func (r *roomViewRoster) onGroupActivated(_ gtki.TreeView, path gtki.TreePath) {
	var icon string

	if r.tree.RowExpanded(path) {
		r.tree.CollapseRow(path)
		icon = roomViewRosterGroupExpandIconName
	} else {
		r.tree.ExpandRow(path, true)
		icon = roomViewRosterGroupCollapseIconName
	}

	if iter, err := r.model.GetIter(path); err == nil {
		_ = r.model.SetValue(iter, roomViewRosterExpanderIconIndex, icon)
	}
}

const (
	roomViewRosterGroupDepth    = 1
	roomViewRosterOccupantDepth = 2
)

// onOccupantRowActivated MUST be called from the UI thread
func (r *roomViewRoster) onOccupantRowActivated(tree gtki.TreeView, path gtki.TreePath) {
	switch path.GetDepth() {
	case roomViewRosterGroupDepth:
		r.onGroupActivated(tree, path)
	case roomViewRosterOccupantDepth:
		r.onOccupantSelected(tree, path)
	}
}

// updateOccupantAffiliation MUST NOT be called from the UI thread
func (r *roomViewRoster) updateOccupantAffiliation(o *muc.Occupant, previousAffiliation data.Affiliation, reason string) {
	r.log.WithFields(log.Fields{
		"where":       "updateOccupantAffiliation",
		"occupant":    fmt.Sprintf("%s", o.RealJid),
		"affiliation": o.Affiliation.Name(),
	}).Info("The occupant affiliation is going to be updated")

	r.roomView.tryUpdateOccupantAffiliation(o, previousAffiliation, reason)
}

// updateOccupantRole MUST NOT be called from the UI thread
func (r *roomViewRoster) updateOccupantRole(o *muc.Occupant, newRole data.Role, reason string) {
	r.log.WithFields(log.Fields{
		"where":    "updateOccupantRole",
		"occupant": o.Nickname,
		"role":     o.Role.Name(),
	}).Info("The occupant role is going to be updated")

	r.roomView.tryUpdateOccupantRole(o, newRole, reason)
}

// showOccupantInfo MUST be called from the UI thread
func (r *roomViewRoster) showOccupantInfo(o *muc.Occupant) {
	r.rosterInfo.showOccupantInfo(o)
	r.showRosterInfoPanel()
}

// showRosterInfoPanel MUST be called from the UI thread
func (r *roomViewRoster) showRosterInfoPanel() {
	r.rosterPanel.Hide()
	r.view.Add(r.rosterInfo.view)
}

// hideRosterInfoPanel MUST be called from the UI thread
func (r *roomViewRoster) hideRosterInfoPanel() {
	r.view.Remove(r.rosterInfo.view)
	r.rosterPanel.Show()
}

func (r *roomViewRoster) getNicknameFromTreeModel(path gtki.TreePath) (string, error) {
	iter, err := r.model.GetIter(path)
	if err != nil {
		r.log.WithError(err).Error("Couldn't activate the selected item")
		return "", err
	}

	iterValue, err := r.model.GetValue(iter, roomViewRosterNicknameIndex)
	if err != nil {
		return "", errors.New("error trying to get iter value")
	}

	return iterValue.GetString()
}

func (r *roomViewRoster) onUpdateRoster() {
	doInUIThread(r.redraw)
}

func (r *roomViewRoster) roomSelfOccupant() *muc.Occupant {
	return r.roomView.room.SelfOccupant()
}

func (r *roomViewRoster) isSelfOccupantInTheRoom() bool {
	return r.roomView.isSelfOccupantInTheRoom()
}

func (r *roomViewRoster) draw() {
	noneRoles, visitors, participants, moderators := r.roster.OccupantsByRole()

	r.drawOccupantsByRole(data.RoleModerator, moderators)
	r.drawOccupantsByRole(data.RoleParticipant, participants)
	r.drawOccupantsByRole(data.RoleVisitor, visitors)
	r.drawOccupantsByRole(data.RoleNone, noneRoles)

	r.tree.ExpandAll()
}

func (r *roomViewRoster) redraw() {
	r.model.Clear()
	r.draw()
}

func (r *roomViewRoster) drawOccupantsByRole(role string, occupants []*muc.Occupant) {
	if len(occupants) == 0 {
		return
	}

	roleHeader := rolePluralName(role)
	roleHeader = i18n.Localf("%s (%v)", roleHeader, len(occupants))

	iter := r.model.Append(nil)

	cs := r.u.unifiedCached.ui.currentMUCColorSet()

	modelSetValues(r.model, iter, map[int]interface{}{
		roomViewRosterTextDisplayIndex:                   roleHeader,
		roomViewRosterFontWeightIndex:                    roomViewRosterFontWeightBold,
		roomViewRosterBackgroundIndex:                    cs.rosterGroupBackground.toCSS(),
		roomViewRosterForegroundIndex:                    cs.rosterGroupForeground.toCSS(),
		roomViewRosterExpanderIconIndex:                  roomViewRosterGroupCollapseIconName,
		roomViewRosterExpanderVisibilityIndex:            true,
		roomViewRosterOccupantImageVisibilityIndex:       false,
		roomViewRosterOccupantAffiliationVisibilityIndex: false,
	})

	for _, o := range occupants {
		r.addOccupantToRoster(o, iter)
	}
}

func (r *roomViewRoster) addOccupantToRoster(o *muc.Occupant, parentIter gtki.TreeIter) {
	iter := r.model.Append(parentIter)

	cs := r.u.currentMUCColorSet()

	displayAffiliation := affiliationDisplayName(o.Affiliation)

	nickname := o.Nickname
	displayNickname := nickname
	nicknameFontWeight := roomViewRosterFontWeightNormal

	if o.Nickname == r.roomView.room.SelfOccupantNickname() {
		displayNickname = i18n.Localf("%s (You)", nickname)
		nicknameFontWeight = roomViewRosterFontWeightBold
	}

	modelSetValues(r.model, iter, map[int]interface{}{
		roomViewRosterImageIndex:                         getOccupantIconForStatus(o.Status),
		roomViewRosterNicknameIndex:                      nickname,
		roomViewRosterTextDisplayIndex:                   displayNickname,
		roomViewRosterAffiliationIndex:                   displayAffiliation,
		roomViewRosterInfoIndex:                          occupantDisplayTooltip(o),
		roomViewRosterFontWeightIndex:                    nicknameFontWeight,
		roomViewRosterOccupantRoleForegroundIndex:        cs.rosterOccupantRoleForeground.toCSS(),
		roomViewRosterOccupantImageVisibilityIndex:       true,
		roomViewRosterOccupantAffiliationVisibilityIndex: displayAffiliation != "",
		roomViewRosterExpanderVisibilityIndex:            false,
	})
}

// roomDisableEvent MUST be called from the UI thread
func (r *roomViewRoster) roomDisableEvent() {
	addRoomDisableClass(r.tree)
	addRoomDisableClass(r.rosterPanel)
}

// roomEnableEvent MUST be called from the UI thread
func (r *roomViewRoster) roomEnableEvent() {
	removeRoomDisableClass(r.tree)
	removeRoomDisableClass(r.rosterPanel)
}

// parentWindow MUST be called from the UI threads
func (r *roomViewRoster) parentWindow() gtki.Window {
	return r.roomView.mainWindow()
}

func getOccupantIconForStatus(s *coyroster.Status) gdki.Pixbuf {
	icon := getOccupantIconNameForStatus(s.Status)
	return getMUCIconPixbuf(icon)
}

func getOccupantIconNameForStatus(status string) string {
	switch status {
	case "unavailable":
		return "occupant-offline"
	case "away":
		return "occupant-away"
	case "dnd":
		return "occupant-busy"
	case "xa":
		return "occupant-extended-away"
	case "chat":
		return "occupant-chat"
	default:
		return "occupant-online"
	}
}

func affiliationDisplayName(affiliation data.Affiliation) string {
	switch {
	case affiliation.IsAdmin():
		return i18n.Local("Admin")
	case affiliation.IsOwner():
		return i18n.Local("Owner")
	case affiliation.IsBanned():
		return i18n.Local("Outcast")
	default: // Member or other values get the default treatment
		return ""
	}
}

func rolePluralName(role string) string {
	switch role {
	case data.RoleNone:
		return i18n.Local("None")
	case data.RoleParticipant:
		return i18n.Local("Participants")
	case data.RoleVisitor:
		return i18n.Local("Visitors")
	case data.RoleModerator:
		return i18n.Local("Moderators")
	default:
		// This should not really be possible, but it is necessary
		// because golang can't prove it
		return ""
	}
}

func roleDisplayName(role data.Role) string {
	switch role.Name() {
	case data.RoleNone:
		return i18n.Local("None")
	case data.RoleParticipant:
		return i18n.Local("Participant")
	case data.RoleVisitor:
		return i18n.Local("Visitor")
	case data.RoleModerator:
		return i18n.Local("Moderator")
	default:
		// This should not really be possible, but it is necessary
		// because golang can't prove it
		return ""
	}
}

func getOccupantStatusClassName(status string) string {
	switch status {
	case "unavailable":
		return "not-available"
	case "away":
		return "away"
	case "dnd":
		return "busy"
	case "xa":
		return "extended-away"
	case "chat":
		return "free-for-chat"
	}
	return "available"
}

func statusDisplayMessage(s *coyroster.Status) string {
	return s.StatusMsg
}

func statusDisplayName(s *coyroster.Status) string {
	return showForDisplay(s.Status, false)
}

func occupantDisplayTooltip(o *muc.Occupant) string {
	ms := []string{
		o.Nickname,
		statusDisplayName(o.Status),
	}

	m := statusDisplayMessage(o.Status)
	if m != "" {
		ms = append(ms, m)
	}

	return strings.Join(ms, "\n")
}

func modelSetValues(model gtki.TreeStore, iter gtki.TreeIter, values map[int]interface{}) {
	for idx, v := range values {
		_ = model.SetValue(iter, idx, v)
	}
}
