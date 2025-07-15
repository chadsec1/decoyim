package gui

import (
	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/i18n"

	"github.com/chadsec1/decoyim/session/muc"
	"github.com/chadsec1/decoyim/session/muc/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gtki"
)

type roomViewDataProvider interface {
	passwordProvider() string   // passwordProvider WILL to be called from any thread
	backToPreviousStep() func() // backToPreviousStep WILL be called from the UI thread
	notifyError(string)         // notifyError WILL be called from the UI thread
	doWhenNoErrorOccurred()     // doWhenNoErrorOccurred WILL be called from the UI thread
}

type roomViewData struct {
	passsword            string
	onBackToPreviousStep func()       // onBackToPreviousStep WILL be called from the UI thread
	onNotifyError        func(string) // onNotifyError WILL be called from the UI thread
	onNoErrors           func()       // onNoErrors WILL be called from the UI thread
}

// passwordProvider implements the `roomViewDataProvider` interface
func (rvd *roomViewData) passwordProvider() string {
	return rvd.passsword
}

// backToPreviousStep implements the `roomViewDataProvider` interface
func (rvd *roomViewData) backToPreviousStep() func() {
	return rvd.onBackToPreviousStep
}

// notifyError implements the `roomViewDataProvider` interface
func (rvd *roomViewData) notifyError(err string) {
	if rvd.onNotifyError != nil {
		rvd.onNotifyError(err)
	}
}

// doWhenNoErrorOccurred implements the `roomViewDataProvider` interface
func (rvd *roomViewData) doWhenNoErrorOccurred() {
	if rvd.onNoErrors != nil {
		rvd.onNoErrors()
	}
}

type roomView struct {
	u       *gtkUI
	account *account
	room    *muc.Room
	window  *roomViewWindow

	cancel chan bool

	opened             bool
	passwordProvider   func() string
	backToPreviousStep func()
	onJoinFinished     *callbacksSet // onJoinFinished WILL be called from the UI thread

	notifications           *roomNotifications
	connectingNotifications []*notificationBar

	warnings           *roomViewWarnings
	warningsInfoBar    *roomViewWarningsInfoBar
	loadingViewOverlay *roomViewLoadingOverlay
	isReconnecting     bool
	enteredAtLeastOnce bool

	subscribers *roomViewSubscribers

	main    *roomViewMain
	toolbar *roomViewToolbar
	roster  *roomViewRoster
	conv    *roomViewConversation
	lobby   *roomViewLobby

	log coylog.Logger
}

func (u *gtkUI) newRoomView(a *account, room *muc.Room) *roomView {
	view := &roomView{
		u:       u,
		account: a,
		room:    room,
		log:     a.log.WithField("room", room.ID),
	}

	view.onJoinFinished = newCallbacksSet(func() {
		view.enteredAtLeastOnce = true
	})

	view.initRoomWindow()
	view.initSubscribers()
	view.initNotifications()

	view.toolbar = view.newRoomViewToolbar()
	view.roster = view.newRoomViewRoster()
	view.conv = view.newRoomViewConversation()

	view.warnings = view.newRoomViewWarnings()
	view.warningsInfoBar = view.newRoomViewWarningsInfoBar()
	view.loadingViewOverlay = view.newRoomViewLoadingOverlay()

	view.initRoomViewComponents()

	view.requestRoomDiscoInfo()

	return view
}

func (v *roomView) initSubscribers() {
	v.subscribers = v.newRoomViewSubscribers()
	v.room.Subscribe(v.handleRoomEvent)

	v.subscribe("room", func(ev roomViewEvent) {
		doInUIThread(func() {
			v.onEventReceived(ev)
		})
	})
}

func (v *roomView) initRoomWindow() {
	v.window = v.newRoomViewWindow()
}

func (v *roomView) initRoomViewComponents() {
	v.lobby = v.newRoomViewLobby()
	v.main = v.newRoomMainView()

	v.window.content.PackStart(v.lobby.content, true, true, 0)
	v.window.content.PackStart(v.main.content, true, true, 0)

	v.window.notificationsArea.Add(v.notifications.notificationsBox())
	v.window.privacyWarningBox.Add(v.warningsInfoBar.view())
	v.window.overlay.AddOverlay(v.loadingViewOverlay.view())

}

