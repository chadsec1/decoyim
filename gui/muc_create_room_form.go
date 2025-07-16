package gui

import (
	"github.com/chadsec1/decoyim/decoylog"
	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/session"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gtki"
)

// initCreateRoomForm MUST be called from the UI thread
func (v *mucCreateRoomView) initCreateRoomForm(d *mucCreateRoomData) {
	f := v.newCreateRoomForm()

	f.addCallbacks(v)

	if d != nil {
		f.roomFormComponent.setCurrentAccount(d.ca)
		f.roomFormComponent.setCurrentRoomName(d.roomName)
		f.roomFormComponent.setCurrentServiceValue(d.where)
		f.roomAutoJoinCheck.SetActive(d.autoJoin)
		f.roomConfigCheck.SetActive(d.customConfig)
		f.roomNameConflictList = d.roomNameConflictList
	}

	f.createRoom = func(ca *account, roomID jid.Bare) {
		v.createRoom(ca, roomID, func(err error) {
			f.onCreateRoomError(roomID, err)
		})
	}

	v.form = f
}

func (v *mucCreateRoomView) showCreateForm() {
	v.success.reset()
	v.container.Remove(v.success.view)
	v.container.Add(v.form.view)
	v.form.isShown = true
}

type mucCreateRoomViewForm struct {
	isShown           bool
	builder           *builder
	roomFormComponent *mucRoomFormComponent

	view              gtki.Box         `gtk-widget:"create-room-form"`
	roomAutoJoinCheck gtki.CheckButton `gtk-widget:"autojoin-check-button"`
	roomConfigCheck   gtki.CheckButton `gtk-widget:"config-room-check-button"`
	createButton      gtki.Button      `gtk-widget:"create-room-button"`
	spinnerBox        gtki.Box         `gtk-widget:"spinner-box"`
	notificationsArea gtki.Box         `gtk-widget:"notification-area-box"`

	spinner       *spinner
	notifications *notificationsComponent

	roomNameConflictList map[jid.Bare]error
	// createRoom MUST NOT be called from the UI thread
	createRoom               func(*account, jid.Bare)
	updateAutoJoinValue      func(bool)
	updateConfigureRoomValue func(bool)

	log func(*account, jid.Bare) decoylog.Logger
}

func (v *mucCreateRoomView) newCreateRoomForm() *mucCreateRoomViewForm {
	f := &mucCreateRoomViewForm{
		roomNameConflictList:     make(map[jid.Bare]error),
		updateAutoJoinValue:      v.updateAutoJoinValue,
		updateConfigureRoomValue: v.updateConfigureRoomValue,
		log:                      v.log,
	}

	f.initBuilder(v)
	f.initNotificationsAndSpinner(v)
	f.initRoomFormComponent(v)

	return f
}

func (f *mucCreateRoomViewForm) initBuilder(v *mucCreateRoomView) {
	f.builder = newBuilder("MUCCreateRoomForm")
	panicOnDevError(f.builder.bindObjects(f))

	f.builder.ConnectSignals(map[string]interface{}{
		"on_cancel":                   v.onCancel,
		"on_create":                   f.onCreateRoom,
		"on_room_name_change":         f.enableCreationIfConditionsAreMet,
		"on_room_autojoin_toggled":    f.onRoomAutoJoinToggled,
		"on_room_config_toggled":      f.onRoomConfigToggled,
		"on_chatservice_entry_change": f.enableCreationIfConditionsAreMet,
	})
}

func (f *mucCreateRoomViewForm) initRoomFormComponent(v *mucCreateRoomView) {
	account := f.builder.get("accounts").(gtki.ComboBox)
	roomEntry := f.builder.get("room-name-entry").(gtki.Entry)
	chatServicesList := f.builder.get("chat-services-list").(gtki.ComboBoxText)
	chatServicesEntry := f.builder.get("chat-services-entry").(gtki.Entry)

	f.roomFormComponent = v.u.createMUCRoomFormComponent(&mucRoomFormData{
		errorNotifications:     f.notifications,
		connectedAccountsInput: account,
		roomNameEntry:          roomEntry,
		chatServicesInput:      chatServicesList,
		chatServicesEntry:      chatServicesEntry,
		onAccountSelected:      f.updateServicesBasedOnAccount,
		onNoAccount:            f.onNoAccountsConnected,
		onChatServiceChanged:   f.enableCreationIfConditionsAreMet,
	})
}

func (f *mucCreateRoomViewForm) initNotificationsAndSpinner(v *mucCreateRoomView) {
	f.spinner = v.u.newSpinnerComponent()
	f.notifications = v.u.newNotificationsComponent()

	f.spinnerBox.Add(f.spinner.spinner())
	f.notificationsArea.Add(f.notifications.contentBox())
}

func (f *mucCreateRoomViewForm) onRoomAutoJoinToggled() {
	f.updateAutoJoinValue(f.roomAutoJoinCheck.GetActive())
}

func (f *mucCreateRoomViewForm) onRoomConfigToggled() {
	f.updateConfigureRoomValue(f.roomConfigCheck.GetActive())
}

