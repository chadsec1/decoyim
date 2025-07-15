package gui

import (
	"errors"
	"math/rand"
	"strings"
	"sync"

	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/glibi"
	"github.com/coyim/gotk3adapter/gtki"
)

const (
	mucListRoomsIndexJid = iota
	mucListRoomsIndexName
	mucListRoomsIndexService
	mucListRoomsIndexDescription
	mucListRoomsIndexOccupants
	mucListRoomsIndexRoomInfo
)

type roomListingUpdateData struct {
	iter       gtki.TreeIter
	view       *mucPublicRoomsView
	generation int
}

func (u *gtkUI) updatedRoomListing(rl *muc.RoomListing, data interface{}) {
	d := data.(*roomListingUpdateData)

	// If we get an old update, we don't want to do anything at all
	if d.view.generation == d.generation {
		doInUIThread(func() {
			d.view.fields.description.set(d.iter, g.glib.MarkupEscapeText(rl.Description))
			d.view.fields.occupants.set(d.iter, rl.Occupants)
		})
	}
}

type mucPublicRoomsFields struct {
	jid         *stringStoreField
	name        *stringStoreField
	service     *stringStoreField
	description *stringStoreField
	occupants   *intStoreField
	roomInfo    *intStoreField
}

func createMucPublicRoomsFields(model gtki.TreeStore) *mucPublicRoomsFields {
	fields := &mucPublicRoomsFields{}
	fields.jid = newStringStoreField(model, mucListRoomsIndexJid)
	fields.name = newStringStoreField(model, mucListRoomsIndexName)
	fields.service = newStringStoreField(model, mucListRoomsIndexService)
	fields.description = newStringStoreField(model, mucListRoomsIndexDescription)
	fields.occupants = newIntStoreField(model, mucListRoomsIndexOccupants)
	fields.roomInfo = newIntStoreField(model, mucListRoomsIndexRoomInfo)
	return fields
}

type mucPublicRoomsView struct {
	u              *gtkUI
	builder        *builder
	ac             *connectedAccountsComponent
	currentAccount *account

	generation    int
	updateLock    sync.RWMutex
	serviceGroups map[string]gtki.TreeIter
	roomInfos     map[int]*muc.RoomListing
	cancel        chan bool

	roomsModel gtki.TreeStore
	fields     *mucPublicRoomsFields

	dialog              gtki.Dialog         `gtk-widget:"public-rooms-dialog"`
	roomsTree           gtki.TreeView       `gtk-widget:"public-rooms-tree"`
	rooms               gtki.ScrolledWindow `gtk-widget:"public-rooms-view"`
	customService       gtki.Entry          `gtk-widget:"custom-service-entry"`
	joinButton          gtki.Button         `gtk-widget:"join-room-button"`
	refreshButton       gtki.Button         `gtk-widget:"refresh-button"`
	customServiceButton gtki.Button         `gtk-widget:"list-rooms-button"`

	notificationsArea gtki.Box `gtk-widget:"notifications-area"`
	notifications     *notificationsComponent

	spinnerOverlay gtki.Overlay `gtk-widget:"spinner-overlay"`
	spinnerBox     gtki.Box     `gtk-widget:"spinner-box"`
	spinner        *spinner
}

func newMUCPublicRoomsView(u *gtkUI) *mucPublicRoomsView {
	view := &mucPublicRoomsView{u: u}

	view.initBuilder()
	view.initModel()
	view.initNotificationsAndSpinner(u)
	view.initConnectedAccountsComponent()
	view.initDefaults()

	return view
}

func (prv *mucPublicRoomsView) initBuilder() {
	prv.builder = newBuilder("MUCPublicRoomsDialog")
	panicOnDevError(prv.builder.bindObjects(prv))

	prv.builder.ConnectSignals(map[string]interface{}{
		"on_cancel":            prv.onCancel,
		"on_close_window":      prv.onWindowClose,
		"on_join":              prv.onJoinRoom,
		"on_activate_room_row": prv.onActivateRoomRow,
		"on_selection_changed": prv.onSelectionChanged,
		"on_custom_service":    prv.onUpdatePublicRooms,
		"on_refresh":           prv.onUpdatePublicRooms,
	})
}

func (prv *mucPublicRoomsView) initModel() {
	roomsModel, _ := g.gtk.TreeStoreNew(
		// jid
		glibi.TYPE_STRING,
		// name
		glibi.TYPE_STRING,
		// service
		glibi.TYPE_STRING,
		// description
		glibi.TYPE_STRING,
		// occupants
		glibi.TYPE_INT,
		// room info reference
		glibi.TYPE_INT,
	)

	prv.roomsModel = roomsModel
	prv.roomsTree.SetModel(prv.roomsModel)
	prv.fields = createMucPublicRoomsFields(roomsModel)
}