// onEventReceived MUST be called from the UI thread
func (v *roomView) onEventReceived(ev roomViewEvent) {
	switch t := ev.(type) {
	case selfOccupantRemovedEvent:
		v.selfOccupantRemovedEvent()
	case roomDiscoInfoReceivedEvent:
		v.roomDiscoInfoReceivedEvent(t.info)
	case roomConfigRequestTimeoutEvent:
		v.roomConfigRequestTimeoutEvent()
	case joinRoomFinishedEvent:
		v.joinRoomFinishedEvent()
	case selfOccupantAffiliationUpdatedEvent:
		v.selfOccupantAffiliationUpdatedEvent(t.selfAffiliationUpdate)
	case selfOccupantAffiliationRoleUpdatedEvent:
		v.selfOccupantAffiliationRoleUpdatedEvent(t.selfAffiliationRoleUpdate)
	case selfOccupantRoleUpdatedEvent:
		v.selfOccupantRoleUpdatedEvent(t.selfRoleUpdate)
	case selfOccupantConnectedEvent:
		v.selfOccupantConnectedEvent()
	case selfOccupantDisconnectedEvent:
		v.selfOccupantDisconnectedEvent()
	case selfOccupantConnectingEvent:
		v.selfOccupantConnectingEvent()
	case nicknameConflictEvent:
		v.nicknameConflictEvent(t.nickname)
	case registrationRequiredEvent:
		v.registrationRequiredEvent()
	case notAuthorizedEvent:
		v.notAuthorizedEvent()
	case serviceUnavailableEvent:
		v.serviceUnavailableEvent()
	case unknownErrorEvent:
		v.unknownErrorEvent()
	case occupantForbiddenEvent:
		v.occupantForbiddenEvent()
	case roomDisableEvent:
		v.roomDisableEvent()
	case selfOccupantRemovedOnAffiliationChangeEvent:
		v.selfOccupantRemovedOnAffiliationChangeEvent()
	}
}

// requestRoomDiscoInfo MUST be called from the UI thread
func (v *roomView) requestRoomDiscoInfo() {
	v.loadingViewOverlay.onRoomDiscoInfoLoad()
	v.notifications.clearErrors()

	go v.account.session.RefreshRoomProperties(v.roomID())
}

// roomDiscoInfoReceivedEvent MUST be called from the UI thread
func (v *roomView) roomDiscoInfoReceivedEvent(di data.RoomDiscoInfo) {
	v.loadingViewOverlay.hide()

	v.warnings.clear()
	v.addRoomWarningsBasedOnInfo(di)

	v.warningsInfoBar.reveal()
}

// roomConfigRequestTimeoutEvent MUST be called from the UI thread
func (v *roomView) roomConfigRequestTimeoutEvent() {
	v.loadingViewOverlay.hide()
	v.warnings.clear()

	v.notifications.error(roomNotificationOptions{
		message: i18n.Local("Loading the room information took longer than usual, " +
			"perhaps the connection to the server was lost. Do you want to try again?"),
		actions: roomNotificationActions{{
			label:        i18n.Local("Yes, try again"),
			responseType: gtki.RESPONSE_YES,
			signals: map[string]interface{}{
				"clicked": v.requestRoomDiscoInfo,
			},
		}},
	})
}

// selfOccupantAffiliationUpdatedEvent MUST be called from the UI thread
func (v *roomView) selfOccupantAffiliationUpdatedEvent(selfAffiliationUpdate data.SelfAffiliationUpdate) {
	notificationInfo := roomNotificationOptions{
		message:   getSelfAffiliationUpdateMessage(selfAffiliationUpdate),
		showTime:  true,
		closeable: true,
	}

	if selfAffiliationUpdate.New.IsBanned() {
		v.notifications.error(notificationInfo)
		v.disableRoomView()
	} else {
		v.notifications.other(notificationInfo)
	}
}

