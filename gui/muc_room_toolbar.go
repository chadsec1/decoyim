package gui

import (
	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/session/muc/data"
	"github.com/coyim/gotk3adapter/gtki"
)

const (
	roomSubjectShownIconName  = "go-up-symbolic"
	roomSubjectHiddenIconName = "go-down-symbolic"
)

const (
	showSubjectButton = iota
	showEditSubjectButton
	showEditSubjectComponent
	showSubjectLabel
)

type editSubjectContext struct {
	canChangeSubject bool
	existsSubject    bool
}

var editSubjectComponentRules = map[editSubjectContext][]bool{
	{true /*canChangeSubject*/, true /*existsSubject*/}:   {true /*showSubjectButton*/, true /*showEditSubjectButton*/, false /*showEditSubjectComponent*/, true /*showSubjectLabel*/},
	{true /*canChangeSubject*/, false /*existsSubject*/}:  {true /*showSubjectButton*/, false /*showEditSubjectButton*/, true /*showEditSubjectComponent*/, false /*showSubjectLabel*/},
	{false /*canChangeSubject*/, true /*existsSubject*/}:  {true /*showSubjectButton*/, false /*showEditSubjectButton*/, false /*showEditSubjectComponent*/, true /*showSubjectLabel*/},
	{false /*canChangeSubject*/, false /*existsSubject*/}: {false /*showSubjectButton*/, false /*showEditSubjectButton*/, false /*showEditSubjectComponent*/, false /*showSubjectLabel*/},
}

type roomViewToolbar struct {
	roomView         *roomView
	isEditingSubject bool

	view                        gtki.Box            `gtk-widget:"room-view-toolbar"`
	roomNameLabel               gtki.Label          `gtk-widget:"room-name-label"`
	roomStatusIcon              gtki.Image          `gtk-widget:"room-status-icon"`
	roomMenuButton              gtki.MenuButton     `gtk-widget:"room-menu-button"`
	roomSubjectButton           gtki.Button         `gtk-widget:"room-subject-button"`
	roomSubjectButtonImage      gtki.Image          `gtk-widget:"room-subject-button-image"`
	roomSubjectRevealer         gtki.Revealer       `gtk-widget:"room-subject-revealer"`
	roomSubjectLabel            gtki.Label          `gtk-widget:"room-subject-label"`
	roomSubjectScrolledWindow   gtki.ScrolledWindow `gtk-widget:"room-subject-editable-content"`
	roomSubjectTextView         gtki.TextView       `gtk-widget:"room-subject-textview"`
	roomSubjectButtonsContainer gtki.Box            `gtk-widget:"room-edit-subject-buttons-container"`
	roomSubjectEditButton       gtki.Button         `gtk-widget:"room-edit-subject-button"`
	roomSubjectApplyButton      gtki.Button         `gtk-widget:"room-edit-subject-apply-button"`
	configureRoomMenuItem       gtki.MenuItem       `gtk-widget:"room-configuration-menu-item"`
	modifyPositionListsMenuItem gtki.MenuItem       `gtk-widget:"modify-position-lists-menu-item"`
	destroyRoomMenuItem         gtki.MenuItem       `gtk-widget:"destroy-room-menu-item"`
}

func (v *roomView) newRoomViewToolbar() *roomViewToolbar {
	t := &roomViewToolbar{
		roomView: v,
	}

	t.initBuilder()
	t.initDefaults()
	t.initSubscribers()

	return t
}

func (t *roomViewToolbar) initBuilder() {
	builder := newBuilder("MUCRoomToolbar")
	panicOnDevError(builder.bindObjects(t))

	builder.ConnectSignals(map[string]interface{}{
		"on_edit_room_subject":        t.onEditRoomSubject,
		"on_cancel_room_subject_edit": t.onCancelEditSubject,
		"on_apply_room_subject_edit":  t.onApplyEditSubject,
		"on_subject_changed":          t.onRoomSubectChanged,
		"on_toggle_room_subject":      t.onToggleRoomSubject,
		"on_show_security_properties": t.roomView.showWarnings,
		"on_configure_room":           t.roomView.onConfigureRoom,
		"on_modify_position_lists":    t.roomView.onRoomPositionsView,
		"on_destroy_room":             t.roomView.onDestroyRoom,
		"on_leave_room":               t.roomView.onLeaveRoom,
	})
}

func (t *roomViewToolbar) initDefaults() {
	t.roomStatusIcon.SetFromPixbuf(getMUCIconPixbuf("room"))

	t.roomNameLabel.SetText(t.roomView.roomID().String())
	mucStyles.setRoomToolbarNameLabelStyle(t.roomNameLabel)

	mucStyles.setRoomToolbarSubjectLabelStyle(t.roomSubjectLabel)

	if t.roomView.room.IsSelfOccupantInTheRoom() {
		t.updateMenuActionsBasedOn(t.roomView.room.SelfOccupant().Affiliation)
	}
}

