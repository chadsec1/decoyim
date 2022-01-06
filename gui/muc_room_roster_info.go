package gui

import (
	"fmt"

	"github.com/coyim/coyim/coylog"
	"github.com/coyim/coyim/i18n"
	"github.com/coyim/coyim/session/muc"
	"github.com/coyim/coyim/session/muc/data"
	"github.com/coyim/coyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gtki"
)

const (
	changeAffiliationActionName = "change-affiliation-listbox-row"
	changeRoleActionName        = "change-role-listbox-row"
)

type roomViewRosterInfo struct {
	u *gtkUI

	account      *account
	roomID       jid.Bare
	occupant     *muc.Occupant
	selfOccupant *muc.Occupant
	rosterView   *roomViewRoster

	view                      gtki.Box        `gtk-widget:"roster-info-box"`
	avatar                    gtki.Image      `gtk-widget:"avatar-image"`
	nicknameLabel             gtki.Label      `gtk-widget:"nickname-label"`
	status                    gtki.Label      `gtk-widget:"status-label"`
	currentAffiliationLabel   gtki.Label      `gtk-widget:"current-affiliation"`
	currentRoleLabel          gtki.Label      `gtk-widget:"current-role"`
	affiliationListBoxRow     gtki.ListBoxRow `gtk-widget:"change-affiliation-listbox-row"`
	affiliationActionImage    gtki.Image      `gtk-widget:"change-affiliation-action-image"`
	roleListBoxRow            gtki.ListBoxRow `gtk-widget:"change-role-listbox-row"`
	roleActionImage           gtki.Image      `gtk-widget:"change-role-action-image"`
	roleDisableLabel          gtki.Label      `gtk-widget:"change-role-disabled"`
	occupantActionsMenuButton gtki.MenuButton `gtk-widget:"occupant-actions-menu-button"`
	banOccupantMenuItem       gtki.MenuItem   `gtk-widget:"ban-occupant"`
	kickOccupantMenuItem      gtki.MenuItem   `gtk-widget:"kick-occupant"`
	nicknamePopoverLabel      gtki.Label      `gtk-widget:"nickname-popover-label"`
	realJidPopoverBox         gtki.Box        `gtk-widget:"user-jid-popover-box"`
	realJidPopoverLabel       gtki.Label      `gtk-widget:"user-jid-popover-label"`
	statusPopoverLabel        gtki.Label      `gtk-widget:"status-popover-label"`
	statusMessagePopoverBox   gtki.Box        `gtk-widget:"status-message-popover-box"`
	statusMessagePopoverLabel gtki.Label      `gtk-widget:"status-message-popover-label"`

	onReset   *callbacksSet
	onRefresh *callbacksSet

	log coylog.Logger
}

func (r *roomViewRoster) newRoomViewRosterInfo() *roomViewRosterInfo {
	ri := &roomViewRosterInfo{
		u:          r.u,
		account:    r.account,
		roomID:     r.roomID,
		rosterView: r,
		onReset:    newCallbacksSet(),
		onRefresh:  newCallbacksSet(),
		log:        r.log,
	}

	ri.initBuilder()
	ri.initSubscribers()
	ri.initCSSStyles()
	ri.initDefaults()

	return ri
}

func (r *roomViewRosterInfo) initBuilder() {
	builder := newBuilder("MUCRoomRosterInfo")
	panicOnDevError(builder.bindObjects(r))

	builder.ConnectSignals(map[string]interface{}{
		"on_hide":            r.hide,
		"on_occupant_action": r.onOccupantActionClicked,
		"on_ban":             r.onBanOccupantClicked,
		"on_kick":            r.onKickOccupantClicked,
	})
}

func (r *roomViewRosterInfo) initSubscribers() {
	r.rosterView.roomView.subscribe("roster-info", func(ev roomViewEvent) {
		switch t := ev.(type) {
		case selfOccupantJoinedEvent, selfOccupantRemovedEvent, selfOccupantReconnectedEvent, selfOccupantRemovedOnAffiliationChangeEvent:
			r.selfOccupantUpdateEvent()
		case occupantUpdatedEvent:
			r.occupantUpdatedEvent(t.nickname)
		case occupantRemovedEvent:
			r.occupantRemovedEvent(t.nickname)
		case occupantLeftEvent:
			r.occupantLeftEvent(t.nickname)
		case occupantRemovedOnAffiliationChangeEvent:
			r.occupantUpdatedEvent(t.nickname)
		case roomDisableEvent:
			r.roomDisableEvent()
		case roomEnableEvent:
			r.roomEnableEvent()
		}
	})
}

func (r *roomViewRosterInfo) initCSSStyles() {
	mucStyles.setRoomRosterInfoStyle(r.view)
}