// selfOccupantAffiliationRoleUpdatedEvent MUST be called from the UI thread
func (v *roomView) selfOccupantAffiliationRoleUpdatedEvent(selfAffiliationRoleUpdate data.SelfAffiliationRoleUpdate) {
	notificationInfo := roomNotificationOptions{
		message:   getSelfAffiliationRoleUpdateMessage(selfAffiliationRoleUpdate),
		showTime:  true,
		closeable: true,
	}

	if selfAffiliationRoleUpdate.NewAffiliation.IsBanned() || selfAffiliationRoleUpdate.NewRole.IsNone() {
		v.notifications.error(notificationInfo)
		v.disableRoomView()
	} else {
		v.notifications.other(notificationInfo)
	}
}

// selfOccupantRoleUpdatedEvent MUST be called from the UI thread
func (v *roomView) selfOccupantRoleUpdatedEvent(selfRoleUpdate data.SelfRoleUpdate) {
	notificationInfo := roomNotificationOptions{
		message:   getSelfRoleUpdateMessage(selfRoleUpdate),
		showTime:  true,
		closeable: true,
	}

	if selfRoleUpdate.New.IsNone() {
		v.notifications.error(notificationInfo)
		v.disableRoomView()
	} else {
		v.notifications.other(notificationInfo)
	}
}

// selfOccupantRemovedEvent MUST be called from the UI thread
func (v *roomView) selfOccupantRemovedEvent() {
	v.notifications.error(roomNotificationOptions{
		message:   i18n.Local("You have been removed from this room because it's now a members-only room."),
		showTime:  true,
		closeable: true,
	})

	v.disableRoomView()
}

// selfOccupantRemovedOnAffiliationChangeEvent MUST be called from the UI thread
func (v *roomView) selfOccupantRemovedOnAffiliationChangeEvent() {
	v.notifications.error(roomNotificationOptions{
		message:   i18n.Local("You have been removed from this members-only room since you are not $affiliation{a member} anymore."),
		showTime:  true,
		closeable: true,
	})

	v.disableRoomView()
}

// enableRoomView MUST be called from the UI thread
func (v *roomView) enableRoomView() {
	v.publishEvent(roomEnableEvent{})
}

// disableRoomView MUST be called from the UI thread
func (v *roomView) disableRoomView() {
	v.publishEvent(roomDisableEvent{})
}

// roomDisableEvent MUST be called from the UI thread
func (v *roomView) roomDisableEvent() {
	v.warningsInfoBar.hide()
}

// onDestroyWindow MUST be called from the UI thread
func (v *roomView) onDestroyWindow() {
	v.opened = false
	v.account.removeRoomView(v.roomID())
	go v.cancelActiveRequests()
}

// confirmWindowClose MUST be called from the UI thread
func (v *roomView) confirmWindowClose() {
	if v.isSelfOccupantInTheRoom() {
		v.showCloseConfirmWindow()
	}
}

// cancelActiveRequests MUST NOT be called from the UI thread
func (v *roomView) cancelActiveRequests() {
	if v.cancel != nil {
		v.cancel <- true
		v.cancel = nil
	}
}

func (v *roomView) isOpen() bool {
	return v.opened
}

func (v *roomView) isSelfOccupantInTheRoom() bool {
	return v.room.IsSelfOccupantInTheRoom()
}

func (v *roomView) show() {
	v.window.show()
	v.opened = true
}

// close MUST be called from the UI thread
func (v *roomView) close() {
	v.window.destroy()
}

// onLeaveRoom MUST be called from the UI thread
func (v *roomView) onLeaveRoom() {
	v.tryLeaveRoom()
}

// tryLeaveRoom MUST be called from the UI thread
func (v *roomView) tryLeaveRoom() {
	onSuccess := func() {
		doInUIThread(v.window.destroy)
	}

	onError := func(err error) {
		v.log.WithError(err).Error("An error occurred when trying to leave the room")
	}

	go v.account.leaveRoom(
		v.roomID(),
		v.room.SelfOccupantNickname(),
		onSuccess,
		onError,
		nil,
	)
}