func (t *roomViewToolbar) initSubscribers() {
	t.roomView.subscribe("toolbar", func(ev roomViewEvent) {
		switch e := ev.(type) {
		case subjectReceivedEvent:
			t.subjectReceivedEvent(e.subject)
		case subjectUpdatedEvent:
			t.subjectUpdatedEvent(e.subject)
		case roomConfigChangedEvent:
			t.onRoomConfigChanged()
		case selfOccupantJoinedEvent, selfOccupantReconnectedEvent:
			t.selfOccupantJoinedEvent()
		case selfOccupantRoleUpdatedEvent:
			t.selfOccupantRoleUpdatedEvent(e.selfRoleUpdate.New)
		case selfOccupantAffiliationUpdatedEvent:
			t.selfOccupantAffiliationUpdatedEvent(e.selfAffiliationUpdate.New)
		case selfOccupantAffiliationRoleUpdatedEvent:
			t.selfOccupantRoleUpdatedEvent(e.selfAffiliationRoleUpdate.NewRole)
			t.selfOccupantAffiliationUpdatedEvent(e.selfAffiliationRoleUpdate.NewAffiliation)
		case roomDisableEvent, roomDestroyedEvent, selfOccupantRemovedEvent:
			t.roomDisableEvent()
		case roomEnableEvent:
			t.roomEnableEvent()
		case reopenRoomEvent:
			t.subjectReceivedEvent(e.subject)
		case selfOccupantDisconnectedEvent:
			t.selfOccupantDisconnectedEvent()
		}
	})
}

func (t *roomViewToolbar) subjectReceivedEvent(subject string) {
	doInUIThread(func() {
		t.displayRoomSubject(subject)
		t.handleEditSubjectComponents()
	})
}

func (t *roomViewToolbar) subjectUpdatedEvent(subject string) {
	t.subjectReceivedEvent(subject)
	if !t.roomView.room.HasSubject() {
		doInUIThread(t.onHideRoomSubject)
	}
}

func (t *roomViewToolbar) onRoomConfigChanged() {
	doInUIThread(t.onEditSubjectContextChanged)
}

// onEditSubjectContextChanged MUST be called from the UI thread
func (t *roomViewToolbar) onEditSubjectContextChanged() {
	t.handleEditSubjectComponents()
	if !t.roomView.room.SubjectCanBeChanged() {
		if !t.roomView.room.HasSubject() {
			t.onHideRoomSubject()
		}
		if getTextViewText(t.roomSubjectTextView) != "" {
			setTextViewText(t.roomSubjectTextView, "")
			t.roomView.notifications.warning(roomNotificationOptions{
				message:   i18n.Local("You are no longer allowed to continue updating the room subject."),
				closeable: true,
			})
		}
	}
}

func (t *roomViewToolbar) selfOccupantRoleUpdatedEvent(role data.Role) {
	doInUIThread(t.onEditSubjectContextChanged)

	if role.IsNone() {
		doInUIThread(t.roomDisableEvent)
	}
}

func (t *roomViewToolbar) selfOccupantAffiliationUpdatedEvent(affiliation data.Affiliation) {
	doInUIThread(func() {
		t.updateMenuActionsBasedOn(affiliation)
	})

	if affiliation.IsBanned() {
		doInUIThread(t.roomDisableEvent)
	}
}

func (t *roomViewToolbar) selfOccupantJoinedEvent() {
	doInUIThread(func() {
		t.updateMenuActionsBasedOn(t.roomView.room.SelfOccupant().Affiliation)
	})
}

func (t *roomViewToolbar) updateMenuActionsBasedOn(affiliation data.Affiliation) {
	t.configureRoomMenuItem.SetVisible(affiliation.IsOwner())
	t.destroyRoomMenuItem.SetVisible(affiliation.IsOwner())
	t.modifyPositionListsMenuItem.SetVisible(affiliation.IsAdmin())
}

// displayRoomSubject MUST be called from the UI thread
func (t *roomViewToolbar) displayRoomSubject(subject string) {
	t.roomSubjectLabel.SetText(subject)
}

// onToggleRoomSubject MUST be called from the UI thread
func (t *roomViewToolbar) onToggleRoomSubject() {
	if t.roomSubjectRevealer.GetRevealChild() {
		t.onHideRoomSubject()
	} else {
		t.onShowRoomSubject()
	}
}