func (r *roomViewRosterInfo) initDefaults() {
	r.onRefresh.add(
		r.refreshOccupantInfo,
		r.refreshOccupantAffiliation,
		r.refreshOccupantRole,
		r.validateOccupantPrivileges,
	)

	r.onReset.add(
		r.removeOccupantInfo,
		r.removeOccupantAffiliationInfo,
		r.removeOccupantRoleInfo,
		r.validateOccupantPrivileges,
	)

	if r.rosterView.isSelfOccupantInTheRoom() {
		r.selfOccupant = r.rosterView.roomSelfOccupant()
	}
}

// onOccupantAction MUST be called from the UI thread
func (r *roomViewRosterInfo) onOccupantActionClicked(_ gtki.ListBox, row gtki.ListBoxRow) {
	switch name, _ := row.GetName(); name {
	case changeAffiliationActionName:
		r.onChangeAffiliation()
	case changeRoleActionName:
		r.onChangeRole()
	}
}

// onBanOccupantClicked MUST be called from the UI thread
func (r *roomViewRosterInfo) onBanOccupantClicked() {
	kd := r.rosterView.newBanOccupantView(r.occupant)
	kd.show()
}

// onKickOccupantClicked MUST be called from the UI thread
func (r *roomViewRosterInfo) onKickOccupantClicked() {
	kd := r.rosterView.newKickOccupantView(r.occupant)
	kd.show()
}

func (r *roomViewRosterInfo) roleToUpdate() data.Role {
	if r.occupant.HasVoice() {
		return &data.VisitorRole{}
	}
	return &data.ParticipantRole{}
}

// onOccupantUpdated MUST NOT be called from the UI thread
func (r *roomViewRosterInfo) onOccupantUpdated() {
	doInUIThread(r.refresh)
}

// updateOccupantAffiliation MUST NOT be called from the UI thread
func (r *roomViewRosterInfo) updateOccupantAffiliation(occupant *muc.Occupant, previousAffiliation data.Affiliation, reason string) {
	r.rosterView.updateOccupantAffiliation(occupant, previousAffiliation, reason)
	r.onOccupantUpdated()
}

// updateOccupantRole MUST NOT be called from the UI thread
func (r *roomViewRosterInfo) updateOccupantRole(occupant *muc.Occupant, role data.Role, reason string) {
	r.rosterView.updateOccupantRole(occupant, role, reason)
	r.onOccupantUpdated()
}

// showOccupantInfo MUST be called from the UI thread
func (r *roomViewRosterInfo) showOccupantInfo(occupant *muc.Occupant) {
	r.occupant = occupant

	r.rosterView.roomView.subscribe("roster-occupant-info", func(ev roomViewEvent) {
		switch e := ev.(type) {
		case occupantUpdatedEvent:
			if e.nickname == r.occupant.Nickname {
				r.onOccupantUpdated()
			}
		}
	})

	r.refresh()
	r.show()
}

// removeOccupantEvents MUST be called from the UI thread
func (r *roomViewRosterInfo) removeOccupantEvents() {
	r.rosterView.roomView.unsubscribe("roster-occupant-info")
}

// validateOccupantPrivileges MUST be called from the UI thread
func (r *roomViewRosterInfo) validateOccupantPrivileges() {
	r.refreshAffiliationSection()
	r.refreshRoleSection()
	r.refreshAdminToolsSection()
}

// refreshAffiliationSection MUST be called from the UI thread
func (r *roomViewRosterInfo) refreshAffiliationSection() {
	canChangeAffiliation := r.selfOccupant.CanChangeAffiliation(r.occupant)
	r.affiliationListBoxRow.SetProperty("activatable", canChangeAffiliation)
	r.affiliationListBoxRow.SetCanFocus(canChangeAffiliation)
	r.affiliationActionImage.SetVisible(canChangeAffiliation)
}

// refreshRoleSection MUST be called from the UI thread
func (r *roomViewRosterInfo) refreshRoleSection() {
	canChangeRole := r.selfOccupant.CanChangeRole(r.occupant)
	r.roleListBoxRow.SetProperty("activatable", canChangeRole)
	r.roleListBoxRow.SetCanFocus(canChangeRole)
	r.roleActionImage.SetVisible(canChangeRole)

	showChangeRoleDisabledLabel := r.selfOccupant.Affiliation.IsOwner() && (r.occupant.Affiliation.IsOwner() || r.occupant.Affiliation.IsAdmin())
	r.roleDisableLabel.SetText(i18n.Localf("Administrators and owners will automatically be moderators for a room. "+
		"Therefore, the role of %s can't be changed.", r.occupant.Nickname))
	r.roleDisableLabel.SetVisible(showChangeRoleDisabledLabel)
}

// refreshAdminToolsSection MUST be called from the UI thread
func (r *roomViewRosterInfo) refreshAdminToolsSection() {
	canKick := r.selfOccupant.CanKickOccupant(r.occupant)
	r.kickOccupantMenuItem.SetVisible(canKick)

	canBan := r.selfOccupant.CanBanOccupant(r.occupant)
	r.banOccupantMenuItem.SetVisible(canBan)

	r.occupantActionsMenuButton.SetSensitive(canKick || canBan)
}