// publishDestroyEvent MUST NOT be called from the UI thread
func (v *roomView) publishDestroyEvent(reason string, alternativeRoomID jid.Bare, password string) {
	v.publishEvent(roomDestroyedEvent{
		reason:      reason,
		alternative: alternativeRoomID,
		password:    password,
	})
}

// tryDestroyRoom MUST be called from the UI thread, but please, note that
func (v *roomView) tryDestroyRoom(reason string, alternativeRoomID jid.Bare, password string) {
	v.loadingViewOverlay.onRoomDestroy()

	sc, ec := v.account.session.DestroyRoom(v.roomID(), reason, alternativeRoomID, password)
	go func() {
		select {
		case <-sc:
			v.log.Info("The room has been destroyed")
			v.publishDestroyEvent(reason, alternativeRoomID, password)
			doInUIThread(func() {
				v.loadingViewOverlay.hide()

				v.notifications.info(roomNotificationOptions{
					message:   i18n.Local("The room has been destroyed"),
					closeable: true,
				})
			})
		case err := <-ec:
			v.log.WithError(err).Error("An error occurred when trying to destroy the room")
			doInUIThread(func() {
				v.loadingViewOverlay.hide()

				dr := createDestroyDialogError(func() {
					v.tryDestroyRoom(reason, alternativeRoomID, password)
				})

				dr.updateErrorMessage(err)
				dr.show()
			})
		}
	}()
}

// tryUpdateOccupantAffiliation MUST NOT be called from the UI thread
func (v *roomView) tryUpdateOccupantAffiliation(o *muc.Occupant, newAffiliation data.Affiliation, reason string) {
	doInUIThread(func() {
		v.loadingViewOverlay.onOccupantAffiliationUpdate()
	})

	previousAffiliation := o.Affiliation
	sc, ec := v.account.session.UpdateOccupantAffiliation(v.roomID(), o.Nickname, o.RealJid, newAffiliation, reason)

	select {
	case <-sc:
		v.log.Info("The affiliation has been changed")
		doInUIThread(func() {
			v.onOccupantAffiliationUpdateSuccess(o, previousAffiliation, newAffiliation)
		})
	case err := <-ec:
		v.log.WithError(err).Error("An error occurred while updating the occupant affiliation")
		doInUIThread(func() {
			v.onOccupantAffiliationUpdateError(o.Nickname, newAffiliation, err)
		})
	}
}

// onOccupantAffiliationUpdateSuccess MUST be called from the UI thread
func (v *roomView) onOccupantAffiliationUpdateSuccess(o *muc.Occupant, previousAffiliation, affiliation data.Affiliation) {
	v.loadingViewOverlay.hide()

	v.notifications.info(roomNotificationOptions{
		message:   getAffiliationUpdateSuccessMessage(o.Nickname, previousAffiliation, affiliation),
		closeable: true,
	})
}

// onOccupantAffiliationUpdateError MUST be called from the UI thread
func (v *roomView) onOccupantAffiliationUpdateError(nickname string, newAffiliation data.Affiliation, err error) {
	messages := getAffiliationUpdateFailureMessage(nickname, newAffiliation, err)

	v.loadingViewOverlay.hide()

	v.notifications.error(roomNotificationOptions{
		message:   messages.notificationMessage,
		closeable: true,
	})

	dr := createDialogErrorComponent(dialogErrorOptions{
		title:        messages.errorDialogTitle,
		header:       messages.errorDialogHeader,
		message:      messages.errorDialogMessage,
		parentWindow: v.mainWindow(),
	})

	dr.show()
}

// tryUpdateOccupantRole MUST NOT be called from the UI thread
func (v *roomView) tryUpdateOccupantRole(o *muc.Occupant, newRole data.Role, reason string) {
	l := v.log.WithField("occupant", o.Nickname)

	doInUIThread(func() {
		v.loadingViewOverlay.onOccupantRoleUpdate(newRole)
	})

	previousRole := o.Role
	sc, ec := v.account.session.UpdateOccupantRole(v.roomID(), o.Nickname, newRole, reason)

	select {
	case <-sc:
		l.Info("The role has been changed")
		doInUIThread(func() {
			v.onOccupantRoleUpdateSuccess(o.Nickname, previousRole, newRole)
		})
	case err := <-ec:
		l.WithError(err).Error("An error occurred while updating the occupant role")
		doInUIThread(func() {
			v.onOccupantRoleUpdateError(o.Nickname, newRole)
		})
	}
}