// onEditRoomSubject MUST be called from the UI thread
func (t *roomViewToolbar) onEditRoomSubject() {
	t.isEditingSubject = true
	t.toggleEditSubjectComponents(false)

	setTextViewText(t.roomSubjectTextView, t.roomSubjectLabel.GetLabel())
}

// onCancelEditSubject MUST be called from the UI thread
func (t *roomViewToolbar) onCancelEditSubject() {
	if t.isEditingSubject {
		t.toggleEditSubjectComponents(true)
		return
	}

	t.resetSubjectComponents()
}

// resetSubjectComponents MUST be called from the UI thread
func (t *roomViewToolbar) resetSubjectComponents() {
	setTextViewText(t.roomSubjectTextView, "")
	t.onToggleRoomSubject()
}

// onApplyEditSubject MUST be called from the UI thread
func (t *roomViewToolbar) onApplyEditSubject() {
	newSubject := getTextViewText(t.roomSubjectTextView)
	t.roomView.updateSubjectRoom(newSubject,
		func() {
			t.roomSubjectLabel.SetText(newSubject)
			t.toggleEditSubjectComponents(true)
			setTextViewText(t.roomSubjectTextView, "")
		})
}

// onRoomSubectChanged MUST be called from the UI thread
func (t *roomViewToolbar) onRoomSubectChanged() {
	t.roomSubjectApplyButton.SetSensitive(t.roomView.room.GetSubject() != getTextViewText(t.roomSubjectTextView))
}

// onShowRoomSubject MUST be called from the UI thread
func (t *roomViewToolbar) onShowRoomSubject() {
	t.roomSubjectRevealer.SetRevealChild(true)
	t.roomSubjectButton.SetTooltipText(i18n.Local("Hide room subject"))
	t.roomSubjectButtonImage.SetFromIconName(roomSubjectShownIconName, gtki.ICON_SIZE_BUTTON)
}

// onHideRoomSubject MUST be called from the UI thread
func (t *roomViewToolbar) onHideRoomSubject() {
	t.roomSubjectRevealer.SetRevealChild(false)
	t.roomSubjectButton.SetTooltipText(i18n.Local("Show room subject"))
	t.roomSubjectButtonImage.SetFromIconName(roomSubjectHiddenIconName, gtki.ICON_SIZE_BUTTON)
}

func (t *roomViewToolbar) toggleEditSubjectComponents(v bool) {
	t.roomSubjectLabel.SetVisible(v)
	t.roomSubjectScrolledWindow.SetVisible(!v)
	t.roomSubjectEditButton.SetVisible(v)
	t.roomSubjectButtonsContainer.SetVisible(!v)
}

// handleEditSubjectComponents MUST be called from the UI thread
func (t *roomViewToolbar) handleEditSubjectComponents() {
	rules, ok := editSubjectComponentRules[editSubjectContext{
		t.roomView.room.SubjectCanBeChanged(),
		t.roomView.room.HasSubject(),
	}]

	if ok {
		t.roomSubjectButton.SetVisible(rules[showSubjectButton])
		t.roomSubjectEditButton.SetVisible(rules[showEditSubjectButton])
		t.SetVisibleEditSubjectComponent(rules[showEditSubjectComponent])
		t.roomSubjectLabel.SetVisible(rules[showSubjectLabel])
	}
}

// SetVisibleEditSubjectComponent MUST be called from the UI thread
func (t *roomViewToolbar) SetVisibleEditSubjectComponent(v bool) {
	t.roomSubjectScrolledWindow.SetVisible(v)
	t.roomSubjectButtonsContainer.SetVisible(v)
}

const (
	roomOfflineIconName = "room"
	roomOnlineIconName  = "room-offline"
)

// roomDisableEvent MUST be called from the UI thread
func (t *roomViewToolbar) roomDisableEvent() {
	t.handleEditSubjectComponents()

	addClassStyle(roomToolbarDisableClassName, t.roomNameLabel)
	t.roomStatusIcon.SetFromPixbuf(getMUCIconPixbuf(roomOnlineIconName))
	t.roomMenuButton.SetSensitive(false)
}

// roomEnableEvent MUST be called from the UI thread
func (t *roomViewToolbar) roomEnableEvent() {
	t.handleEditSubjectComponents()

	removeClassStyle(roomToolbarDisableClassName, t.roomNameLabel)
	t.roomStatusIcon.SetFromPixbuf(getMUCIconPixbuf(roomOfflineIconName))
	t.roomMenuButton.SetSensitive(true)
}

// selfOccupantDisconnectedEvent MUST be called from the UI thread
func (t *roomViewToolbar) selfOccupantDisconnectedEvent() {
	t.handleEditSubjectComponents()
}