// refresh MUST be called from the UI thread
func (r *roomViewRosterInfo) refresh() {
	r.reset()
	r.onRefresh.invokeAll()
}

// reset MUST be called from the UI thread
func (r *roomViewRosterInfo) reset() {
	r.onReset.invokeAll()
}

// refresh MUST be called from the UI thread
func (r *roomViewRosterInfo) refreshOccupantInfo() {
	occupant := r.occupant
	status := r.occupant.Status

	avatar := fmt.Sprintf("%s-large", getOccupantIconNameForStatus(status.Status))
	r.avatar.SetFromPixbuf(getMUCIconPixbuf(avatar))
	setLabelText(r.nicknameLabel, occupant.Nickname)
	setLabelText(r.nicknamePopoverLabel, occupant.Nickname)

	r.realJidPopoverBox.SetVisible(false)
	if occupant.RealJid != nil {
		r.realJidPopoverLabel.SetText(occupant.RealJid.String())
		r.realJidPopoverBox.SetVisible(true)
	}

	statusDisplay := showForDisplay(status.Status, false)
	r.status.SetText(statusDisplay)
	r.statusPopoverLabel.SetText(statusDisplay)

	r.statusMessagePopoverBox.SetVisible(false)
	if status.StatusMsg != "" {
		r.statusMessagePopoverLabel.SetText(status.StatusMsg)
		r.statusMessagePopoverLabel.SetVisible(true)
		r.statusMessagePopoverBox.SetVisible(true)
	}
}

// removeOccupantInfo MUST be called from the UI thread
func (r *roomViewRosterInfo) removeOccupantInfo() {
	r.avatar.Clear()

	r.nicknameLabel.SetText("")

	r.realJidPopoverLabel.SetText("")
	r.realJidPopoverBox.SetVisible(false)

	r.statusMessagePopoverLabel.SetText("")
	r.statusMessagePopoverLabel.SetVisible(false)
}

// refreshOccupantAffiliation MUST be called from the UI thread
func (r *roomViewRosterInfo) refreshOccupantAffiliation() {
	r.currentAffiliationLabel.SetText(occupantAffiliationName(r.occupant.Affiliation))
}

// removeOccupantAffiliationInfo MUST be called from the UI thread
func (r *roomViewRosterInfo) removeOccupantAffiliationInfo() {
	r.currentAffiliationLabel.SetText("")
}

// refreshOccupantAffiliation MUST be called from the UI thread
func (r *roomViewRosterInfo) refreshOccupantRole() {
	r.currentRoleLabel.SetText(occupantRoleName(r.occupant.Role))
}

// removeOccupantAffiliationInfo MUST be called from the UI thread
func (r *roomViewRosterInfo) removeOccupantRoleInfo() {
	r.currentRoleLabel.SetText("")
}

// show MUST be called from the UI thread
func (r *roomViewRosterInfo) show() {
	r.view.Show()
}

// show MUST be called from the UI thread
func (r *roomViewRosterInfo) hide() {
	r.view.Hide()
	r.rosterView.hideRosterInfoPanel()
	r.removeOccupantEvents()
	r.reset()
}

// isOpen MUST be called from the UI thread
func (r *roomViewRosterInfo) isOpen() bool {
	return r.occupant != nil && r.view.IsVisible()
}

func (r *roomViewRosterInfo) isTheSameOccupant(nickname string) bool {
	o, ok := r.rosterView.occupantFromRoster(nickname)
	return ok && r.occupant == o
}

func (r *roomViewRosterInfo) selfOccupantUpdateEvent() {
	r.selfOccupant = r.rosterView.roomSelfOccupant()

	if r.isOpen() {
		r.validateOccupantPrivileges()
	}
}

func (r *roomViewRosterInfo) occupantUpdatedEvent(nickname string) {
	r.onOccupantUpdate(nickname)
}

func (r *roomViewRosterInfo) occupantRemovedEvent(nickname string) {
	r.onOccupantUpdate(nickname)
}

func (r *roomViewRosterInfo) occupantLeftEvent(nickname string) {
	r.onOccupantUpdate(nickname)
}

func (r *roomViewRosterInfo) onOccupantUpdate(nickname string) {
	if r.isOpen() && r.isTheSameOccupant(nickname) {
		r.validateOccupantPrivileges()
	}
}

func (r *roomViewRosterInfo) roomDisableEvent() {
	r.affiliationListBoxRow.SetSensitive(false)
	r.roleListBoxRow.SetSensitive(false)

	if r.isOpen() {
		r.validateOccupantPrivileges()
	}
}

func (r *roomViewRosterInfo) roomEnableEvent() {
	r.affiliationListBoxRow.SetSensitive(true)
	r.roleListBoxRow.SetSensitive(true)

	if r.isOpen() {
		r.validateOccupantPrivileges()
	}
}

// parentWindow MUST be called from the UI threads
func (r *roomViewRosterInfo) parentWindow() gtki.Window {
	return r.rosterView.parentWindow()
}