// onOccupantRoleUpdateSuccess MUST be called from the UI thread
func (v *roomView) onOccupantRoleUpdateSuccess(nickname string, previousRole, newRole data.Role) {
	v.loadingViewOverlay.hide()

	v.notifications.info(roomNotificationOptions{
		message:   getRoleUpdateSuccessMessage(nickname, previousRole, newRole),
		closeable: true,
	})
}

// onOccupantRoleUpdateError MUST be called from the UI thread
func (v *roomView) onOccupantRoleUpdateError(nickname string, newRole data.Role) {
	messages := getRoleUpdateFailureMessage(nickname, newRole)

	v.loadingViewOverlay.hide()

	v.notifications.error(roomNotificationOptions{
		message:   messages.notificationMessage,
		closeable: true,
	})

	dr := createDialogErrorComponent(dialogErrorOptions{
		title:        messages.errorDialogTitle,
		header:       messages.errorDialogHeader,
		message:      messages.errorDialogMessage,
		parentWindow: v.window.view(),
	})

	dr.show()
}

// updateSubjectRoom MUST be called from the UI thread
func (v *roomView) updateSubjectRoom(s string, onSuccess func()) {
	err := v.account.session.UpdateRoomSubject(v.roomID(), v.room.SelfOccupant().RealJid.String(), s)
	if err != nil {
		v.notifications.error(roomNotificationOptions{
			message:   i18n.Local("The room subject couldn't be updated."),
			closeable: true,
		})
		return
	}

	v.notifications.info(roomNotificationOptions{
		message:   i18n.Local("The room subject has been updated."),
		closeable: true,
	})

	if onSuccess != nil {
		onSuccess()
	}
}

// switchToLobbyView MUST be called from the UI thread
func (v *roomView) switchToLobbyView() {
	l := i18n.Local("Cancel")
	if v.backToPreviousStep != nil {
		l = i18n.Local("Return")
	}
	setFieldLabel(v.lobby.cancelButton, l)

	v.warningsInfoBar.whenRequestedToClose(nil)

	v.hideMainView()
	v.showLobbyView()
}

// switchToMainView MUST be called from the UI thread
func (v *roomView) switchToMainView() {
	v.warningsInfoBar.whenRequestedToClose(v.warningsInfoBar.hide)

	v.hideLobbyView()
	v.showMainView()
}

// showLobbyView MUST be called from the UI thread
func (v *roomView) showLobbyView() {
	v.lobby.content.Show()
}

// hideLobbyView MUST be called from the UI thread
func (v *roomView) hideLobbyView() {
	v.lobby.content.Hide()
}

// showMainView MUST be called from the UI thread
func (v *roomView) showMainView() {
	v.main.content.Show()
}

// hideMainView MUST be called from the UI thread
func (v *roomView) hideMainView() {
	v.main.content.Hide()
}

// sendJoinRoomRequest MUST NOT be called from the UI thread
func (v *roomView) sendJoinRoomRequest(nickname, password string, doAfterRequestSent func()) {
	v.room.UpdatePassword(password)

	err := v.account.session.JoinRoom(v.roomID(), nickname, password)
	if err != nil {
		v.finishJoinRequestWithError(err)
	}

	if doAfterRequestSent != nil {
		doAfterRequestSent()
	}
}

// finishJoinRequestWithError MUST NOT be called from the UI thread
func (v *roomView) finishJoinRequestWithError(err error) {
	v.log.WithError(err).Error("An error occurred while trying to join the room")
	doInUIThread(func() {
		v.loadingViewOverlay.hide()

		v.switchToLobbyView()
		v.lobby.onJoinFailed(err)
	})
}

// joinRoomFinishedEvent MUST be called from the UI thread
func (v *roomView) joinRoomFinishedEvent() {
	v.onJoinFinished.invokeAll()

	v.loadingViewOverlay.hide()

	// TODO: This will change to something more proper in this case.
	// For now, we assume that we are in the lobby when joining the room.
	v.switchToMainView()
}