func (prv *mucPublicRoomsView) initNotificationsAndSpinner(u *gtkUI) {
	prv.notifications = u.newNotificationsComponent()
	prv.notificationsArea.Add(prv.notifications.contentBox())

	prv.spinner = u.newSpinnerComponent()
	s := prv.spinner.spinner()

	// This is a GTK trick to set the size of the spinner,
	// so if the parent has a size of 40x40 for example, with
	// the bellow properties the spinner will be 40x40 too
	s.SetProperty("hexpand", true)
	s.SetProperty("vexpand", true)

	prv.spinnerBox.Add(s)
}

func (prv *mucPublicRoomsView) initConnectedAccountsComponent() {
	accountsInput := prv.builder.get("accounts").(gtki.ComboBox)
	ac := prv.u.createConnectedAccountsComponent(accountsInput, prv.notifications, prv.onAccountsUpdated, prv.onNoAccounts)
	prv.ac = ac
}

func (prv *mucPublicRoomsView) initDefaults() {
	prv.serviceGroups = make(map[string]gtki.TreeIter)
	prv.roomInfos = make(map[int]*muc.RoomListing)
	prv.joinButton.SetSensitive(false)
}

func (prv *mucPublicRoomsView) log() coylog.Logger {
	l := prv.u.hasLog.log
	if prv.currentAccount != nil {
		l = prv.currentAccount.log
	}

	l.WithField("where", "mucPublilcRoomsView")

	return l
}

func (prv *mucPublicRoomsView) onAccountsUpdated(ca *account) {
	if prv.currentAccount == nil || prv.currentAccount.Account() != ca.Account() {
		prv.currentAccount = ca
		prv.updatePublicRoomsForCurrentAccount()
	}
}

func (prv *mucPublicRoomsView) onNoAccounts() {
	prv.currentAccount = nil

	doInUIThread(func() {
		prv.disableRoomsView()
		prv.hideSpinner()

		prv.roomsModel.Clear()
		prv.refreshButton.SetSensitive(false)
		prv.customServiceButton.SetSensitive(false)
	})
}

func (prv *mucPublicRoomsView) onCancel() {
	prv.dialog.Destroy()
}

func (prv *mucPublicRoomsView) onWindowClose() {
	prv.cancelActiveUpdate()
	prv.ac.onDestroy()
}

func (prv *mucPublicRoomsView) cancelActiveUpdate() {
	if prv.cancel == nil {
		return
	}

	prv.cancel <- true
	prv.cancel = nil
}

var (
	errNoPossibleSelection = errors.New("problem getting selection")
	errNoSelection         = errors.New("nothing is selected")
	errNoRoomSelected      = errors.New("a service is selected, not a room, so we can't activate it")
	errNoService           = errors.New("no service is available")
)

func (prv *mucPublicRoomsView) getRoomListingFromIter(iter gtki.TreeIter) (*muc.RoomListing, error) {
	roomInfoRealVal, err := prv.getRoomInfoFromIter(iter)
	if err != nil {
		return nil, err
	}

	rl, ok := prv.roomInfos[roomInfoRealVal]
	if !ok || rl == nil {
		return nil, errNoPossibleSelection
	}
	return rl, nil
}

func (prv *mucPublicRoomsView) getRoomInfoFromIter(iter gtki.TreeIter) (int, error) {
	roomInfoRealVal, e := prv.fields.roomInfo.getWithError(iter)
	if e != nil {
		return 0, errNoRoomSelected
	}

	return roomInfoRealVal, nil
}

func (prv *mucPublicRoomsView) getRoomIDFromIter(iter gtki.TreeIter) (jid.Bare, error) {
	roomName := prv.fields.jid.get(iter)

	_, ok := prv.serviceGroups[roomName]
	if ok {
		return nil, errNoRoomSelected
	}

	service := prv.fields.service.get(iter)
	_, ok = prv.serviceGroups[service]
	if !ok {
		return nil, errNoService
	}

	return jid.NewBare(jid.NewLocal(roomName), jid.NewDomain(service)), nil
}

func (prv *mucPublicRoomsView) getRoomFromIter(iter gtki.TreeIter) (jid.Bare, *muc.RoomListing, error) {
	rl, err := prv.getRoomListingFromIter(iter)
	if err != nil {
		return nil, nil, err
	}

	roomID, err := prv.getRoomIDFromIter(iter)
	if err != nil {
		return nil, nil, err
	}

	return roomID, rl, nil
}

func (prv *mucPublicRoomsView) getSelectedRoom() (jid.Bare, *muc.RoomListing, error) {
	selection, err := prv.roomsTree.GetSelection()
	if err != nil {
		return nil, nil, errNoPossibleSelection
	}

	_, iter, ok := selection.GetSelected()
	if !ok {
		return nil, nil, errNoSelection
	}

	return prv.getRoomFromIter(iter)
}

