package gui

import (
	"github.com/chadsec1/decoyim/decoylog"
	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gtki"
)

type mucJoinRoomView struct {
	u                 *gtkUI
	builder           *builder
	roomFormComponent *mucRoomFormComponent

	dialog           gtki.Dialog `gtk-widget:"join-room-dialog"`
	joinButton       gtki.Button `gtk-widget:"join-room-button"`
	spinnerBox       gtki.Box    `gtk-widget:"spinner-box"`
	notificationArea gtki.Box    `gtk-widget:"notification-area-box"`

	spinner       *spinner
	notifications *notificationsComponent
}

func newMUCJoinRoomView(u *gtkUI) *mucJoinRoomView {
	view := &mucJoinRoomView{
		u: u,
	}

	view.loadUIDefinition()
	view.initNotificationsAndSpinner()
	view.initRoomFormComponent()

	u.connectShortcutsChildWindow(view.dialog)

	return view
}

func (v *mucJoinRoomView) setBuilder(b *builder) {
	v.builder = b
}

func (v *mucJoinRoomView) connectUISignals(b *builder) {
	b.ConnectSignals(map[string]interface{}{
		"on_close_window":     v.onCloseWindow,
		"on_roomname_changed": v.enableJoinIfConditionsAreMet,
		"on_cancel":           v.dialog.Destroy,
		"on_join":             doOnlyOnceAtATime(v.tryJoinRoom),
	})
}

func (v *mucJoinRoomView) loadUIDefinition() {
	buildUserInterface("MUCJoinRoomDialog", v, v.setBuilder, v.connectUISignals)
}

func (v *mucJoinRoomView) initRoomFormComponent() {
	account := v.builder.get("accounts").(gtki.ComboBox)
	roomEntry := v.builder.get("room-name-entry").(gtki.Entry)
	chatServicesList := v.builder.get("chat-services-list").(gtki.ComboBoxText)
	chatServicesEntry := v.builder.get("chat-services-entry").(gtki.Entry)

	v.roomFormComponent = v.u.createMUCRoomFormComponent(&mucRoomFormData{
		errorNotifications:     v.notifications,
		connectedAccountsInput: account,
		roomNameEntry:          roomEntry,
		chatServicesInput:      chatServicesList,
		chatServicesEntry:      chatServicesEntry,
		onAccountSelected:      v.updateServicesBasedOnAccount,
		onNoAccount:            v.onNoAccountsConnected,
		onChatServiceChanged:   v.enableJoinIfConditionsAreMet,
	})
}

func (v *mucJoinRoomView) updateServicesBasedOnAccount(ca *account) {
	doInUIThread(func() {
		v.notifications.clearErrors()
		v.enableJoinIfConditionsAreMet()
	})
}

func (v *mucJoinRoomView) onNoAccountsConnected() {
	doInUIThread(v.enableJoinIfConditionsAreMet)
}

func (v *mucJoinRoomView) initNotificationsAndSpinner() {
	v.notifications = v.u.newNotificationsComponent()
	v.spinner = v.u.newSpinnerComponent()

	v.notificationArea.Add(v.notifications.contentBox())
	v.spinnerBox.Add(v.spinner.spinner())
}

func (v *mucJoinRoomView) onCloseWindow() {
	v.roomFormComponent.onDestroy()
}

func (v *mucJoinRoomView) typedRoomName() string {
	return v.roomFormComponent.currentRoomNameValue()
}

// enableJoinIfConditionsAreMet MUST be called from the UI thread
func (v *mucJoinRoomView) enableJoinIfConditionsAreMet() {
	v.joinButton.SetSensitive(v.roomFormComponent.isFilled())
}

func (v *mucJoinRoomView) beforeJoiningRoom() {
	v.notifications.clearErrors()
	v.disableJoinFields()
	v.spinner.show()
}

func (v *mucJoinRoomView) onJoinSuccess(a *account, roomID jid.Bare, roomInfo *muc.RoomListing) {
	onNoErrors := func() {
		v.dialog.Hide()
		v.spinner.hide()
	}

	onNotifyError := func(err string) {
		v.spinner.hide()
		v.enableJoinFields()

		v.notifications.clearErrors()
		v.notifications.notifyOnError(err)
	}

	v.u.joinRoom(a, roomID, &roomViewData{
		onBackToPreviousStep: v.returnToJoinRoomView,
		onNotifyError:        onNotifyError,
		onNoErrors:           onNoErrors,
	})
}

func (v *mucJoinRoomView) returnToJoinRoomView() {
	v.enableJoinFields()
	v.dialog.Show()
}

func (v *mucJoinRoomView) onJoinFails(a *account, roomID jid.Bare) {
	a.log.WithField("room", roomID).Warn("The room is not provided by the service")

	doInUIThread(func() {
		v.notifications.error(i18n.Local("We couldn't find a room with that name."))
		v.enableJoinFields()
		v.spinner.hide()
	})
}

func (v *mucJoinRoomView) onJoinError(a *account, roomID jid.Bare, err error) {
	a.log.WithField("room", roomID).WithError(err).Warn("An error occurred trying to find the room")

	doInUIThread(func() {
		v.notifications.error(i18n.Local("It looks like the room you are trying to connect to doesn't exist, please verify the provided information."))
		v.enableJoinFields()
		v.spinner.hide()
	})
}

func (v *mucJoinRoomView) onServiceUnavailable(a *account, roomID jid.Bare) {
	a.log.WithField("room", roomID).Warn("An error occurred trying to find the room")

	doInUIThread(func() {
		v.notifications.error(i18n.Local("We can't get access to the service, please check your Internet connection or make sure the service exists."))
		v.enableJoinFields()
		v.spinner.hide()
	})
}

func (v *mucJoinRoomView) log() decoylog.Logger {
	l := v.u.hasLog.log

	ca := v.roomFormComponent.currentAccount()
	if ca != nil {
		l = ca.log
	}

	l.WithField("where", "mucJoinRoomView")

	return l
}

func (v *mucJoinRoomView) validateFieldsAndGetBareIfOk() (jid.Bare, bool) {
	local := v.roomFormComponent.currentRoomName()
	if !local.Valid() {
		v.notifications.error(i18n.Local("You must provide a valid room name."))
		return nil, false
	}

	chatServiceName := v.roomFormComponent.currentService()
	if !chatServiceName.Valid() {
		v.notifications.error(i18n.Local("You must provide a valid service name."))
		return nil, false
	}

	return jid.NewBare(local, chatServiceName), true
}

// tryJoinRoom MUST be called from the UI thread
func (v *mucJoinRoomView) tryJoinRoom(done func()) {
	if v.roomFormComponent.isValid() {
		c := v.newJoinRoomContext(v.roomFormComponent.currentAccount(), v.roomFormComponent.currentRoomID(), done)
		c.joinRoom()
		return
	}

	done()
}

func (v *mucJoinRoomView) isValidRoomName(name string) bool {
	return jid.ValidBareJID(name)
}

func (v *mucJoinRoomView) setSensitivityForJoin(f bool) {
	v.joinButton.SetSensitive(f)
}

func (v *mucJoinRoomView) disableJoinFields() {
	v.setSensitivityForJoin(false)
	v.roomFormComponent.disableFields()
}

func (v *mucJoinRoomView) enableJoinFields() {
	v.setSensitivityForJoin(true)
	v.roomFormComponent.enableFields()
}

func (u *gtkUI) mucShowJoinRoom() {
	view := newMUCJoinRoomView(u)

	u.connectShortcutsChildWindow(view.dialog)

	view.dialog.SetTransientFor(u.window)
	view.dialog.Show()
}
