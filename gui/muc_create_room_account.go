package gui

import (
	"time"

	"github.com/chadsec1/decoyim/session/muc"
	"github.com/chadsec1/decoyim/xmpp/jid"
)

// createInstantRoom IS SAFE to be called from the UI thread
func (a *account) createInstantRoom(roomID jid.Bare, onSuccess func(), onError func(error)) {
	rc, ec := a.session.CreateInstantRoom(roomID)
	go func() {
		select {
		case err := <-ec:
			onError(err)
		case <-rc:
			onSuccess()
		case <-time.After(timeoutThreshold):
			onError(errCreateRoomTimeout)
		}
	}()
}

func (v *mucCreateRoomView) createInstantRoom(ca *account, roomID jid.Bare, errHandler func(error)) {
	d := &mucCreateRoomData{
		autoJoin:      v.autoJoin,
		onNotifyError: v.notifications.notifyOnError,
		onNoErrors:    v.dialog.Destroy,
	}

	onSuccess := func() {
		v.onCreateRoomFinished(ca, roomID, d, func() {
			v.success.updateJoinRoomData(d)
			v.showSuccessView(ca, roomID)
			v.dialog.ShowAll()
		})
	}

	onError := func(err error) {
		v.log(ca, roomID).WithError(err).Error("Something went wrong when trying to create the instant room")
		errHandler(err)
	}

	ca.createInstantRoom(roomID, onSuccess, onError)
}

// checkIfRoomExists IS SAFE to be called from the UI thread
func (v *mucCreateRoomView) checkIfRoomExists(ca *account, roomID jid.Bare, result chan bool, errors chan error) {
	rc, ec := ca.session.HasRoom(roomID, nil)
	go func() {
		select {
		case err := <-ec:
			v.log(ca, roomID).WithError(err).Error("Error trying to validate if room exists")
			errors <- errCreateRoomCheckIfExistsFails
		case exists := <-rc:
			if exists {
				errors <- errCreateRoomAlreadyExists
				return
			}
			result <- true
		case <-v.cancel:
		}
	}()
}

func (v *mucCreateRoomView) createRoomDataBasedOnConfigForm(ca *account, roomID jid.Bare, cf *muc.RoomConfigForm) *mucCreateRoomData {
	return &mucCreateRoomData{
		ca:           ca,
		roomName:     roomID.Local(),
		where:        roomID.Host(),
		password:     cf.GetConfiguredPassword(),
		autoJoin:     v.autoJoin,
		customConfig: true,
	}
}

// onReserveRoomFinished MUST be called from the UI thread
func (v *mucCreateRoomView) onReserveRoomFinished(ca *account, roomID jid.Bare, cf *muc.RoomConfigForm) {
	v.destroy()

	createRoomData := v.createRoomDataBasedOnConfigForm(ca, roomID, cf)
	createRoomData.roomNameConflictList = v.form.roomNameConflictList

	onSuccess := func(autoJoin bool) {
		createRoomData.autoJoin = autoJoin
		createRoomData.password = cf.GetConfiguredPassword()
		v.onCreateRoomFinished(ca, roomID, createRoomData, func() {
			v.u.mucShowCreateRoomSuccess(ca, roomID, createRoomData)
		})
	}

	onCancel := func() {
		v.cancelConfiguration(ca, roomID, nil)
		v.u.mucShowCreateRoomForm(createRoomData)
	}

	v.u.launchRoomConfigView(roomConfigScenarioCreate, &roomConfigData{
		account:                ca,
		roomID:                 roomID,
		configForm:             cf,
		autoJoinRoomAfterSaved: v.autoJoin,
		doAfterConfigSaved:     onSuccess,
		doAfterConfigCanceled:  onCancel,
	})
}

// cancelConfiguration IS SAFE to be called from the UI thread
func (v *mucCreateRoomView) cancelConfiguration(ca *account, roomID jid.Bare, onError func(error)) {
	sc, ec := ca.session.DestroyRoom(roomID, "", nil, "")

	go func() {
		select {
		case <-sc:
			// do nothing
		case err := <-ec:
			v.log(ca, roomID).WithError(err).Error("An error occurred when trying to cancel the room configuration")
			if onError != nil {
				doInUIThread(func() {
					onError(err)
				})
			}
		}
	}()
}

// onCreateRoomFinished MUST NOT be called from the UI thread
func (v *mucCreateRoomView) onCreateRoomFinished(ca *account, roomID jid.Bare, createRoomData *mucCreateRoomData, onNoAutoJoin func()) {
	if createRoomData.autoJoin {
		v.joinRoom(ca, roomID, createRoomData)
		return
	}

	if onNoAutoJoin != nil {
		doInUIThread(onNoAutoJoin)
	}
}