func (prv *mucPublicRoomsView) onJoinRoom() {
	ident, rl, err := prv.getSelectedRoom()
	if err != nil {
		prv.log().WithError(err).Error("An error occurred when trying to join the room")
		prv.showUserMessageForError(err)
		return
	}

	go prv.joinRoom(ident, rl)
}

func (prv *mucPublicRoomsView) onActivateRoomRow(_ gtki.TreeView, path gtki.TreePath) {
	iter, err := prv.roomsModel.GetIter(path)
	if err != nil {
		prv.log().WithError(err).Error("Couldn't activate the selected item")
		return
	}

	ident, rl, err := prv.getRoomFromIter(iter)
	if err != nil {
		prv.log().WithError(err).Error("Couldn't join to the room based on the current selection")
		prv.showUserMessageForError(err)
		return
	}

	go prv.joinRoom(ident, rl)
}

func (prv *mucPublicRoomsView) onSelectionChanged() {
	_, _, err := prv.getSelectedRoom()
	prv.joinButton.SetSensitive(err == nil)
}

func (prv *mucPublicRoomsView) onUpdatePublicRooms() {
	prv.updatePublicRoomsForCurrentAccount()
}

func (prv *mucPublicRoomsView) updatePublicRoomsForCurrentAccount() {
	if prv.currentAccount != nil {
		go prv.mucUpdatePublicRoomsOn(prv.currentAccount)
	}
}

func (prv *mucPublicRoomsView) getFreshRoomInfoIdentifierAndSet(rl *muc.RoomListing) int {
	for {
		// We are getting a 31 bit integer here to avoid negative numbers
		// We also want to have it be smaller than normal size Golang numbers because
		// this number will be sent out into C, with Glib. For some reason, it did not
		// like negative numbers, and it doesn't work well will 64 bit numbers either
		v := int(rand.Int31())
		_, ok := prv.roomInfos[v]
		if !ok {
			prv.roomInfos[v] = rl
			return v
		}
	}
}

func (prv *mucPublicRoomsView) beforeUpdatingPublicRooms() {
	prv.notifications.clearErrors()
	prv.disableRoomsViewAndShowSpinner()

	prv.roomsModel.Clear()
	prv.refreshButton.SetSensitive(true)
	prv.customServiceButton.SetSensitive(true)
}

func (prv *mucPublicRoomsView) onUpdatePublicRoomsNoResults(customService string) {
	prv.enableRoomsViewAndHideSpinner()
	if customService != "" {
		prv.notifications.error(i18n.Local("That service doesn't seem to exist."))
	} else {
		prv.notifications.error(i18n.Local("Your XMPP server doesn't seem to have any chat room services."))
	}
}

func (prv *mucPublicRoomsView) showSpinner() {
	prv.spinnerOverlay.Show()
	prv.spinner.show()
}

func (prv *mucPublicRoomsView) hideSpinner() {
	prv.spinner.hide()
	prv.spinnerOverlay.Hide()
}

func (prv *mucPublicRoomsView) enableRoomsViewAndHideSpinner() {
	prv.rooms.SetSensitive(true)
	prv.hideSpinner()
}

func (prv *mucPublicRoomsView) disableRoomsViewAndShowSpinner() {
	prv.disableRoomsView()
	prv.showSpinner()
}

func (prv *mucPublicRoomsView) disableRoomsView() {
	prv.rooms.SetSensitive(false)
}

func (prv *mucPublicRoomsView) addNewServiceToModel(roomName, serviceName string) gtki.TreeIter {
	serv := prv.roomsModel.Append(nil)

	prv.serviceGroups[roomName] = serv
	prv.fields.jid.set(serv, g.glib.MarkupEscapeText(roomName))
	prv.fields.name.set(serv, g.glib.MarkupEscapeText(serviceName))

	return serv
}

func (prv *mucPublicRoomsView) addNewRoomToModel(parentIter gtki.TreeIter, rl *muc.RoomListing, gen int) {
	iter := prv.roomsModel.Append(parentIter)

	id := rl.Jid.Local().String()
	name := rl.Name
	if strings.TrimSpace(name) == "" {
		name = id
	}

	prv.fields.jid.set(iter, id)
	prv.fields.name.set(iter, g.glib.MarkupEscapeText(name))
	prv.fields.service.set(iter, rl.Service.String())

	// This will block while finding an unused identifier. However, since
	// we don't expect to get millions of room listings, it's not likely this will ever be a problem.
	roomInfoRef := prv.getFreshRoomInfoIdentifierAndSet(rl)
	prv.fields.roomInfo.set(iter, roomInfoRef)

	rl.OnUpdate(prv.u.updatedRoomListing, &roomListingUpdateData{iter, prv, gen})

	prv.roomsTree.ExpandAll()
}