// onJoinCancel MUST be called from the UI thread
func (v *roomView) onJoinCancel() {
	v.window.destroy()

	if v.backToPreviousStep != nil {
		v.backToPreviousStep()
	}
}

// messageForbidden MUST NOT be called from the UI thread
func (v *roomView) messageForbidden() {
	v.publishEvent(messageForbidden{})
}

// messageNotAccepted MUST NOT be called from the UI thread
func (v *roomView) messageNotAccepted() {
	v.publishEvent(messageNotAcceptable{})
}

// nicknameConflict MUST NOT be called from the UI thread
func (v *roomView) nicknameConflict(nickname string) {
	v.publishEvent(nicknameConflictEvent{nickname})
}

// serviceUnavailableEvent MUST NOT be called from the UI thread
func (v *roomView) serviceUnavailable() {
	v.publishEvent(serviceUnavailableEvent{})
}

// unknownError MUST NOT be called from the UI thread
func (v *roomView) unknownError() {
	v.publishEvent(unknownErrorEvent{})
}

// registrationRequired MUST NOT be called from the UI thread
func (v *roomView) registrationRequired(nickname string) {
	v.publishEvent(registrationRequiredEvent{nickname})
}

// notAuthorized MUST NOT be called from the UI thread
func (v *roomView) notAuthorized() {
	v.publishEvent(notAuthorizedEvent{})
}

// occupantForbidden MUST NOT be called from the UI thread
func (v *roomView) occupantForbidden() {
	v.publishEvent(occupantForbiddenEvent{})
}

// publishOccupantAffiliationUpdatedEvent MUST NOT be called from the UI thread
func (v *roomView) publishOccupantAffiliationRoleUpdatedEvent(affiliationRoleUpdate data.AffiliationRoleUpdate) {
	v.publishEvent(occupantAffiliationRoleUpdatedEvent{affiliationRoleUpdate})
}

// publishSelfOccupantAffiliationUpdatedEvent MUST NOT be called from the UI thread
func (v *roomView) publishSelfOccupantAffiliationRoleUpdatedEvent(selfAffiliationRoleUpdate data.SelfAffiliationRoleUpdate) {
	v.publishEvent(selfOccupantAffiliationRoleUpdatedEvent{selfAffiliationRoleUpdate})
}

// publishOccupantAffiliationUpdatedEvent MUST NOT be called from the UI thread
func (v *roomView) publishOccupantAffiliationUpdatedEvent(affiliationUpdate data.AffiliationUpdate) {
	v.publishEvent(occupantAffiliationUpdatedEvent{affiliationUpdate})
}

// publishSelfOccupantAffiliationUpdatedEvent MUST NOT be called from the UI thread
func (v *roomView) publishSelfOccupantAffiliationUpdatedEvent(selfAffiliationUpdate data.SelfAffiliationUpdate) {
	v.publishEvent(selfOccupantAffiliationUpdatedEvent{selfAffiliationUpdate})
}

// publishOccupantRoleUpdatedEvent MUST NOT be called from the UI thread
func (v *roomView) publishOccupantRoleUpdatedEvent(roleUpdate data.RoleUpdate) {
	v.publishEvent(occupantRoleUpdatedEvent{roleUpdate})
}

// publishSelfOccupantRoleUpdatedEvent MUST NOT be called from the UI thread
func (v *roomView) publishSelfOccupantRoleUpdatedEvent(selfRoleUpdate data.SelfRoleUpdate) {
	v.publishEvent(selfOccupantRoleUpdatedEvent{selfRoleUpdate})
}

// handleDiscoInfoReceived MUST NOT be called from the UI thread
func (v *roomView) handleDiscoInfoReceived(di data.RoomDiscoInfo) {
	v.publishEvent(roomDiscoInfoReceivedEvent{di})

	reconnecting := v.isReconnecting
	if reconnecting {
		doInUIThread(func() {
			v.onReconnectingRoomInfoReceived(di)
		})
	}
}

