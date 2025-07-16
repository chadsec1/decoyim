package gui

import (
	"github.com/chadsec1/decoyim/decoylog"
	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/chadsec1/decoyim/session/muc/data"
	"github.com/coyim/gotk3adapter/gtki"
	log "github.com/sirupsen/logrus"
)

// onRoomPositionsView MUST be called from the UI thread
func (v *roomView) onRoomPositionsView() {
	rpv := v.newRoomPositionsView()
	rpv.show()
}

type roomPositionsView struct {
	roomView              *roomView
	banned                muc.RoomOccupantItemList
	none                  muc.RoomOccupantItemList
	onUpdateOccupantLists *callbacksSet

	dialog      gtki.Window `gtk-widget:"positions-window"`
	content     gtki.Box    `gtk-widget:"content"`
	applyButton gtki.Button `gtk-widget:"apply-button"`

	log decoylog.Logger
}

func (v *roomView) newRoomPositionsView() *roomPositionsView {
	rpv := &roomPositionsView{
		roomView:              v,
		onUpdateOccupantLists: newCallbacksSet(),
	}

	rpv.log = v.log.WithFields(log.Fields{
		"room":  v.roomID(),
		"where": "roomPositionsView",
	})

	rpv.initBuilder()
	rpv.initDefaults()

	return rpv
}

func (rpv *roomPositionsView) initBuilder() {
	builder := newBuilder("MUCRoomPositionsDialog")
	panicOnDevError(builder.bindObjects(rpv))

	builder.ConnectSignals(map[string]interface{}{
		"on_apply":  rpv.onApply,
		"on_cancel": rpv.dialog.Destroy,
	})
}

func (rpv *roomPositionsView) initDefaults() {
	rpv.dialog.SetTransientFor(rpv.roomView.mainWindow())
	mucStyles.setRoomConfigPageStyle(rpv.content)
}

// setBanList MUST be called from the UI thread
func (rpv *roomPositionsView) setBanList(list muc.RoomOccupantItemList) {
	rpv.banned = list
}

// updateRemovedOccupantList MUST be called from the UI thread
func (rpv *roomPositionsView) updateRemovedOccupantList(list muc.RoomOccupantItemList) {
	rpv.none = append(rpv.none, list...)
}

func (rpv *roomPositionsView) positionsToUpdate() muc.RoomOccupantItemList {
	occupantsToUpdate := append(muc.RoomOccupantItemList{}, rpv.banned.RetrieveOccupantsToUpdate()...)
	return append(occupantsToUpdate, rpv.none...)
}

// onApply MUST be called from the UI thread
func (rpv *roomPositionsView) onApply() {
	rpv.onUpdateOccupantLists.invokeAll()

	rpv.dialog.Destroy()
	rpv.roomView.loadingViewOverlay.onRoomPositionsUpdate()

	rc, ec := rpv.roomView.account.session.UpdateOccupantAffiliations(rpv.roomView.roomID(), rpv.positionsToUpdate())
	go func() {
		select {
		case <-rc:
			doInUIThread(func() {
				rpv.roomView.notifications.info(roomNotificationOptions{
					message:   i18n.Local("The positions were updated."),
					closeable: true,
				})
				rpv.roomView.loadingViewOverlay.hide()
			})
		case <-ec:
			doInUIThread(func() {
				rpv.roomView.notifications.error(roomNotificationOptions{
					message:   i18n.Local("Unable to update positions."),
					closeable: true,
				})
				rpv.roomView.loadingViewOverlay.hide()
			})
		}
	}()
}

// requestRoomPositions MUST NOT be called from the UI thread
// 	- onOccupantListReceived WILL be called from the UI thread
//	- onNoOccupantList WILL be called from the UI thread
func (rpv *roomPositionsView) requestRoomPositions(onOccupantListReceived func(muc.RoomOccupantItemList), onNoOccupantList func()) {
	rc, ec := rpv.roomView.account.session.GetRoomOccupantsByAffiliation(rpv.roomView.roomID(), &data.OutcastAffiliation{})

	select {
	case ol := <-rc:
		doInUIThread(func() {
			onOccupantListReceived(ol)
		})
	case <-ec:
		doInUIThread(onNoOccupantList)
	}
}

// addPositionComponent MUST be called from the UI thread
func (rpv *roomPositionsView) addPositionComponent(positionComponent hasRoomConfigFormField) {
	rpv.content.Add(positionComponent.fieldWidget())
	positionComponent.refreshContent()
	rpv.onUpdateOccupantLists.add(positionComponent.updateFieldValue)
}

// show MUST be called from the UI thread
func (rpv *roomPositionsView) show() {
	rpv.roomView.loadingViewOverlay.onRoomPositionsRequest()

	onPositionsAvailable := func(list muc.RoomOccupantItemList) {
		rpv.setBanList(list)

		pv := newRoomConfigPositionsWithApplyButton(rpv.applyButton, roomConfigPositionsOptions{
			affiliation:            outcastAffiliation,
			occupantList:           rpv.banned,
			setOccupantList:        rpv.setBanList,
			setRemovedOccupantList: rpv.updateRemovedOccupantList,
			parentWindow:           rpv.dialog,
		})
		rpv.addPositionComponent(pv)

		rpv.roomView.loadingViewOverlay.hide()
		rpv.dialog.Show()
	}

	onNoPositions := func() {
		rpv.roomView.loadingViewOverlay.hide()
		rpv.roomView.notifications.error(roomNotificationOptions{
			message:   i18n.Local("We couldn't get the occupants by affiliation"),
			closeable: true,
		})
	}

	go rpv.requestRoomPositions(onPositionsAvailable, onNoPositions)
}