func (prv *mucPublicRoomsView) handleReceivedServiceListing(sl *muc.ServiceListing) {
	_, ok := prv.serviceGroups[sl.Jid.String()]
	if !ok {
		doInUIThread(func() {
			prv.addNewServiceToModel(sl.Jid.String(), sl.Name)
		})
	}
}

func (prv *mucPublicRoomsView) handleReceivedRoomListing(rl *muc.RoomListing, gen int) {
	serv, ok := prv.serviceGroups[rl.Service.String()]
	doInUIThread(func() {
		if !ok {
			serv = prv.addNewServiceToModel(rl.Service.String(), rl.ServiceName)
		}

		prv.addNewRoomToModel(serv, rl, gen)
	})
}

func (prv *mucPublicRoomsView) handleReceivedError(err error) {
	prv.log().WithError(err).Error("Something went wrong when trying to get chat rooms")
	doInUIThread(func() {
		prv.notifications.error(i18n.Local("Something went wrong when trying to get chat rooms."))
	})
}

func (prv *mucPublicRoomsView) listenPublicRoomsResponse(gen int, res <-chan *muc.RoomListing, resServices <-chan *muc.ServiceListing, ec <-chan error) bool {
	select {
	case sl, ok := <-resServices:
		if !ok {
			return false
		}

		prv.handleReceivedServiceListing(sl)
	case rl, ok := <-res:
		if !ok || rl == nil {
			return false
		}

		prv.handleReceivedRoomListing(rl, gen)
	case err, ok := <-ec:
		if !ok {
			return false
		}
		if err != nil {
			prv.handleReceivedError(err)
		}
		return false
	case <-prv.cancel:
		return false
	}
	return true
}

func (prv *mucPublicRoomsView) listenPublicRoomsUpdate(customService string, gen int, res <-chan *muc.RoomListing, resServices <-chan *muc.ServiceListing, ec <-chan error) {
	hasSomething := false

	defer func() {
		if !hasSomething {
			doInUIThread(func() {
				prv.onUpdatePublicRoomsNoResults(customService)
			})
		}

		prv.updateLock.Unlock()
	}()

	for prv.listenPublicRoomsResponse(gen, res, resServices, ec) {
		if !hasSomething {
			hasSomething = true
			doInUIThread(prv.enableRoomsViewAndHideSpinner)
		}
	}
}

// mucUpdatePublicRoomsOn MUST NOT be called from the UI thread
func (prv *mucPublicRoomsView) mucUpdatePublicRoomsOn(a *account) {
	prv.cancelActiveUpdate()

	prv.updateLock.Lock()
	prv.cancel = make(chan bool, 1)

	doInUIThread(prv.beforeUpdatingPublicRooms)

	prv.generation++
	prv.serviceGroups = make(map[string]gtki.TreeIter)
	prv.roomInfos = make(map[int]*muc.RoomListing)

	// We save the generation value here, in case it gets modified inside the view later
	gen := prv.generation

	customService, _ := prv.customService.GetText()

	res, resServices, ec := a.session.GetRooms(jid.Parse(a.Account()).Host(), customService)
	go prv.listenPublicRoomsUpdate(customService, gen, res, resServices, ec)
}

func (prv *mucPublicRoomsView) showUserMessageForError(err error) {
	userMessage := i18n.Local("An unknown error occurred, please try again.")

	switch err {
	case errNoPossibleSelection:
		userMessage = i18n.Local("We can't determine what has been selected, please try again.")
	case errNoRoomSelected:
		userMessage = i18n.Local("The selected item is not a room, select one room from the list to join to.")
	case errNoSelection:
		userMessage = i18n.Local("Please, select one room from the list to join to.")
	case errNoService:
		userMessage = i18n.Local("We can't determine which service has been selected, please try again.")
	}

	prv.notifications.error(userMessage)
}

// mucShowPublicRooms MUST be called from the UI thread
func (u *gtkUI) mucShowPublicRooms() {
	view := newMUCPublicRoomsView(u)

	u.connectShortcutsChildWindow(view.dialog)

	view.dialog.SetTransientFor(u.window)
	view.dialog.Show()
}

// joinRoom MUST NOT be called from the UI thread
func (prv *mucPublicRoomsView) joinRoom(roomJid jid.Bare, roomInfo *muc.RoomListing) {
	if prv.currentAccount == nil {
		prv.log().WithField("room", roomJid).Debug("joinRoom(): no account is selected")
		doInUIThread(func() {
			prv.notifications.error(i18n.Local("No account was selected, please select one account from the list."))
		})
		return
	}

	prv.log().WithField("room", roomJid).Debug("joinRoom()")
	doInUIThread(prv.dialog.Destroy)
	prv.u.joinRoom(prv.currentAccount, roomJid, nil)
}