// handleDiscoInfoTimeout MUST NOT be called from the UI thread
func (v *roomView) handleDiscoInfoTimeout() {
	v.publishEvent(roomConfigRequestTimeoutEvent{})

	reconnecting := v.isReconnecting
	if reconnecting {
		doInUIThread(v.onReconnectingRoomInfoTimeout)
		v.isReconnecting = false
	}
}

// onReconnectingRoomInfoReceived MUST be called from the UI thread
func (v *roomView) onReconnectingRoomInfoReceived(di data.RoomDiscoInfo) {
	v.lobby.clearNicknameConflictList()

	v.notifications.removeAll(v.connectingNotifications...)
	v.notifications.info(
		roomNotificationOptions{
			message:   i18n.Local("Your connection has been restored; you can join this room again."),
			showTime:  true,
			closeable: true,
		})

	password := ""
	if di.PasswordProtected {
		password = v.room.Password()
	}
	go v.sendJoinRoomRequest(v.room.SelfOccupantNickname(), password, func() { v.enableRoomView() })
}

// onReconnectingRoomInfoTimeout MUST be called from the UI thread
func (v *roomView) onReconnectingRoomInfoTimeout() {
	v.notifications.removeAll(v.connectingNotifications...)
	v.notifications.error(roomNotificationOptions{
		message: i18n.Local("Your connection was recovered but " +
			"loading the room information took longer than usual, " +
			"perhaps the connection to the server was lost. Do you want to try again?"),
		showTime:  true,
		closeable: true,
		actions: roomNotificationActions{
			{
				label:        i18n.Local("Yes, try again"),
				responseType: gtki.RESPONSE_YES,
				signals: map[string]interface{}{
					"clicked": func() {
						v.notifications.clearAll()
						v.notifications.other(roomNotificationOptions{
							message:     i18n.Local("Trying to connect to the room..."),
							showTime:    true,
							showSpinner: true,
						})
						go v.requestRoomInfoOnReconnect()
					},
				},
			},
		},
	})
}

// requestRoomInfoOnReconnect MUST NOT be called from the UI thread
func (v *roomView) requestRoomInfoOnReconnect() {
	v.isReconnecting = true
	v.account.session.RefreshRoomProperties(v.roomID())

	previousOnJoinFinished := v.onJoinFinished
	v.onJoinFinished = newCallbacksSet(func() {
		doInUIThread(func() {
			v.roomReconnectFinished(previousOnJoinFinished)
		})
	})
}

// roomReconnectFinished MUST be called from the UI thread
func (v *roomView) roomReconnectFinished(previousOnJoinFinished *callbacksSet) {
	v.isReconnecting = false

	v.onJoinFinished = previousOnJoinFinished
	v.onJoinFinished.invokeAll()
}

// selfOccupantConnectedEvent MUST NOT be called from the UI thread
func (v *roomView) selfOccupantConnectedEvent() {
	go v.requestRoomInfoOnReconnect()
}

// selfOccupantDisconnectedEvent MUST be called from the UI thread
func (v *roomView) selfOccupantDisconnectedEvent() {
	v.notifications.removeAll(v.connectingNotifications...)
	v.notifications.error(roomNotificationOptions{
		message:           i18n.Local("The connection to the server has been lost, please verify your connection."),
		showTime:          true,
		closeable:         true,
		onlyCloseManually: true,
	})

	v.disableRoomView()
}

// selfOccupantConnectingEvent MUST be called from the UI thread
func (v *roomView) selfOccupantConnectingEvent() {
	n := v.notifications.newNotification(gtki.MESSAGE_OTHER, roomNotificationOptions{
		message:     i18n.Local("Connecting to the room..."),
		showTime:    true,
		showSpinner: true,
	})
	v.connectingNotifications = append(v.connectingNotifications, n)
}

// mainWindow MUST be called from the UI thread
func (v *roomView) mainWindow() gtki.Window {
	return v.window.view()
}

// present MUST be called from the UI thread
func (v *roomView) present() {
	v.window.present()
}

func (v *roomView) roomID() jid.Bare {
	return v.room.ID
}