func (f *mucCreateRoomViewForm) onCreateRoomError(roomID jid.Bare, err error) {
	doInUIThread(f.hideSpinnerAndEnableFields)

	switch err {
	case errCreateRoomAlreadyExists, session.ErrInformationQueryResponseWithGoneTag:
		f.roomNameConflictList[roomID] = err
		doInUIThread(f.disableCreateRoomButton)
	}

	doInUIThread(func() {
		f.notifyBasedOnError(err)
	})
}

// notifyBasedOnError MUST be called from the UI thread
func (f *mucCreateRoomViewForm) notifyBasedOnError(err error) {
	switch err {
	case errCreateRoomCheckIfExistsFails:
		f.notifications.error(i18n.Local("Couldn't connect to the service, please verify that it exists or try again later."))
	case errCreateRoomAlreadyExists:
		f.notifications.error(i18n.Local("That room already exists, try again with a different name."))
	case errCreateRoomTimeout:
		f.notifications.error(i18n.Local("We didn't receive a response from the server."))
	default:
		f.onCreateRoomFailed(err)
	}
}

// hideSpinnerAndEnableFields MUST be called from the UI thread
func (f *mucCreateRoomViewForm) hideSpinnerAndEnableFields() {
	f.spinner.hide()
	f.enableFields()
}

func (f *mucCreateRoomViewForm) onCreateRoomFailed(err error) {
	displayErr, ok := supportedCreateMUCErrors[err]
	if ok {
		f.notifications.error(displayErr)
	} else {
		f.notifications.error(i18n.Local("Could not create the room."))
	}
}

func (f *mucCreateRoomViewForm) addCallbacks(v *mucCreateRoomView) {
	v.onCreateOptionChange.add(func() {
		f.onCreateOptionsChange(v.autoJoin, v.configureRoom)
	})

	v.onDestroy.add(f.destroy)
}

func (f *mucCreateRoomViewForm) setCreateRoomButtonLabel(l string) {
	f.createButton.SetProperty("label", l)
}

func (f *mucCreateRoomViewForm) onCreateOptionsChange(autoJoin, configRoom bool) {
	f.setCreateRoomButtonLabel(labelForCreateRoomButton(autoJoin, configRoom))
}

func (f *mucCreateRoomViewForm) onCreateRoom() {
	if f.roomFormComponent.isValid() {
		f.beforeCreatingTheRoom()
		go f.createRoom(f.roomFormComponent.currentAccount(), f.roomFormComponent.currentRoomID())
	}
}

func (f *mucCreateRoomViewForm) beforeCreatingTheRoom() {
	f.spinner.show()
	f.disableFields()
	f.setFieldsSensitive(false)
}

func (f *mucCreateRoomViewForm) destroy() {
	f.isShown = false
	f.roomFormComponent.onDestroy()
}

func (f *mucCreateRoomViewForm) clearFields() {
	f.roomFormComponent.resetFields()
	f.enableCreationIfConditionsAreMet()
}

func (f *mucCreateRoomViewForm) reset() {
	f.spinner.hide()
	f.enableFields()
	f.clearFields()
}

func (f *mucCreateRoomViewForm) enableFields() {
	f.roomFormComponent.enableFields()
	f.setFieldsSensitive(true)
}

func (f *mucCreateRoomViewForm) disableFields() {
	f.roomFormComponent.disableFields()
	f.setFieldsSensitive(false)
}

func (f *mucCreateRoomViewForm) setFieldsSensitive(v bool) {
	f.createButton.SetSensitive(v)
	f.roomAutoJoinCheck.SetSensitive(v)
	f.roomConfigCheck.SetSensitive(v)
}

func (f *mucCreateRoomViewForm) updateServicesBasedOnAccount(ca *account) {
	doInUIThread(func() {
		f.notifications.clearErrors()
		f.enableCreationIfConditionsAreMet()
	})
}

func (f *mucCreateRoomViewForm) onNoAccountsConnected() {
	doInUIThread(f.enableCreationIfConditionsAreMet)
}

func (f *mucCreateRoomViewForm) enableCreationIfConditionsAreMet() {
	if f.roomFormComponent.hasNoErrorsReported() {
		f.notifications.clearErrors()
	}

	f.disableCreateRoomButton()

	if f.roomFormComponent.isEmpty() || f.checkIfRoomNameHasConflict() {
		return
	}

	f.enableCreateRoomButton()
}

func (f *mucCreateRoomViewForm) checkIfRoomNameHasConflict() bool {
	if err, ok := f.roomNameConflictList[f.roomFormComponent.currentRoomID()]; ok {
		f.notifyBasedOnError(err)
		return true
	}
	return false
}

func (f *mucCreateRoomViewForm) enableCreateRoomButton() {
	f.createButton.SetSensitive(true)
}

func (f *mucCreateRoomViewForm) disableCreateRoomButton() {
	f.createButton.SetSensitive(false)
}

func labelForCreateRoomButton(autoJoin, configRoom bool) string {
	if configRoom {
		return i18n.Local("Configure Room")
	}

	if autoJoin {
		return i18n.Local("Create Room & Join")
	}

	return i18n.Local("Create Room")
}
